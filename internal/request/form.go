package request

import (
	"net/url"

	"app.getnutshell/htee/internal/itemsyntax"
	"app.getnutshell/htee/internal/ordered"
)

// buildFormBody url-encodes ri.Data (an *ordered.Map of string/primitive
// values) as application/x-www-form-urlencoded.
func buildFormBody(data *ordered.Map) []byte {
	if data == nil {
		return nil
	}
	values := url.Values{}
	for _, k := range data.Keys() {
		v, _ := data.Get(k)
		values.Add(k, itemsyntax.FormatPrimitive(v))
	}
	return []byte(values.Encode())
}
