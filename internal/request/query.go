package request

import (
	"net/url"
	"strings"

	"app.getnutshell/htee/internal/itemsyntax"
)

// applyQueryParams appends `==`/`==@` query items to an existing raw query
// string, preserving declaration order (unlike net/url.Values.Encode, which
// alphabetizes keys).
func applyQueryParams(existing string, params []itemsyntax.QueryParam) string {
	if len(params) == 0 {
		return existing
	}
	var parts []string
	if existing != "" {
		parts = append(parts, existing)
	}
	for _, p := range params {
		parts = append(parts, url.QueryEscape(p.Name)+"="+url.QueryEscape(p.Value))
	}
	return strings.Join(parts, "&")
}
