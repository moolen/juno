package tracer

import (
	"fmt"
	"net"
	"strings"
)

// TraceEvent ..
type TraceEvent struct {
	Metadata   *TraceMetadata
	SourceAddr net.IP
	SourcePort uint16
	DestAddr   net.IP
	DestPort   uint16
	L3Proto    L3Proto
	L4Proto    L4Proto
	L7Proto    L7Proto
	L7Metadata L7Metadata
}

type L3Proto string

const (
	L3IPv4 L3Proto = "IPv4"
	L3IPv6 L3Proto = "IPv6"
)

type L4Proto string

const (
	L4TCP L4Proto = "TCP"
	L4UDP L4Proto = "UDP"
)

type L7Proto string

const (
	L7HTTP L7Proto = "HTTP"
	L7DNS  L7Proto = "DNS"
)

// L7Metadata ..
type L7Metadata map[string]string

func (t *TraceEvent) String() string {
	return fmt.Sprintf(
		"[%s] [%d] [%s/%s/%s] %s:%d -> %s:%d | %s",
		t.Metadata.Ifname, t.Metadata.SKBLen, t.L3Proto, t.L4Proto, t.L7Proto,
		t.SourceAddr, t.SourcePort, t.DestAddr, t.DestPort, t.L7Metadata.String(),
	)
}
func (m L7Metadata) String() string {
	var s string
	for k, v := range m {
		s += fmt.Sprintf("%s=%s\n", k, v)
	}
	return strings.TrimRight(s, "\n")
}
