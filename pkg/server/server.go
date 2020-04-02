package server

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/moolen/juno/pkg/ipcache"
	pb "github.com/moolen/juno/proto"
	sg "github.com/moolen/statusgraph/pkg/store"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/client-go/kubernetes"
)

type Observer struct {
	listener net.Listener
	server   *grpc.Server
	gw       *TraceProviderClient
	ipcache  *ipcache.State
}

func New(client *kubernetes.Clientset, target string, port int, syncInterval time.Duration, bufferSize int) (*Observer, error) {
	gw, err := NewGateway(target)
	if err != nil {
		return nil, err
	}
	ipcache := ipcache.New(client, syncInterval, bufferSize)
	ipcache.Run()
	server := &Observer{
		gw:      gw,
		ipcache: ipcache,
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	server.listener = listener
	grpcServer := grpc.NewServer(
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	)
	server.server = grpcServer
	return server, nil
}

func (o *Observer) fetchTraces() {
	log.Infof("fetch traces: make client call")
	cl, err := o.gw.client.GetTraces(context.Background(), &pb.GetTracesRequest{})
	if err != nil {
		log.Error(err)
		return
	}

	log.Infof("run recv loop")
	//NewGraph()

	for {
		trace, err := cl.Recv()
		if err != nil {
			log.Error(err)
			return
		}

		// add metadata
		srcEP, err := o.ipcache.GetEndpointByIP(trace.Trace.IP.GetSource())
		if err != nil {
			log.Debugf("could not find endpoint for src: %s", trace.Trace.IP.GetSource())
			continue
		}
		dstEP, err := o.ipcache.GetEndpointByIP(trace.Trace.IP.GetDestination())
		if err != nil {
			log.Debugf("could not find endpoint for dst: %s", trace.Trace.IP.GetDestination())
			continue
		}
		log.Infof("srcEP %v, dstEP %v", srcEP, dstEP)
		// build service id
		// srcID, dstID, err := buildID(trace.Trace, srcEP, dstEP)
		// if err != nil {
		// 	log.Debug(err)
		// 	continue
		// }

		// add to graph
		// var srcNode, dstNode *Node
		// srcNode = g.FindNode(srcID)
		// if srcNode == nil {
		// 	srcNode = &Node{
		// 		ServiceID: srcID,
		// 	}
		// 	g.AddNode(srcNode)
		// }
		// dstNode = g.FindNode(dstID)
		// if dstNode == nil {
		// 	dstNode = &Node{
		// 		ServiceID: dstID,
		// 	}
		// 	g.AddNode(dstNode)
		// }
		// g.EnsureEdge(srcNode, dstNode)

		// // this is for debugging RN
		// data, err := g.JSONGraph()
		// g.WriteDotGraph("graph.svg")
		// if err != nil {
		// 	log.Error(err)
		// }
		// err = ioutil.WriteFile("graph.json", data, os.ModePerm)
		// if err != nil {
		// 	log.Error(err)
		// }

	}
}

func buildID(t *pb.Trace, srcEP, dstEP *ipcache.Endpoint) (*sg.Node, *sg.Node, error) {
	src := &sg.Node{}
	dst := &sg.Node{}
	tcp := t.GetL4().GetTCP()
	udp := t.GetL4().GetUDP()

	var dport, sport uint32
	if tcp != nil {
		dport = tcp.GetDestinationPort()
		sport = tcp.GetSourcePort()
	} else if udp != nil {
		dport = udp.GetDestinationPort()
		sport = udp.GetSourcePort()
	}
	deph := isEphemeralPort(dport)
	seph := isEphemeralPort(sport)
	// skip invalid l4 & ephemere connections
	if dport == 0 || sport == 0 {
		return nil, nil, fmt.Errorf("missing L4 proto")
	}

	dst.Name = getIdentity(t.IP.Destination, dstEP)
	src.Name = getIdentity(t.IP.Source, srcEP)
	if deph && seph {
		if !portMatchesEndpoint(dport, dstEP) && !portMatchesEndpoint(sport, srcEP) {
			return nil, nil, fmt.Errorf("ephemere connection: %s:%d -> %s:%d (%#v | %#v)", t.IP.Source, sport, t.IP.Destination, dport, srcEP, dstEP)
		}
		if portMatchesEndpoint(dport, dstEP) {
			dst.Connector = []sg.NodeConnector{
				{
					Name:  dstEP.Ports[0].Name,
					Label: strconv.Itoa(int(dstEP.Ports[0].Port)),
				},
			}
			return src, dst, nil
		}
		if portMatchesEndpoint(sport, srcEP) {
			src.Connector = []sg.NodeConnector{
				{
					Name:  srcEP.Ports[0].Name,
					Label: strconv.Itoa(int(srcEP.Ports[0].Port)),
				},
			}
			return dst, src, nil
		}
	}

	if !seph {
		src.Connector = []sg.NodeConnector{
			{
				Name:  srcEP.Ports[0].Name,
				Label: strconv.Itoa(int(srcEP.Ports[0].Port)),
			},
		}
	}

	if !deph {
		dst.Connector = []sg.NodeConnector{
			{
				Name:  dstEP.Ports[0].Name,
				Label: strconv.Itoa(int(dstEP.Ports[0].Port)),
			},
		}
	}

	// switch directions if dport is ephemeral
	if isEphemeralPort(dport) && !portMatchesEndpoint(dport, dstEP) {
		return dst, src, nil
	}
	return src, dst, nil
}

func portMatchesEndpoint(port uint32, ep *ipcache.Endpoint) bool {
	for _, p := range ep.Ports {
		if p.Port == port {
			return true
		}
	}
	return false
}

func isEphemeralPort(port uint32) bool {
	if port > 32768 {
		return true
	}
	return false
}

func getIdentity(addr string, t *ipcache.Endpoint) string {
	ip := net.ParseIP(addr)
	if isPublicIP(ip) {
		return "www"
	}
	for _, l := range []string{"app", "k8s-app"} {
		if t.Labels[l] != "" {
			return t.Labels[l]
		}
	}
	return t.Name
}

func isPublicIP(IP net.IP) bool {
	if IP.IsLoopback() || IP.IsLinkLocalMulticast() || IP.IsLinkLocalUnicast() {
		return false
	}
	if ip4 := IP.To4(); ip4 != nil {
		switch {
		case ip4[0] == 10:
			return false
		case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
			return false
		case ip4[0] == 192 && ip4[1] == 168:
			return false
		default:
			return true
		}
	}
	return false
}

func (srv *Observer) Serve(ctx context.Context) {
	log.Infof("serve")
	go srv.fetchTraces()
	log.Fatal(srv.server.Serve(srv.listener))

}

func (srv *Observer) Stop() {
	log.Infof("stop")
	srv.server.GracefulStop()
}
