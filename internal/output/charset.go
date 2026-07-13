package output

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
)

// decodeCharset re-encodes body from the named charset (e.g. "iso-8859-1")
// into UTF-8, for --response-charset. Returns an error for unrecognized
// encoding names, matching httpie's response_charset_type validation.
func decodeCharset(body []byte, name string) ([]byte, error) {
	enc, err := htmlindex.Get(name)
	if err != nil {
		return nil, fmt.Errorf("unknown --response-charset %q", name)
	}
	out, err := io.ReadAll(enc.NewDecoder().Reader(strings.NewReader(string(body))))
	if err != nil {
		return nil, fmt.Errorf("failed to decode body as %q: %w", name, err)
	}
	return out, nil
}
