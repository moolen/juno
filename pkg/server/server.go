package server

import (
	"context"
	"fmt"
	"net"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/moolen/juno/pkg/ipcache"
	pb "github.com/moolen/juno/proto"
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
	for {
		trace, err := cl.Recv()
		if err != nil {
			log.Error(err)
			return
		}

		// add metadata
		trace.Trace.Source, err = o.ipcache.GetEndpointByIP(trace.Trace.IP.GetSource())
		if err != nil {
			log.Debugf("could not find endpoint for src: ", trace.Trace.IP.GetSource())
			continue
		}
		trace.Trace.Destination, err = o.ipcache.GetEndpointByIP(trace.Trace.IP.GetDestination())
		if err != nil {
			log.Debugf("could not find endpoint for dst: ", trace.Trace.IP.GetDestination())
			continue
		}

		log.Infof("received trace: %+v", trace)

		//TODO: build graph

	}
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
