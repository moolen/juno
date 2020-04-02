package controller

import (
	"context"
	"fmt"
	"net"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/moolen/juno/pkg/ring"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// TraceServer ..
type TraceServer struct {
	listener   net.Listener
	server     *grpc.Server
	ring       *ring.Ring
}

// NewTraceServer ..
func NewTraceServer(ring *ring.Ring) (*TraceServer, error) {

	ts := &TraceServer{
		ring: ring,
		server: grpc.NewServer(
			grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
			grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
		),
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", 3000))
	if err != nil {
		return nil, err
	}
	ts.listener = listener
	pb.RegisterTracerServer(ts.server, ts)
	return ts, nil
}

// Serve ..
func (srv *TraceServer) Serve(ctx context.Context) {
	log.Infof("serve")
	log.Infof("grpc listening on :%d", 3000)
	srv.server.Serve(srv.listener)
}

// Stop ..
func (srv *TraceServer) Stop() {
	log.Infof("stop")
	srv.server.GracefulStop()
}
