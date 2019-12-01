package tracer

import (
	"bufio"
	"fmt"
	"net/textproto"
	"strings"
)

// HTTPMetadata contains HTTP metadata
type HTTPMetadata struct {
	Method string
	URI    string
	Proto  string
}

func parseHTTPMetadata(input *bufio.Reader) (map[string]string, error) {
	tr := textproto.NewReader(input)
	line, err := tr.ReadLine()
	if err != nil {
		return nil, err
	}
	method, uri, proto, ok := parseRequestLine(line)
	if !ok {
		return nil, fmt.Errorf("could not parse HTTP request line")
	}
	return map[string]string{
		"METHOD": method,
		"URI":    uri,
		"PROTO":  proto,
	}, nil
}

func (h *HTTPMetadata) String() string {
	return fmt.Sprintf("%s %s %s", h.Method, h.URI, h.Proto)
}

// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func parseRequestLine(line string) (method, requestURI, proto string, ok bool) {
	s1 := strings.Index(line, " ")
	s2 := strings.Index(line[s1+1:], " ")
	if s1 < 0 || s2 < 0 {
		return
	}
	s2 += s1 + 1
	return line[:s1], line[s1+1 : s2], line[s2+1:], true
}
