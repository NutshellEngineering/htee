package output

import (
	"sort"
	"strings"

	"app.getnutshell/htee/internal/message"
)

// sortHeaders stably sorts headers by name, retaining the relative order
// of repeated header names - mirrors output/formatters/headers.py.
func sortHeaders(headers []message.Header) []message.Header {
	out := append([]message.Header(nil), headers...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// mimeOf extracts the base MIME type (no parameters, e.g. "; charset=utf-8")
// from a header list's Content-Type value, lowercased.
func mimeOf(headers []message.Header) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, "Content-Type") {
			mime, _, _ := strings.Cut(h.Value, ";")
			return strings.ToLower(strings.TrimSpace(mime))
		}
	}
	return ""
}
