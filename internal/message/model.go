// Package message models HTTP request/response messages for rendering
// (independent of *http.Request/*http.Response so --offline and normal
// send/receive share the same renderer), and resolves httpie's print-flag
// (-p/-h/-b/-v/--all) semantics.
package message

import (
	"net/http"
	"sort"
)

// Header is one rendered header line.
type Header struct {
	Name  string
	Value string
}

// Request is a renderable request message.
type Request struct {
	Method  string
	Target  string // request-target: path + "?query"
	Proto   string
	Headers []Header
	Body    []byte
}

// Response is a renderable response message.
type Response struct {
	Proto      string
	StatusCode int
	Reason     string
	Headers    []Header
	Body       []byte
	Elapsed    float64 // seconds
}

// FromRequest builds a Request from a sent/about-to-be-sent *http.Request,
// rendering headers in declaration order (headerOrder) and synthesizing a
// Host header at the end if one wasn't explicitly set - Go stores the host
// out-of-band on the Request rather than in its Header map.
func FromRequest(req *http.Request, body []byte, headerOrder []string) Request {
	proto := req.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}
	var headers []Header
	hasHost := false
	for _, name := range headerOrder {
		if name == "Host" {
			hasHost = true
		}
		for _, v := range req.Header.Values(name) {
			headers = append(headers, Header{Name: name, Value: v})
		}
	}
	if !hasHost {
		host := req.Host
		if host == "" {
			host = req.URL.Host
		}
		headers = append(headers, Header{Name: "Host", Value: host})
	}
	return Request{
		Method:  req.Method,
		Target:  req.URL.RequestURI(),
		Proto:   proto,
		Headers: headers,
		Body:    body,
	}
}

// FromResponse builds a Response from an *http.Response. Response header
// order isn't recoverable from Go's http.Header (a map, populated during
// wire parsing without preserving name-to-name order), so headers are
// rendered alphabetically - which also matches httpie's own default
// `headers.sort` output-formatting behavior.
func FromResponse(resp *http.Response, body []byte, elapsedSeconds float64) Response {
	names := make([]string, 0, len(resp.Header))
	for name := range resp.Header {
		names = append(names, name)
	}
	sort.Strings(names)

	var headers []Header
	for _, name := range names {
		for _, v := range resp.Header.Values(name) {
			headers = append(headers, Header{Name: name, Value: v})
		}
	}

	reason := resp.Status
	// resp.Status is typically "200 OK"; strip the leading status code and space.
	for i := 0; i < len(reason); i++ {
		if reason[i] == ' ' {
			reason = reason[i+1:]
			break
		}
	}

	proto := resp.Proto
	if proto == "" {
		proto = "HTTP/1.1"
	}

	return Response{
		Proto:      proto,
		StatusCode: resp.StatusCode,
		Reason:     reason,
		Headers:    headers,
		Body:       body,
		Elapsed:    elapsedSeconds,
	}
}
