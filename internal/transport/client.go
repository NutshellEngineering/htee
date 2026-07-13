// Package transport sends built requests and captures the response.
package transport

import (
	"io"
	"net/http"
	"time"
)

// Client sends requests. In this phase, redirects are never followed
// automatically (httpie's own default: a 3xx response is simply printed,
// not chased, unless --follow is given) - the manual redirect loop with
// --follow/--max-redirects/--all support is added in a later phase.
type Client struct {
	HTTP *http.Client
}

// New returns a Client with httpie-like defaults: no automatic redirects.
func New() *Client {
	return &Client{
		HTTP: &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// Result is a sent request's outcome: the response and its fully-read body.
type Result struct {
	Response *http.Response
	Body     []byte
	Elapsed  time.Duration
}

// Send sends req and reads the full response body.
func (c *Client) Send(req *http.Request) (*Result, error) {
	start := time.Now()
	resp, err := c.HTTP.Do(req)
	elapsed := time.Since(start)
	if err != nil {
		return nil, err
	}
	defer mustClose(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &Result{Response: resp, Body: body, Elapsed: elapsed}, nil
}

// mustClose closes c, panicking if the close fails. A failing Close on an
// already fully-read response body indicates a broken transport, not a
// recoverable per-call condition.
func mustClose(c io.Closer) {
	if err := c.Close(); err != nil {
		panic(err)
	}
}
