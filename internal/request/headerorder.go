package request

import "net/textproto"

// HeaderOrder tracks the declaration order of header names as they're set,
// since Go's http.Header (a map) has no insertion order of its own but
// httpie's request-line rendering (`-p H`) must show headers in the order
// they were set/overridden, with removed headers dropped entirely.
type HeaderOrder struct {
	names []string
	seen  map[string]bool
}

func newHeaderOrder() *HeaderOrder {
	return &HeaderOrder{seen: map[string]bool{}}
}

func (h *HeaderOrder) add(name string) {
	canon := textproto.CanonicalMIMEHeaderKey(name)
	if h.seen[canon] {
		return
	}
	h.seen[canon] = true
	h.names = append(h.names, canon)
}

func (h *HeaderOrder) remove(name string) {
	canon := textproto.CanonicalMIMEHeaderKey(name)
	if !h.seen[canon] {
		return
	}
	delete(h.seen, canon)
	for i, n := range h.names {
		if n == canon {
			h.names = append(h.names[:i], h.names[i+1:]...)
			break
		}
	}
}

// Names returns the header names in declaration order.
func (h *HeaderOrder) Names() []string {
	return h.names
}
