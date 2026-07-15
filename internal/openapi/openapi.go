// Package openapi extracts the bits of an OpenAPI 3 spec that `ht init`
// uses to pre-populate a new .ht/conf.toml: its servers and its shortest
// GET path, offered as a suggested entrypoint.
package openapi

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Extracted holds the config-relevant data pulled from an OpenAPI document.
type Extracted struct {
	// Servers maps a generated name to each server's URL, in document order.
	Servers map[string]string
	// Entrypoint is the shortest path with a GET operation, or "" if the
	// spec has none.
	Entrypoint string
}

// Load parses the OpenAPI 3 document (YAML or JSON) at path and extracts
// its servers and suggested entrypoint. It does not validate the document
// beyond what's needed to read those fields.
func Load(path string) (Extracted, error) {
	doc, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		return Extracted{}, fmt.Errorf("reading OpenAPI spec: %w", err)
	}

	return Extracted{
		Servers:    serverMap(doc.Servers),
		Entrypoint: shortestGetPath(doc.Paths),
	}, nil
}

func serverMap(servers openapi3.Servers) map[string]string {
	result := make(map[string]string, len(servers))
	for i, s := range servers {
		if s == nil {
			continue
		}
		result[uniqueName(result, serverName(i, s))] = s.URL
	}
	return result
}

func serverName(idx int, s *openapi3.Server) string {
	if slug := slugify(s.Description); slug != "" {
		return slug
	}
	return fmt.Sprintf("server%d", idx+1)
}

func uniqueName(existing map[string]string, name string) string {
	if _, taken := existing[name]; !taken {
		return name
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", name, n)
		if _, taken := existing[candidate]; !taken {
			return candidate
		}
	}
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	slug := nonAlnum.ReplaceAllString(strings.ToLower(s), "-")
	return strings.Trim(slug, "-")
}

// shortestGetPath returns the shortest path (by rune count) with a GET
// operation. Ties break alphabetically for deterministic output.
func shortestGetPath(paths *openapi3.Paths) string {
	if paths == nil {
		return ""
	}

	var candidates []string
	for p, item := range paths.Map() {
		if item != nil && item.Get != nil {
			candidates = append(candidates, p)
		}
	}
	sort.Strings(candidates)

	best := ""
	for _, p := range candidates {
		if best == "" || len(p) < len(best) {
			best = p
		}
	}
	return best
}
