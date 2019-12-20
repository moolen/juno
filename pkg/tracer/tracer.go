package tracer

import (
	"os"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
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
			if err == ErrSkipPkg || err == ErrInvalidDataLen {
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
