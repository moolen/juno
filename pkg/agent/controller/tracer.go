package controller

// import (
// 	log "github.com/sirupsen/logrus"

// 	"github.com/moolen/juno/pkg/tracer"
// )

// type tcpEventTracer struct {
// 	lastTimestampV4 uint64
// 	lastTimestampV6 uint64
// }

// func (t *tcpEventTracer) TCPEventV4(e tracer.TcpV4) {

// 	if e.Type != tracer.EventFdInstall {
// 		tcpEventCounter.WithLabelValues("ipv4", e.Type.String(), e.SAddr.String(), e.DAddr.String()).Inc()
// 		log.Debugf("%v %s %v %s %v:%v %v:%v %v\n",
// 			e.Timestamp, e.Type, e.Pid, e.Comm, e.SAddr, e.SPort, e.DAddr, e.DPort, e.NetNS)
// 	}
// }

// func (t *tcpEventTracer) TCPEventV6(e tracer.TcpV6) {
// 	tcpEventCounter.WithLabelValues("ipv6", e.Type.String(), e.SAddr.String(), e.DAddr.String()).Inc()
// 	log.Debugf("%v %s %v %s %v:%v %v:%v %v\n",
// 		e.Timestamp, e.Type, e.Pid, e.Comm, e.SAddr, e.SPort, e.DAddr, e.DPort, e.NetNS)
// }

// func (t *tcpEventTracer) LostV4(count uint64) {
// 	log.Warnf("ERROR: lost %d events!\n", count)
// }

// func (t *tcpEventTracer) LostV6(count uint64) {
// 	log.Warnf("ERROR: lost %d events!\n", count)
// }
