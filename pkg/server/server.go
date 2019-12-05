package server

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"net"
	"strconv"

	lru "github.com/hashicorp/golang-lru"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type Observer struct {
	listener net.Listener
	server   *grpc.Server
	gw       *TraceProviderClient
	cache    *lru.ARCCache
}

func New(target string, port int) (*Observer, error) {
	cache, err := lru.NewARC(1024)
	if err != nil {
		return nil, err
	}
	gw, err := NewGateway(target)
	if err != nil {
		return nil, err
	}
	server := &Observer{
		gw:    gw,
		cache: cache,
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
		log.Infof("received trace: %+v", trace)

		id, err := id(trace.Trace)
		if err != nil {
			log.Error(err)
			continue
		}
		o.cache.Add(id, trace.Trace)
	}
}

func id(t *pb.Trace) (string, error) {
	tcp := t.L4.GetTCP()
	udp := t.L4.GetUDP()
	h := md5.New()
	b := bytes.Buffer{}
	b.WriteString(t.IP.Source)
	b.WriteString(t.IP.Destination)

	// TODO: we need to figure out the direction of the trace
	// to ignore ephemeral ports
	// check ephemeral ports: https://en.wikipedia.org/wiki/Ephemeral_port#cite_note-3
	if tcp != nil {
		if tcp.DestinationPort < 32768 {
			b.WriteString(strconv.Itoa(int(tcp.DestinationPort)))
		}
		if tcp.SourcePort < 32768 {
			b.WriteString(strconv.Itoa(int(tcp.SourcePort)))
		}
	} else if udp != nil {
		if udp.DestinationPort < 32768 {
			b.WriteString(strconv.Itoa(int(udp.DestinationPort)))
		}
		if udp.SourcePort < 32768 {
			b.WriteString(strconv.Itoa(int(udp.SourcePort)))
		}
	}

	_, err := h.Write(b.Bytes())
	if err != nil {
		return "", err
	}
	var outBuf bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &outBuf)
	defer encoder.Close()
	_, err = encoder.Write(h.Sum(nil))
	if err != nil {
		return "", nil
	}
	return outBuf.String(), nil
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
