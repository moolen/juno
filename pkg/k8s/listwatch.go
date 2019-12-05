package k8s

import (
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

func NewListWatch(client *kubernetes.Clientset, resource string) *cache.ListWatch {
	return cache.NewListWatchFromClient(client.CoreV1().RESTClient(), resource, "", fields.Everything())
}
