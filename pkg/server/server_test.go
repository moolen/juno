package server

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/moolen/juno/pkg/ipcache"
	pb "github.com/moolen/juno/proto"
	sg "github.com/moolen/statusgraph/pkg/store"
)

func TestBuildID(t *testing.T) {

	tbl := []struct {
		trace  *pb.Trace
		src    *ipcache.Endpoint
		dst    *ipcache.Endpoint
		expSrc *sg.Node
		expDst *sg.Node
	}{
		{
			// sauce:39198 -> dest:8080
			trace: &pb.Trace{
				IP: &pb.IP{
					Source:      "10.0.3.11",
					Destination: "10.0.3.22",
				},
				L4: &pb.Layer4{
					Protocol: &pb.Layer4_TCP{
						TCP: &pb.TCP{
							DestinationPort: 8080,
							SourcePort:      39198,
						},
					},
				},
			},
			src: &ipcache.Endpoint{
				Name:      "sauce",
				Namespace: "sauce-ns",
				Labels: map[string]string{
					"app": "sauce-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 3000,
					},
				},
			},
			dst: &ipcache.Endpoint{
				Name:      "dest",
				Namespace: "dest-ns",
				Labels: map[string]string{
					"app": "dest-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
			expSrc: &sg.Node{
				Name: "sauce-app",
			},
			expDst: &sg.Node{
				Name: "dest-app",
				Connector: []sg.NodeConnector{
					{
						Name:  "http",
						Label: "8080",
					},
				},
			},
		},
		{
			// response packets
			// dest:8080 -> sauce:39198
			trace: &pb.Trace{
				IP: &pb.IP{
					Source:      "10.0.3.22",
					Destination: "10.0.3.11",
				},
				L4: &pb.Layer4{
					Protocol: &pb.Layer4_TCP{
						TCP: &pb.TCP{
							SourcePort:      8080,
							DestinationPort: 39198,
						},
					},
				},
			},
			dst: &ipcache.Endpoint{
				Name:      "sauce",
				Namespace: "sauce-ns",
				Labels: map[string]string{
					"app": "sauce-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 3000,
					},
				},
			},
			src: &ipcache.Endpoint{
				Name:      "dest",
				Namespace: "dest-ns",
				Labels: map[string]string{
					"app": "dest-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
			expSrc: &sg.Node{
				Name: "sauce-app",
			},
			expDst: &sg.Node{
				Name: "dest-app",
				Connector: []sg.NodeConnector{
					{
						Name:  "http",
						Label: "8080",
					},
				},
			},
		},
		{
			// ephemere, but port exists
			// dest:33333 -> sauce:33333
			trace: &pb.Trace{
				IP: &pb.IP{
					Source:      "10.0.3.22",
					Destination: "10.0.3.11",
				},
				L4: &pb.Layer4{
					Protocol: &pb.Layer4_TCP{
						TCP: &pb.TCP{
							SourcePort:      33333,
							DestinationPort: 33333,
						},
					},
				},
			},
			dst: &ipcache.Endpoint{
				Name:      "sauce",
				Namespace: "sauce-ns",
				Labels: map[string]string{
					"app": "sauce-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 33333,
					},
				},
			},
			src: &ipcache.Endpoint{
				Name:      "dest",
				Namespace: "dest-ns",
				Labels: map[string]string{
					"app": "dest-app",
				},
				Ports: []ipcache.Port{
					{
						Name: "http",
						Port: 8080,
					},
				},
			},
			expSrc: &sg.Node{
				Name: "dest-app",
			},
			expDst: &sg.Node{
				Name: "sauce-app",
				Connector: []sg.NodeConnector{
					{
						Name:  "http",
						Label: "33333",
					},
				},
			},
		},
	}

	for i, row := range tbl {
		s, d, err := buildID(row.trace, row.src, row.dst)
		if err != nil {
			t.Errorf("[%d] unexpected err", i)
		}
		if !cmp.Equal(s, row.expSrc) {
			t.Errorf("[%d] unexpected source. found %v, expected %v", i, s, row.expSrc)
		}
		if !cmp.Equal(d, row.expDst) {
			t.Errorf("[%d] unexpected dst. found %v, expected %v", i, d, row.expDst)
		}

	}
}
