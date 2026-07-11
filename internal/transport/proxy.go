package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ProxyFunc builds an http.Transport-compatible Proxy function from
// httpie's repeatable `--proxy protocol:URL` flag, e.g.
// "http:http://localhost:8080", "https:http://localhost:8080", or an
// "all:..." catch-all applied when no scheme-specific entry matches.
// Returns a nil func (not an error) when raw is empty, so callers can
// leave http.Transport.Proxy at its zero value.
func ProxyFunc(raw []string) (func(*http.Request) (*url.URL, error), error) {
	if len(raw) == 0 {
		return nil, nil
	}
	byScheme := make(map[string]*url.URL, len(raw))
	for _, entry := range raw {
		scheme, target, ok := strings.Cut(entry, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --proxy %q (expected protocol:URL, e.g. http:http://localhost:8080)", entry)
		}
		u, err := url.Parse(target)
		if err != nil {
			return nil, fmt.Errorf("invalid --proxy URL %q: %w", target, err)
		}
		byScheme[scheme] = u
	}
	return func(req *http.Request) (*url.URL, error) {
		if u, ok := byScheme[req.URL.Scheme]; ok {
			return u, nil
		}
		if u, ok := byScheme["all"]; ok {
			return u, nil
		}
		return nil, nil
	}, nil
}
