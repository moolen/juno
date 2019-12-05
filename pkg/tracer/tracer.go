package tracer

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/pkg/errors"

	pb "github.com/moolen/juno/proto"

	log "github.com/sirupsen/logrus"
)

// Tracer contains all the information to manage
// a eBPF trace program
type Tracer struct {
	coll         *ebpf.Collection
	perfReader   *perf.Reader
	outChan      chan pb.Trace
	pollInterval time.Duration
	ifacePrefix  string
	stopChan     chan struct{}
}

// NewTracer prepares a eBPF program and a perf event reader
func NewTracer(ifacePrefix string, perfPollInterval time.Duration) (*Tracer, error) {
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
		coll:         coll,
		perfReader:   pr,
		outChan:      make(chan pb.Trace),
		stopChan:     make(chan struct{}),
		pollInterval: perfPollInterval,
		ifacePrefix:  ifacePrefix,
	}, nil
}

// ErrSkipPkg indicates that this packet should be skipped
var ErrSkipPkg = fmt.Errorf("skipped packet")

func processSample(data []byte) (*pb.Trace, error) {
	trace := &pb.Trace{
		Time: &timestamp.Timestamp{
			Seconds: int64(time.Now().Second()),
		},
	}
	var err error
	_, skb, err := perfEventToGo(data)
	if err != nil {
		return nil, err
	}
	packet := gopacket.NewPacket(skb, layers.LayerTypeEthernet, gopacket.Default)

	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		if len(ip.SrcIP) == 0 || len(ip.DstIP) == 0 {
			log.Debugf("skipping empty ip address")
			return nil, ErrSkipPkg
		}
		trace.IP = &pb.IP{
			Source:      ip.SrcIP.String(),
			Destination: ip.DstIP.String(),
			IpVersion:   pb.IPVersion_IPv4,
		}
	} else {
		log.Debugf("skipping non-ip")
		return nil, ErrSkipPkg
	}
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		trace.L4 = &pb.Layer4{
			Protocol: &pb.Layer4_TCP{
				TCP: &pb.TCP{
					SourcePort:      uint32(tcp.SrcPort),
					DestinationPort: uint32(tcp.DstPort),
					Flags: &pb.TCPFlags{
						SYN: tcp.SYN,
						ACK: tcp.ACK,
						PSH: tcp.PSH,
						FIN: tcp.FIN,
						RST: tcp.RST,
						CWR: tcp.CWR,
						ECE: tcp.ECE,
						NS:  tcp.NS,
						URG: tcp.URG,
					},
				},
			},
		}
		appLayer := packet.ApplicationLayer()
		if appLayer != nil {
			rd := strings.NewReader(string(appLayer.Payload()) + "\r\n")
			// TODO: implement header parser
			method, uri, proto, code, err := parseHTTPMetadata(bufio.NewReader(rd))
			if err == nil {
				trace.L7 = &pb.Layer7{
					Record: &pb.Layer7_Http{
						Http: &pb.HTTP{
							Code:     code,
							Method:   method,
							Url:      uri,
							Protocol: proto,
						},
					},
				}
			}
		}
	} else if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		trace.L4 = &pb.Layer4{
			Protocol: &pb.Layer4_UDP{
				UDP: &pb.UDP{
					SourcePort:      uint32(udp.SrcPort),
					DestinationPort: uint32(udp.DstPort),
				},
			},
		}

		if dnsLayer := packet.Layer(layers.LayerTypeDNS); dnsLayer != nil {
			//dns, _ := dnsLayer.(*layers.DNS)
			trace.L7 = &pb.Layer7{
				Record: &pb.Layer7_Dns{
					Dns: &pb.DNS{},
				},
			}
		}

	}
	return trace, nil
}

func (s *Tracer) pollPerfMap() {
	log.Debugf("reading from perfReader channel")
	for {
		select {
		case <-s.stopChan:
			return
		default:
			var flow *pb.Trace
			rec, err := s.perfReader.Read()
			if err != nil {
				log.Error(err)
				continue
			}
			flow, err = processSample(rec.RawSample)
			if err == ErrSkipPkg || err == InvalidDataLen {
				continue
			} else if err != nil {
				log.Error(err)
				continue
			}
			s.outChan <- *flow
			<-time.After(s.pollInterval)
		}
	}
}

// Read returns a channel which outputs trace events
func (s *Tracer) Read() <-chan pb.Trace {
	return s.outChan
}

// Start starts reading from the perf event buffer,
// processes the packets and forwards them to outChan
// Start should be called only once
func (s *Tracer) Start() error {
	log.Debug("starting tracer")
	go s.pollPerfMap()
	err := replaceDatapath(s.coll, s.ifacePrefix)
	if err != nil {
		return err
	}
	return err
}

// Stop stops the internal goroutine for reading from perf event buffer
// and resets the datapath eBPF programs
func (s *Tracer) Stop() {
	log.Debug("stopping tracer")
	close(s.stopChan)
	err := resetDatapath(s.coll, s.ifacePrefix)
	if err != nil {
		log.Error(err)
	}
	s.coll.Close()
}
