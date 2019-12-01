package tracer

import (
	"bufio"
	"os"
	"strings"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

// Tracer ..
type Tracer struct {
	coll        *ebpf.Collection
	perfReader  *perf.Reader
	outChan     chan TraceEvent
	ifacePrefix string
}

// NewTracer ..
func NewTracer(ifacePrefix string) (*Tracer, error) {
	log.Info("loading tracer")
	coll, err := compileAndLoad()
	if err != nil {
		return nil, errors.Wrap(err, "error compiling and loading eBPF")
	}
	perfMap := coll.Maps["EVENTS_MAP"]
	if perfMap == nil {
		return nil, errors.Wrap(err, "missing events map")
	}
	pr, err := perf.NewReader(perfMap, os.Getpagesize())
	if err != nil {
		return nil, errors.Wrap(err, "error creating event reader")
	}
	return &Tracer{
		coll:        coll,
		perfReader:  pr,
		outChan:     make(chan TraceEvent),
		ifacePrefix: ifacePrefix,
	}, nil
}

func processSample(data []byte) (*TraceEvent, error) {
	trace := &TraceEvent{}
	var err error
	metadata, skb, err := perfEventToGo(data)
	if err != nil {
		return nil, err
	}
	trace.Metadata = metadata
	log.Debugf("(%d) %x", len(data), skb)

	packet := gopacket.NewPacket(skb, layers.LayerTypeEthernet, gopacket.Default)
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		trace.SourceAddr = ip.SrcIP
		trace.DestAddr = ip.DstIP
		trace.L3Proto = "IPv4"
	}
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		trace.SourcePort = uint16(tcp.SrcPort)
		trace.DestPort = uint16(tcp.DstPort)
		trace.L4Proto = "TCP"

		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			trace.L7Proto = "unknown"
			rd := strings.NewReader(string(appLayer.Payload()) + "\r\n")
			l7meta, _ := parseHTTPMetadata(bufio.NewReader(rd))
			if err != nil {
				trace.L7Proto = "HTTP"
				trace.L7Metadata = l7meta
			}
		}
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		trace.SourcePort = uint16(udp.SrcPort)
		trace.DestPort = uint16(udp.DstPort)
		trace.L4Proto = "UDP"
		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			dns, _ := dnsLayer.(*layers.DNS)
			trace.L7Proto = "DNS"
			trace.L7Metadata, err = parseDNSMetadata(dns)
		}
	}
	return trace, nil
}

func (s *Tracer) Read() <-chan TraceEvent {
	return s.outChan
}

// Start ..
func (s *Tracer) Start() error {
	go func() {
		for {
			var trace *TraceEvent
			rec, err := s.perfReader.Read()
			if err != nil {
				log.Error(err)
				continue
			}
			trace, err = processSample(rec.RawSample)
			if err != nil {
				log.Error(err)
				continue
			}
			s.outChan <- *trace
		}
	}()
	err := replaceDatapath(s.coll, s.ifacePrefix)
	if err != nil {
		return err
	}
	return err
}

// Stop ..
func (s *Tracer) Stop() {
	err := resetDatapath(s.coll, s.ifacePrefix)
	if err != nil {
		log.Error(err)
	}
	s.coll.Close()
}
