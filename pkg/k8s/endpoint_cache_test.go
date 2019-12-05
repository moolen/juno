package k8s

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kt "k8s.io/client-go/tools/cache/testing"
)

const bufferSize = 10

func TestFindMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := kt.NewFakeControllerSource()
	c := NewEndpointCache(source, time.Second, bufferSize)
	source.Add(NewEndpoint("foo", "default", map[string]string{
		"app": "foo",
	}, "1.1.1.1"))
	source.Add(NewEndpoint("bar", "default", map[string]string{
		"app": "bar",
	}, "1.2.3.4"))
	err := c.Run(ctx)
	if err != nil {
		t.Error(err)
	}
	defer source.Shutdown()
	found, _ := c.GetMetadataByIP("1.2.3.4")
	if found == nil {
		t.Error("should have found endpoint")
	}
	if found["app"] != "bar" {
		t.Errorf("should have found app=bar")
	}

	found, _ = c.GetMetadataByIP("1.1.1.1")
	if found == nil {
		t.Error("should have found endpoint")
	}
	if found["app"] != "foo" {
		t.Errorf("should have found app=foo")
	}

}

func TestFindIP(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := kt.NewFakeControllerSource()
	c := NewEndpointCache(source, time.Second, bufferSize)
	source.Add(NewEndpoint("foo", "default", map[string]string{
		"app": "foo",
	}, "1.1.1.1"))
	source.Add(NewEndpoint("bar", "default", map[string]string{
		"app": "bar",
	}, "1.2.3.4"))
	err := c.Run(ctx)
	if err != nil {
		t.Error(err)
	}
	defer source.Shutdown()
	ep, _ := c.GetEndpointByIP("1.2.3.4")
	if ep.ObjectMeta.Name != "bar" {
		t.Errorf("should have found endpoint. found: %#v", ep)
	}

	ep, _ = c.GetEndpointByIP("1.1.1.1")
	if ep.ObjectMeta.Name != "foo" {
		t.Errorf("should have found endpoint. found: %#v", ep)
	}

}

func NewEndpoint(name, namespace string, labels map[string]string, ip string) *v1.Endpoints {
	return &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         namespace,
			Name:              name,
			ResourceVersion:   fmt.Sprintf("%d", time.Now().UnixNano()),
			CreationTimestamp: metav1.Now(),
			Labels:            labels,
		},
		Subsets: []v1.EndpointSubset{
			{
				Addresses: []v1.EndpointAddress{
					{
						IP: ip,
					},
				},
			},
		},
	}
}
