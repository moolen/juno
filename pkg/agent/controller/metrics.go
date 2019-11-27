package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	tcpEventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tcp_event",
		Help: "tcp event",
	}, []string{"protocol", "status", "source", "destination"})
)

func init() {
	metrics.Registry.Register(tcpEventCounter)
}
