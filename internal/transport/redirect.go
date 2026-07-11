package transport

import (
	"bytes"
	"fmt"
	"net/http"
	"net/textproto"
)

// isRedirectStatus reports whether code is one of the HTTP redirect status
// codes httpie/requests follows: 301, 302, 303, 307, 308. (300 and 304 are
// deliberately excluded - they don't carry a resource redirect Location in
// the same "go fetch this instead" sense.)
func isRedirectStatus(code int) bool {
	switch code {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	}
	return false
}

// nextRequest builds the request for the hop after resp, applying the same
// method/body-rewrite rules as Go's stdlib redirect handling and Python
// requests' Session.resolve_redirects: 301/302 rewrite a POST to a
// bodyless GET; 303 rewrites any non-HEAD method to a bodyless GET; 307/308
// preserve the method and body exactly. The Authorization header (and its
// entry in the header-order list used for display) is dropped when the
// redirect target's host differs from the original request's host, so
// credentials aren't leaked to a different origin.
func nextRequest(prev *http.Request, prevBody []byte, prevHeaderOrder []string, resp *http.Response) (*http.Request, []byte, []string, error) {
	loc := resp.Header.Get("Location")
	if loc == "" {
		return nil, nil, nil, fmt.Errorf("redirect response (%d) missing Location header", resp.StatusCode)
	}
	target, err := prev.URL.Parse(loc)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid redirect Location %q: %w", loc, err)
	}

	method := prev.Method
	body := prevBody
	switch resp.StatusCode {
	case http.StatusMovedPermanently, http.StatusFound:
		if method == http.MethodPost {
			method, body = http.MethodGet, nil
		}
	case http.StatusSeeOther:
		if method != http.MethodHead {
			method, body = http.MethodGet, nil
		}
	}

	next, err := http.NewRequest(method, target.String(), bytes.NewReader(body))
	if err != nil {
		return nil, nil, nil, err
	}
	next.Header = prev.Header.Clone()
	order := prevHeaderOrder

	if body == nil && prevBody != nil {
		next.Header.Del("Content-Type")
		next.Header.Del("Content-Length")
		order = removeHeaderNames(order, "Content-Type", "Content-Length")
	}
	next.ContentLength = int64(len(body))

	if target.Host != prev.URL.Host {
		next.Header.Del("Authorization")
		order = removeHeaderNames(order, "Authorization")
	}

	return next, body, order, nil
}

// removeHeaderNames returns order with the canonicalized names dropped,
// preserving the relative order of everything else.
func removeHeaderNames(order []string, names ...string) []string {
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[textproto.CanonicalMIMEHeaderKey(n)] = true
	}
	out := make([]string, 0, len(order))
	for _, n := range order {
		if !drop[n] {
			out = append(out, n)
		}
	}
	return out
}
