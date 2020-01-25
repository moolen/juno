package k8s

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type EndpointCache struct {
	endpoints  chan *v1.Endpoints
	indexer    cache.Indexer
	controller cache.Controller
}

func NewEndpointCache(source cache.ListerWatcher, syncInterval time.Duration, bufferSize int) *EndpointCache {
	endpoints := make(chan *v1.Endpoints, bufferSize)
	endpointHandler := &endpointHandler{endpoints}
	indexer, controller := cache.NewIndexerInformer(source, &v1.Endpoints{}, syncInterval, endpointHandler, cache.Indexers{
		indexByIP: endpointIPIndex,
	})
	EndpointCache := &EndpointCache{
		endpoints:  endpoints,
		indexer:    indexer,
		controller: controller,
	}
	return EndpointCache
}

func (s *EndpointCache) getMetadataForIP(ip string) (map[string]string, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("endpoint for %s not found", ip)
	}
	for _, obj := range items {
		ep := obj.(*v1.Endpoints)
		return ep.ObjectMeta.Labels, nil
	}
	return nil, fmt.Errorf("multiple endpoints found")
}

// GetMetadataByIP returns the Endpoint metadata
func (s *EndpointCache) GetMetadataByIP(ip string) (map[string]string, error) {
	return s.getMetadataForIP(ip)
}

func (s *EndpointCache) getByIP(ip string) (*v1.Endpoints, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("endpoint for %s not found", ip)
	}
	for _, obj := range items {
		ep := obj.(*v1.Endpoints)
		return ep, nil
	}
	return nil, fmt.Errorf("multiple endpoints found")
}

// GetByIP returns the Endpoint
func (s *EndpointCache) GetByIP(ip string) (*v1.Endpoints, error) {
	return s.getByIP(ip)
}

const (
	indexByIP = "byIP"
)

func endpointIPIndex(obj interface{}) ([]string, error) {
	endpoints := obj.(*v1.Endpoints)
	var out []string
	for _, sub := range endpoints.Subsets {
		for _, addr := range sub.Addresses {
			out = append(out, addr.IP)
		}
	}
	return out, nil
}

// Run starts the controller processing updates. Blocks until the cache has synced
func (s *EndpointCache) Run(ctx context.Context) error {
	go s.controller.Run(ctx.Done())
	log.Infof("started cache controller")
	ok := cache.WaitForCacheSync(ctx.Done(), s.controller.HasSynced)
	if !ok {
		return fmt.Errorf("error waiting for sync")
	}
	return nil
}

type endpointHandler struct {
	endpoints chan<- *v1.Endpoints
}

func (o *endpointHandler) announce(endpoint *v1.Endpoints) {
	logger := log.WithFields(EndpointFields(endpoint))
	select {
	case o.endpoints <- endpoint:
		logger.Debugf("announced endpoint")
	default:
		logger.Warnf("endpoint announcement full, dropping")
	}
}

func (o *endpointHandler) OnAdd(obj interface{}) {
	ep, isEp := obj.(*v1.Endpoints)
	if !isEp {
		log.Errorf("OnAdd unexpected object: %+v", obj)
		return
	}
	log.WithFields(EndpointFields(ep)).Debugf("added ep")
	o.announce(ep)
}

func (o *endpointHandler) OnDelete(obj interface{}) {
	ep, isEp := obj.(*v1.Endpoints)
	if !isEp {
		deletedObj, isDeleted := obj.(cache.DeletedFinalStateUnknown)
		if !isDeleted {
			log.Errorf("OnDelete unexpected object: %+v", obj)
			return
		}
		ep, isEp = deletedObj.Obj.(*v1.Endpoints)
		if !isEp {
			log.Errorf("OnDelete unexpected DeletedFinalStateUnknown object: %+v", deletedObj.Obj)
		}
		log.WithFields(EndpointFields(ep)).Debugf("deleted ep")
		return
	}
	log.WithFields(EndpointFields(ep)).Debugf("deleted ep")
	return
}

func (o *endpointHandler) OnUpdate(old, new interface{}) {
	ep, isEp := new.(*v1.Endpoints)
	if !isEp {
		log.Errorf("OnUpdate unexpected object: %+v", new)
		return
	}
	log.WithFields(EndpointFields(ep)).Debugf("updated ep")
}

func EndpointFields(endpoints *v1.Endpoints) log.Fields {
	return log.Fields{
		"endpoints.namespace": endpoints.ObjectMeta.Namespace,
		"endpoints.name":      endpoints.ObjectMeta.Name,
		"endpoints.addr":      endpoints.Subsets,
		"resource.version":    endpoints.ObjectMeta.ResourceVersion,
		"generation.metadata": endpoints.ObjectMeta.Generation,
	}
}
