package ipcache

import (
	"context"
	"fmt"
	"time"

	"github.com/moolen/juno/pkg/k8s"
	"k8s.io/client-go/kubernetes"
)

type Endpoint struct {
	Name      string
	Namespace string
	Labels    map[string]string
	Ports     []Port
}

type Port struct {
	Name     string
	Port     uint32
	Protocol string
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

func (s *State) GetEndpointByIP(ip string) (*Endpoint, error) {
	ep, err := s.endpoints.GetByIP(ip)
	if err == nil {
		meta, err := s.endpoints.GetMetadataByIP(ip)
		e := &Endpoint{
			Name:      ep.ObjectMeta.Name,
			Namespace: ep.ObjectMeta.Namespace,
			Labels:    meta,
		}
		for _, sub := range ep.Subsets {
			for _, p := range sub.Ports {
				e.Ports = append(e.Ports, Port{
					Name:     p.Name,
					Port:     uint32(p.Port),
					Protocol: string(p.Protocol),
				})
			}
		}
		return e, err
	}
	po, err := s.pods.GetByIP(ip)
	if err == nil {
		meta, err := s.pods.GetMetadataByIP(ip)
		e := &Endpoint{
			Name:      po.ObjectMeta.Name,
			Namespace: po.ObjectMeta.Namespace,
			Labels:    meta,
		}
		e.Name = po.ObjectMeta.Name
		e.Namespace = po.ObjectMeta.Namespace
		for _, c := range po.Spec.Containers {
			for _, p := range c.Ports {
				e.Ports = append(e.Ports, Port{
					Name:     p.Name,
					Port:     uint32(p.ContainerPort),
					Protocol: string(p.Protocol),
				})
			}
		}
		return e, err
	}
	svc, err := s.services.GetByIP(ip)
	if err == nil {
		meta, err := s.services.GetMetadataByIP(ip)
		e := &Endpoint{
			Name:      svc.ObjectMeta.Name,
			Namespace: svc.ObjectMeta.Namespace,
			Labels:    meta,
		}
		for _, p := range svc.Spec.Ports {
			e.Ports = append(e.Ports, Port{
				Name:     p.Name,
				Port:     uint32(p.Port),
				Protocol: string(p.Protocol),
			})
		}
		return e, err
	}
	node, err := s.nodes.GetByIP(ip)
	if err == nil {
		meta, err := s.nodes.GetMetadataByIP(ip)
		return &Endpoint{
			Name:      node.ObjectMeta.Name,
			Namespace: node.ObjectMeta.Namespace,
			Labels:    meta,
		}, err
	}

	return nil, ErrNotFound
}

func (s *State) Run() {
	s.endpoints.Run(context.Background())
	s.services.Run(context.Background())
	s.pods.Run(context.Background())
	s.nodes.Run(context.Background())
}
