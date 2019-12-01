package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	httpEventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_event",
		Help: "http trace event",
	}, []string{"method", "source_address", "dest_address", "source_port", "dest_port"})
)

func init() {
	metrics.Registry.Register(httpEventCounter)
}
