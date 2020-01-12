package tracer

import (
	"bufio"
	"fmt"
	"net/textproto"
	"strconv"
	"strings"
	"unicode/utf8"
)

// HTTPMetadata contains HTTP metadata
type HTTPMetadata struct {
	Method string
	URI    string
	Proto  string
}

var ErrUnsupportedProto = fmt.Errorf("unsupported l7 proto")

func parseHTTPMetadata(input *bufio.Reader) (string, string, string, uint32, error) {
	tr := textproto.NewReader(input)
	line, err := tr.ReadLine()
	if err != nil {
		return "", "", "", 0, err
	}
	p1, p2, p3, ok := parseRequestLine(line)
	if !ok {
		return "", "", "", 0, fmt.Errorf("could not parse HTTP request line")
	}
	var method, uri, proto string
	var code uint32
	// this is a response
	if p1 == "HTTP/1.1" {
		proto = p1
		c, _ := strconv.Atoi(p2)
		code = uint32(c)
	} else if p3 == "HTTP/1.1" {
		method = p1
		uri = p2
		proto = p3
	} else {
		return "", "", "", 0, ErrUnsupportedProto
	}

	for _, v := range []string{method, uri, proto} {
		if !utf8.ValidString(v) {
			return "", "", "", 0, fmt.Errorf("invalid utf8 string: %#v", v)
		}
	}

	return method, uri, proto, code, nil
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
