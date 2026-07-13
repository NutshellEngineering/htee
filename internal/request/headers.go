package request

import (
	"fmt"
	"net/http"

	"app.getnutshell/htee/internal/itemsyntax"
)

// Version is the client version reported in the User-Agent header.
var Version = "0.1.0"

// applyDefaultHeaders sets httpie's default headers for the given body
// mode, in an order such that explicit header items (applied afterward)
// can still override or remove them.
func applyDefaultHeaders(h http.Header, order *HeaderOrder, mode BodyMode, hasBody bool) {
	h.Set("User-Agent", fmt.Sprintf("htee/%s", Version))
	order.add("User-Agent")
	switch mode {
	case BodyModeJSON:
		h.Set("Accept", "application/json, */*;q=0.5")
		order.add("Accept")
		if hasBody {
			h.Set("Content-Type", "application/json")
			order.add("Content-Type")
		}
	case BodyModeForm:
		if hasBody {
			h.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
			order.add("Content-Type")
		}
	case BodyModeMultipart:
		// Content-Type (with boundary) is set separately once known.
	case BodyModeRaw:
		// No default Content-Type; left to the user via an explicit header item.
	}
}

// applyHeaderItems applies `:`/`;`/`:@` header items on top of the default
// headers. A nil Value (from an empty-header item, e.g. `Accept;`) removes
// the header entirely (used to suppress a default httpie would otherwise
// send, e.g. `Accept:` or `Connection:`).
func applyHeaderItems(h http.Header, order *HeaderOrder, headers []itemsyntax.HeaderField) {
	for _, hf := range headers {
		if hf.Value == nil {
			h.Del(hf.Name)
			order.remove(hf.Name)
			continue
		}
		h.Set(hf.Name, *hf.Value)
		order.add(hf.Name)
	}
}
