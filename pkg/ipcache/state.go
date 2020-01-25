package ipcache

import (
	"context"
	"fmt"
	"time"

	pb "github.com/moolen/juno/proto"

	"github.com/moolen/juno/pkg/k8s"
	"k8s.io/client-go/kubernetes"
)

type Entry struct {
	Name      string
	Namespace string
	Labels    map[string]string
}

type State struct {
	client    *kubernetes.Clientset
	endpoints *k8s.EndpointCache
	pods      *k8s.PodCache
	services  *k8s.ServiceCache
	nodes     *k8s.NodeCache
}

func New(client *kubernetes.Clientset, syncInterval time.Duration, bufferSize int) *State {
	endpoints := k8s.NewEndpointCache(k8s.NewListWatch(client, "endpoints"), syncInterval, bufferSize)
	services := k8s.NewServiceCache(k8s.NewListWatch(client, "services"), syncInterval, bufferSize)
	pods := k8s.NewPodCache(k8s.NewListWatch(client, "pods"), syncInterval, bufferSize)
	nodes := k8s.NewNodeCache(k8s.NewListWatch(client, "nodes"), syncInterval, bufferSize)
	return &State{
		client:    client,
		endpoints: endpoints,
		pods:      pods,
		services:  services,
		nodes:     nodes,
	}
}

var ErrNotFound = fmt.Errorf("ip addr not found")

func (s *State) GetEndpointByIP(ip string) (*pb.Endpoint, error) {
	e := pb.Endpoint{}
	ep, err := s.endpoints.GetByIP(ip)
	if err == nil {
		e.Name = ep.ObjectMeta.Name
		e.Namespace = ep.ObjectMeta.Namespace
		e.Labels, _ = s.endpoints.GetMetadataByIP(ip)
	}
	po, err := s.pods.GetByIP(ip)
	if err == nil {
		e.Name = po.ObjectMeta.Name
		e.Namespace = po.ObjectMeta.Namespace
		e.Labels, _ = s.pods.GetMetadataByIP(ip)
	}
	svc, err := s.services.GetByIP(ip)
	if err == nil {
		e.Name = svc.ObjectMeta.Name
		e.Namespace = svc.ObjectMeta.Namespace
		e.Labels, _ = s.services.GetMetadataByIP(ip)
	}
	node, err := s.nodes.GetByIP(ip)
	if err == nil {
		e.Name = node.ObjectMeta.Name
		e.Namespace = node.ObjectMeta.Namespace
		e.Labels, _ = s.nodes.GetMetadataByIP(ip)
	}

	return &e, nil
}

func (s *State) Run() {
	s.endpoints.Run(context.Background())
	s.services.Run(context.Background())
	s.pods.Run(context.Background())
	s.nodes.Run(context.Background())
}
