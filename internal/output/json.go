package output

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"

	"app.getnutshell/htee/internal/ordered"
)

// formatJSONBody re-indents (and optionally key-sorts) a JSON body,
// mirroring output/formatters/json.py's JSONFormatter. It tolerates a
// leading non-JSON prefix (e.g. an XSSI guard like `while(1);`) by locating
// the first `{`/`[` and parsing from there, same as load_prefixed_json.
// Invalid JSON is returned unchanged (best-effort, matching httpie).
func formatJSONBody(body []byte, opts FormatOptions) []byte {
	if !opts.JSONFormat {
		return body
	}
	prefix, jsonPart, ok := splitJSONPrefix(body)
	if !ok {
		return body
	}
	val, err := ordered.DecodeJSON(jsonPart)
	if err != nil {
		return body
	}
	if opts.JSONSortKeys {
		val = sortJSONValue(val)
	}
	compact, err := json.Marshal(val)
	if err != nil {
		return body
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, compact, "", strings.Repeat(" ", maxInt(opts.JSONIndent, 0))); err != nil {
		return body
	}
	out := make([]byte, 0, len(prefix)+buf.Len())
	out = append(out, prefix...)
	out = append(out, buf.Bytes()...)
	return out
}

// splitJSONPrefix finds the JSON value within body, tolerating a leading
// non-JSON prefix. Returns the prefix, the JSON slice, and whether a
// plausible JSON value was found at all (parsing is attempted by the
// caller; this only locates candidate boundaries).
func splitJSONPrefix(body []byte) (prefix []byte, jsonPart []byte, ok bool) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil, nil, false
	}
	if trimmed[0] == '{' || trimmed[0] == '[' {
		return nil, trimmed, true
	}
	idx := bytes.IndexAny(body, "{[")
	if idx < 0 {
		return nil, nil, false
	}
	return body[:idx], body[idx:], true
}

// sortJSONValue recursively sorts *ordered.Map keys (json.sort_keys),
// applied throughout nested objects/arrays like Python's
// json.dumps(sort_keys=True).
func sortJSONValue(v any) any {
	switch t := v.(type) {
	case *ordered.Map:
		keys := append([]string(nil), t.Keys()...)
		sort.Strings(keys)
		sorted := ordered.NewMap()
		for _, k := range keys {
			val, _ := t.Get(k)
			sorted.Set(k, sortJSONValue(val))
		}
		return sorted
	case []any:
		out := make([]any, len(t))
		for i, e := range t {
			out[i] = sortJSONValue(e)
		}
		return out
	default:
		return v
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
