package tracer

import (
	"strconv"

	"github.com/google/gopacket/layers"
)

func parseDNSMetadata(dns *layers.DNS) (map[string]string, error) {
	m := make(map[string]string)
	m["OPCODE"] = dns.OpCode.String()
	m["QR"] = strconv.FormatBool(dns.QR)
	return m, nil
}
