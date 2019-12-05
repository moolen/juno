package controller

import (
	"context"

	"github.com/moolen/juno/pkg/ring"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
)

// GetTraces ..
func (o *TraceServer) GetTraces(req *pb.GetTracesRequest, gfs pb.Tracer_GetTracesServer) error {
	rr := ring.NewRingReader(o.ring, 0)

	for {
		select {
		case <-gfs.Context().Done():
			return gfs.Context().Err()
		default:
		}

		t := rr.NextFollow(gfs.Context())
		err := gfs.Send(&pb.GetTracesResponse{
			Trace: t,
		})
		if err != nil {
			return err
		}
	}
}

// ServerStatus returns some details
func (o *TraceServer) ServerStatus(context.Context, *pb.ServerStatusRequest) (*pb.ServerStatusResponse, error) {
	log.Infof("send status")
	res := &pb.ServerStatusResponse{
		MaxFlows: o.ring.Cap(),
		NumFlows: o.ring.Len(),
	}
	return res, nil
}
