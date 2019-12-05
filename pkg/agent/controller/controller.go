package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/moolen/juno/pkg/k8s"
	"github.com/moolen/juno/pkg/ring"
	"github.com/moolen/juno/pkg/tracer"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// Controller ...
type Controller struct {
	Client        *kubernetes.Clientset
	Tracer        *tracer.Tracer
	nodeName      string
	apiserverAddr string
	ring          *ring.Ring
	srv           *TraceServer
	ep            *k8s.EndpointCache

	// this bool signals shutdown
	stop bool
}

type EndpointMetadata struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

// New ...
func New(
	client *kubernetes.Clientset,
	ifacePrefix, nodeName string, apiserverAddr string,
	syncInterval time.Duration,
	perfPollInterval time.Duration,
	listenPort, bufferSize int) (*Controller, error) {
	ring := ring.NewRing(2048)
	t, err := tracer.NewTracer(ifacePrefix, perfPollInterval)
	if err != nil {
		return nil, err
	}
	srv, err := NewTraceServer(listenPort, ring)
	if err != nil {
		return nil, err
	}
	ep := k8s.NewEndpointCache(k8s.NewListWatch(client, "endpoints"), syncInterval, bufferSize)
	ep.Run(context.Background())

	return &Controller{
		Client:        client,
		Tracer:        t,
		nodeName:      nodeName,
		apiserverAddr: apiserverAddr,
		ring:          ring,
		srv:           srv,
		ep:            ep,
	}, nil
}

func (c *Controller) pollEvents() {
	log.Infof("start polling perfMap events")
	for {
		select {

		// trace events should be enriched with metadata:
		// * who is the origin/destination pod or service?
		//   -> including metadata like namespace and labels
		//
		// TODO: We can not maintain a full list of pods/services (it simply does not scale)
		//       For the sake of simplicity we will do this right now
		case trace := <-c.Tracer.Read():
			err := c.process(&trace)
			if err != nil {
				log.Errorf("error processing trace: %s", err)
			}
		default:
			if c.stop {
				return
			}
		}
	}
}

func (c *Controller) process(trace *pb.Trace) error {
	srcEndpoint, _ := c.ep.GetEndpointByIP(trace.IP.Source)
	srcRef, _ := getTargetReference(srcEndpoint, trace.IP.Source)
	dstEndpoint, _ := c.ep.GetEndpointByIP(trace.IP.Destination)
	dstRef, _ := getTargetReference(dstEndpoint, trace.IP.Destination)
	// add metadata if found
	if srcEndpoint != nil && dstEndpoint != nil {
		trace.Destination = &pb.Endpoint{
			Namespace: dstRef.Namespace,
			Name:      dstRef.Name,
			Labels:    labelList(dstEndpoint),
		}
		trace.Source = &pb.Endpoint{
			Namespace: srcRef.Namespace,
			Name:      srcRef.Name,
			Labels:    labelList(srcEndpoint),
		}
	}
	c.ring.Write(trace)
	return nil
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

func labelList(ep *corev1.Endpoints) []string {
	list := []string{
		fmt.Sprintf("k8s:io.kubernetes.pod.name=%s", ep.Name),
		fmt.Sprintf("k8s:io.kubernetes.pod.namespace=%s", ep.Namespace),
	}
	for k, v := range ep.ObjectMeta.Labels {
		list = append(list, fmt.Sprintf("k8s:%s=%v", k, v))
	}
	return list
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
