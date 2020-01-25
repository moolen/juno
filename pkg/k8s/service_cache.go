package k8s

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ServiceCache struct {
	services   chan *v1.Service
	indexer    cache.Indexer
	controller cache.Controller
}

func NewServiceCache(source cache.ListerWatcher, syncInterval time.Duration, bufferSize int) *ServiceCache {
	services := make(chan *v1.Service, bufferSize)
	serviceHandler := &serviceHandler{services}
	indexer, controller := cache.NewIndexerInformer(source, &v1.Service{}, syncInterval, serviceHandler, cache.Indexers{
		indexByIP: serviceIPIndex,
	})
	ServiceCache := &ServiceCache{
		services:   services,
		indexer:    indexer,
		controller: controller,
	}
	return ServiceCache
}

func (s *ServiceCache) getMetadataForIP(ip string) (map[string]string, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("service for %s not found", ip)
	}
	for _, obj := range items {
		svc := obj.(*v1.Service)
		return svc.ObjectMeta.Labels, nil
	}
	return nil, fmt.Errorf("multiple services found")
}

// GetMetadataByIP returns the svc metadata
func (s *ServiceCache) GetMetadataByIP(ip string) (map[string]string, error) {
	return s.getMetadataForIP(ip)
}

func (s *ServiceCache) getByIP(ip string) (*v1.Service, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("service for %s not found", ip)
	}
	for _, obj := range items {
		svc := obj.(*v1.Service)
		return svc, nil
	}
	return nil, fmt.Errorf("multiple services found")
}

// GetByIP returns the service
func (s *ServiceCache) GetByIP(ip string) (*v1.Service, error) {
	return s.getByIP(ip)
}

func serviceIPIndex(obj interface{}) ([]string, error) {
	service := obj.(*v1.Service)
	var out []string
	if service.Spec.ClusterIP != "" {
		out = append(out, service.Spec.ClusterIP)
	}

	return out, nil
}

// Run starts the controller processing updates. Blocks until the cache has synced
func (s *ServiceCache) Run(ctx context.Context) error {
	go s.controller.Run(ctx.Done())
	log.Infof("started cache controller")
	ok := cache.WaitForCacheSync(ctx.Done(), s.controller.HasSynced)
	if !ok {
		return fmt.Errorf("error waiting for sync")
	}
	return nil
}

type serviceHandler struct {
	service chan<- *v1.Service
}

func (o *serviceHandler) announce(svc *v1.Service) {
	logger := log.WithFields(ServiceFields(svc))
	select {
	case o.service <- svc:
		logger.Debugf("announced service")
	default:
		logger.Warnf("service announcement full, dropping")
	}
}

func (o *serviceHandler) OnAdd(obj interface{}) {
	svc, isSvc := obj.(*v1.Service)
	if !isSvc {
		log.Errorf("OnAdd unexpected object: %+v", obj)
		return
	}
	log.WithFields(ServiceFields(svc)).Debugf("added service")
	o.announce(svc)
}

func (o *serviceHandler) OnDelete(obj interface{}) {
	svc, isSvc := obj.(*v1.Service)
	if !isSvc {
		deletedObj, isDeleted := obj.(cache.DeletedFinalStateUnknown)
		if !isDeleted {
			log.Errorf("OnDelete unexpected object: %+v", obj)
			return
		}
		svc, isSvc = deletedObj.Obj.(*v1.Service)
		if !isSvc {
			log.Errorf("OnDelete unexpected DeletedFinalStateUnknown object: %+v", deletedObj.Obj)
		}
		log.WithFields(ServiceFields(svc)).Debugf("deleted service")
		return
	}
	log.WithFields(ServiceFields(svc)).Debugf("deleted service")
	return
}

func (o *serviceHandler) OnUpdate(old, new interface{}) {
	svc, isSvc := new.(*v1.Service)
	if !isSvc {
		log.Errorf("OnUpdate unexpected object: %+v", new)
		return
	}
	log.WithFields(ServiceFields(svc)).Debugf("updated service")
}

func ServiceFields(svc *v1.Service) log.Fields {
	return log.Fields{
		"service.namespace":   svc.ObjectMeta.Namespace,
		"service.name":        svc.ObjectMeta.Name,
		"service.ClusterIP":   svc.Spec.ClusterIP,
		"resource.version":    svc.ObjectMeta.ResourceVersion,
		"generation.metadata": svc.ObjectMeta.Generation,
	}
}
