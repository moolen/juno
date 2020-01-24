package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/moolen/juno/pkg/ring"
	"github.com/moolen/juno/pkg/tracer"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Controller ...
type Controller struct {
	Client   *kubernetes.Clientset
	Tracer   *tracer.Tracer
	nodeName string
	ring     *ring.Ring
	srv      *TraceServer

	// this bool signals shutdown
	stop bool
}

// New ...
func New(
	client *kubernetes.Clientset,
	ifacePrefix, nodeName string,
	syncInterval time.Duration,
	perfPollInterval time.Duration,
	listenPort int) (*Controller, error) {
	ring := ring.NewRing(2048)
	t, err := tracer.NewTracer(ifacePrefix, perfPollInterval, syncInterval)
	if err != nil {
		return nil, err
	}
	srv, err := NewTraceServer(listenPort, ring)
	if err != nil {
		return nil, err
	}

	return &Controller{
		Client:   client,
		Tracer:   t,
		nodeName: nodeName,
		ring:     ring,
		srv:      srv,
	}, nil
}

func (c *Controller) pollEvents() {
	log.Infof("start polling perfMap events")
	for {
		select {
		case trace := <-c.Tracer.Read():
			trace.NodeName = c.nodeName
			c.ring.Write(&trace)
		default:
			if c.stop {
				return
			}
		}
	}
}

var errRefNotFound = fmt.Errorf("targetRef not found")

func getTargetReference(ep *corev1.Endpoints, targetAddr string) (*corev1.ObjectReference, error) {
	if ep == nil {
		return nil, nil
	}
	for _, s := range ep.Subsets {
		for _, addr := range s.Addresses {
			if addr.IP == targetAddr {
				return addr.TargetRef, nil
			}
		}
	}
	return nil, errRefNotFound
}

// Start ..
func (c *Controller) Start() {
	log.Debugf("starting controller")
	go c.pollEvents()
	go c.srv.Serve(context.Background())
	c.Tracer.Start()
}

// Stop ..
func (c *Controller) Stop() {
	c.Tracer.Stop()
	c.stop = true
}
