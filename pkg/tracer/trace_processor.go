package tracer

import (
	"bufio"
	"fmt"
	"strings"
	"time"

	timestamp "github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	pb "github.com/moolen/juno/proto"
	log "github.com/sirupsen/logrus"
)

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
