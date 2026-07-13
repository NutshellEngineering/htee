package request

import (
	"encoding/json"

	"app.getnutshell/htee/internal/ordered"
)

// buildJSONBody marshals ri.Data to JSON. An empty object (no data items at
// all) produces no body at all, matching httpie: an empty dict is not sent
// as literal "{}" .
func buildJSONBody(data any) (body []byte, hasBody bool, err error) {
	if data == nil {
		return nil, false, nil
	}
	if m, ok := data.(*ordered.Map); ok && m.Len() == 0 {
		return nil, false, nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return nil, false, err
	}
	return b, true, nil
}
