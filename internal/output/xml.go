package output

import (
	"bytes"
	"encoding/xml"
	"io"
	"strings"
)

// formatXMLBody re-indents an XML body, mirroring output/formatters/xml.py.
// Whitespace-only text nodes from the original formatting are dropped so
// the encoder's own indentation isn't doubled up (toprettyxml's blank-line
// removal, done here by simply never emitting them). Invalid XML is
// returned unchanged.
func formatXMLBody(body []byte, opts FormatOptions) []byte {
	if !opts.XMLFormat {
		return body
	}

	dec := xml.NewDecoder(bytes.NewReader(body))
	var buf bytes.Buffer
	enc := xml.NewEncoder(&buf)
	// Close (idempotent) flushes and validates the token stream is
	// well-formed; deferring it here guarantees it runs on every return
	// path below, including the early "invalid input" returns.
	defer func() {
		_ = enc.Close()
	}()
	enc.Indent("", strings.Repeat(" ", maxInt(opts.XMLIndent, 0)))

	wrote := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return body
		}
		if cd, isCharData := tok.(xml.CharData); isCharData && len(bytes.TrimSpace(cd)) == 0 {
			continue
		}
		if err := enc.EncodeToken(xml.CopyToken(tok)); err != nil {
			return body
		}
		wrote = true
	}
	if !wrote {
		return body
	}
	if err := enc.Close(); err != nil {
		return body
	}
	return buf.Bytes()
}
