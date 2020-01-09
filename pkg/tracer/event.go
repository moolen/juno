// +build linux

package tracer

import (
	"encoding/binary"
	"fmt"
)

type TraceMetadata struct {
	Ifname string
	SKBLen uint16
}

// ErrInvalidDataLen indicates that the delivered frame had a invalid length
var ErrInvalidDataLen = fmt.Errorf("invalid data length")

func perfEventToGo(data []byte) (*TraceMetadata, []byte, error) {
	if len(data) < 6 {
		return nil, nil, ErrInvalidDataLen
	}
	metadata := data[:6]
	skb := data[6:]
	return &TraceMetadata{
		Ifname: ifname(int(binary.LittleEndian.Uint32(metadata[0:4]))),
		SKBLen: binary.LittleEndian.Uint16(metadata[4:]),
	}, skb, nil
}
