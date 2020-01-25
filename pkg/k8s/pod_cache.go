package k8s

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type PodCache struct {
	pods       chan *v1.Pod
	indexer    cache.Indexer
	controller cache.Controller
}

func NewPodCache(source cache.ListerWatcher, syncInterval time.Duration, bufferSize int) *PodCache {
	pods := make(chan *v1.Pod, bufferSize)
	podHandler := &podHandler{pods}
	indexer, controller := cache.NewIndexerInformer(source, &v1.Pod{}, syncInterval, podHandler, cache.Indexers{
		indexByIP: podIPIndex,
	})
	PodCache := &PodCache{
		pods:       pods,
		indexer:    indexer,
		controller: controller,
	}
	return PodCache
}

func (s *PodCache) getMetadataForIP(ip string) (map[string]string, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("pod for %s not found", ip)
	}
	for _, obj := range items {
		po := obj.(*v1.Pod)
		return po.ObjectMeta.Labels, nil
	}
	return nil, fmt.Errorf("multiple pods found")
}

// GetMetadataByIP returns the svc metadata
func (s *PodCache) GetMetadataByIP(ip string) (map[string]string, error) {
	return s.getMetadataForIP(ip)
}

func (s *PodCache) getByIP(ip string) (*v1.Pod, error) {
	items, err := s.indexer.ByIndex(indexByIP, ip)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("pod for %s not found", ip)
	}
	for _, obj := range items {
		svc := obj.(*v1.Pod)
		return svc, nil
	}
	return nil, fmt.Errorf("multiple pods found")
}

// GetByIP returns the pod
func (s *PodCache) GetByIP(ip string) (*v1.Pod, error) {
	return s.getByIP(ip)
}

func podIPIndex(obj interface{}) ([]string, error) {
	po := obj.(*v1.Pod)
	var out []string
	if po.Status.PodIP != "" {
		out = append(out, po.Status.PodIP)
	}

	return out, nil
}

// Run starts the controller processing updates. Blocks until the cache has synced
func (s *PodCache) Run(ctx context.Context) error {
	go s.controller.Run(ctx.Done())
	log.Infof("started cache controller")
	ok := cache.WaitForCacheSync(ctx.Done(), s.controller.HasSynced)
	if !ok {
		return fmt.Errorf("error waiting for sync")
	}
	return nil
}

type podHandler struct {
	pod chan<- *v1.Pod
}

func (o *podHandler) announce(po *v1.Pod) {
	logger := log.WithFields(PodFields(po))
	select {
	case o.pod <- po:
		logger.Debugf("announced pod")
	default:
		logger.Warnf("pod announcement full, dropping")
	}
}

func (o *podHandler) OnAdd(obj interface{}) {
	svc, isSvc := obj.(*v1.Pod)
	if !isSvc {
		log.Errorf("OnAdd unexpected object: %+v", obj)
		return
	}
	log.WithFields(PodFields(svc)).Debugf("added pod")
	o.announce(svc)
}

func (o *podHandler) OnDelete(obj interface{}) {
	svc, isSvc := obj.(*v1.Pod)
	if !isSvc {
		deletedObj, isDeleted := obj.(cache.DeletedFinalStateUnknown)
		if !isDeleted {
			log.Errorf("OnDelete unexpected object: %+v", obj)
			return
		}
		svc, isSvc = deletedObj.Obj.(*v1.Pod)
		if !isSvc {
			log.Errorf("OnDelete unexpected DeletedFinalStateUnknown object: %+v", deletedObj.Obj)
		}
		log.WithFields(PodFields(svc)).Debugf("deleted pod")
		return
	}
	log.WithFields(PodFields(svc)).Debugf("deleted pod")
	return
}

func (o *podHandler) OnUpdate(old, new interface{}) {
	svc, isSvc := new.(*v1.Pod)
	if !isSvc {
		log.Errorf("OnUpdate unexpected object: %+v", new)
		return
	}
	log.WithFields(PodFields(svc)).Debugf("updated pod")
}

func PodFields(svc *v1.Pod) log.Fields {
	return log.Fields{
		"pod.namespace":       svc.ObjectMeta.Namespace,
		"pod.name":            svc.ObjectMeta.Name,
		"pod.Status.PodIP":    svc.Status.PodIP,
		"resource.version":    svc.ObjectMeta.ResourceVersion,
		"generation.metadata": svc.ObjectMeta.Generation,
	}
}
