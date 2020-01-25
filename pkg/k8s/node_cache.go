package k8s

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type NodeCache struct {
	nodes      chan *v1.Node
	indexer    cache.Indexer
	controller cache.Controller
}

func NewNodeCache(source cache.ListerWatcher, syncInterval time.Duration, bufferSize int) *NodeCache {
	nodes := make(chan *v1.Node, bufferSize)
	nodeHandler := &nodeHandler{nodes}
	indexer, controller := cache.NewIndexerInformer(source, &v1.Node{}, syncInterval, nodeHandler, cache.Indexers{
		indexByIP: nodeIPIndex,
	})
	NodeCache := &NodeCache{
		nodes:      nodes,
		indexer:    indexer,
		controller: controller,
	}
	return NodeCache
}

func (s *NodeCache) getMetadataForIP(ip string) (map[string]string, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("node for %s not found", ip)
	}
	for _, obj := range items {
		node := obj.(*v1.Node)
		return node.ObjectMeta.Labels, nil
	}
	return nil, fmt.Errorf("multiple nodes found")
}

// GetMetadataByIP returns the node metadata
func (s *NodeCache) GetMetadataByIP(ip string) (map[string]string, error) {
	return s.getMetadataForIP(ip)
}

func (s *NodeCache) getByIP(ip string) (*v1.Node, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("node for %s not found", ip)
	}
	for _, obj := range items {
		node := obj.(*v1.Node)
		return node, nil
	}
	return nil, fmt.Errorf("multiple nodes found")
}

// GetByIP returns the node
func (s *NodeCache) GetByIP(ip string) (*v1.Node, error) {
	return s.getByIP(ip)
}

func nodeIPIndex(obj interface{}) ([]string, error) {
	node := obj.(*v1.Node)
	var out []string
	for _, addr := range node.Status.Addresses {
		out = append(out, addr.Address)
	}

	return out, nil
}

// Run starts the controller processing updates. Blocks until the cache has synced
func (s *NodeCache) Run(ctx context.Context) error {
	go s.controller.Run(ctx.Done())
	log.Infof("started cache controller")
	ok := cache.WaitForCacheSync(ctx.Done(), s.controller.HasSynced)
	if !ok {
		return fmt.Errorf("error waiting for sync")
	}
	return nil
}

type nodeHandler struct {
	node chan<- *v1.Node
}

func (o *nodeHandler) announce(node *v1.Node) {
	logger := log.WithFields(NodeFields(node))
	select {
	case o.node <- node:
		logger.Debugf("announced node")
	default:
		logger.Warnf("node announcement full, dropping")
	}
}

func (o *nodeHandler) OnAdd(obj interface{}) {
	node, isNode := obj.(*v1.Node)
	if !isNode {
		log.Errorf("OnAdd unexpected object: %+v", obj)
		return
	}
	log.WithFields(NodeFields(node)).Debugf("added node")
	o.announce(node)
}

func (o *nodeHandler) OnDelete(obj interface{}) {
	node, isNode := obj.(*v1.Node)
	if !isNode {
		deletedObj, isDeleted := obj.(cache.DeletedFinalStateUnknown)
		if !isDeleted {
			log.Errorf("OnDelete unexpected object: %+v", obj)
			return
		}
		node, isNode = deletedObj.Obj.(*v1.Node)
		if !isNode {
			log.Errorf("OnDelete unexpected DeletedFinalStateUnknown object: %+v", deletedObj.Obj)
		}
		log.WithFields(NodeFields(node)).Debugf("deleted node")
		return
	}
	log.WithFields(NodeFields(node)).Debugf("deleted node")
	return
}

func (o *nodeHandler) OnUpdate(old, new interface{}) {
	node, isNode := new.(*v1.Node)
	if !isNode {
		log.Errorf("OnUpdate unexpected object: %+v", new)
		return
	}
	log.WithFields(NodeFields(node)).Debugf("updated node")
}

func NodeFields(node *v1.Node) log.Fields {
	return log.Fields{
		"node.namespace":                  node.ObjectMeta.Namespace,
		"node.name":                       node.ObjectMeta.Name,
		"node.status.Addresses.0.address": node.Status.Addresses[0].Address,
		"resource.version":                node.ObjectMeta.ResourceVersion,
		"generation.metadata":             node.ObjectMeta.Generation,
	}
}
