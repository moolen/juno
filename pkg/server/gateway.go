package server

import (
	"context"
	"fmt"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

// TraceProviderClient ..
type TraceProviderClient struct {
	conn   *grpc.ClientConn
	client pb.TracerClient
}

const (
	// RetryInterval ..
	RetryInterval = 10 * time.Millisecond
)

// NewGateway ..
func NewGateway(address string) (*TraceProviderClient, error) {
	log.Infof("creating grpc gateway: %s", address)
	callOpts := []retry.CallOption{
		retry.WithBackoff(retry.BackoffLinear(RetryInterval)),
	}
	dialOpts := []grpc.DialOption{
		// TODO: implement proper credentials handling
		grpc.WithInsecure(),

		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(retry.UnaryClientInterceptor(callOpts...), grpc_prometheus.UnaryClientInterceptor)),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithDisableServiceConfig(),
		grpc.WithBlock(),
		grpc.WithStreamInterceptor(grpc_prometheus.StreamClientInterceptor),
	}
	conn, err := grpc.DialContext(context.TODO(), address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("error dialing grpc server: %v", err)
	}

	client := pb.NewTracerClient(conn)
	return &TraceProviderClient{
		conn,
		client,
	}, nil
}
