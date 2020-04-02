package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	traceEventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "trace_event_count",
		Help: "juno agent trace event counter",
	}, []string{"node"})
)

func init() {
}
