package tracer

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseHTTPPartial(t *testing.T) {
	for _, row := range []struct {
		input  string
		method string
		uri    string
		proto  string
		code   uint32
		err    bool
	}{
		{
			input: "",
			err:   true,
		},
		{
			input: "y u no goat",
			err:   true,
		},
		{
			input: "GET /foo HTTP/2.7",
			err:   true,
		},
		{
			input:  "GET /foo HTTP/1.1",
			code:   0,
			method: "GET",
			proto:  "HTTP/1.1",
			uri:    "/foo",
			err:    false,
		},
		{
			input:  "HTTP/1.1 200 OK",
			code:   200,
			method: "",
			proto:  "HTTP/1.1",
			uri:    "",
			err:    false,
		},
	} {
		method, uri, proto, code, err := parseHTTPMetadata(bufio.NewReader(strings.NewReader(row.input)))
		if (err != nil && !row.err) || (err == nil && row.err) {
			t.Errorf("unexpected err result. expected %v, got %v", row.err, err)
		}

		if row.code != code {
			t.Errorf("unexpected code. expected %v, found %v", row.code, code)
		}

		if row.uri != uri {
			t.Errorf("unexpected uri. expected %v, found %v", row.uri, uri)
		}

		if row.proto != proto {
			t.Errorf("unexpected proto. expected %v, found %v", row.proto, proto)
		}

		if row.method != method {
			t.Errorf("unexpected method. expected %v, found %v", row.method, method)
		}
	}

}
