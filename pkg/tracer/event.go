// +build linux

package tracer

/*
#include "../../bpf/tcptracer-sock-bpf.h"
*/
import "C"
import (
	"encoding/binary"
	"fmt"
)

type TraceMetadata struct {
	Ifname string
	SKBLen uint16
}

var InvalidDataLen = fmt.Errorf("invalid data length")

func perfEventToGo(data []byte) (*TraceMetadata, []byte, error) {
	if len(data) < 6 {
		return nil, nil, InvalidDataLen
	}
	metadata := data[:6]
	skb := data[6:]
	return &TraceMetadata{
		Ifname: ifname(int(binary.LittleEndian.Uint32(metadata[0:4]))),
		SKBLen: binary.LittleEndian.Uint16(metadata[4:]),
	}, skb, nil
}
