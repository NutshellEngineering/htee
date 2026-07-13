package nestedjson

import (
	"fmt"
	"strings"
)

// SyntaxError is raised for malformed bracket-path keys or type conflicts
// while interpreting them (e.g. indexing into a string). Mirrors
// httpie/cli/nested_json/errors.py:NestedJSONSyntaxError.
type SyntaxError struct {
	Source      string
	Token       *Token // nil if there's no specific token to highlight
	Message     string
	MessageKind string // "Syntax" or "Type" or "Value"
}

func (e *SyntaxError) Error() string {
	kind := e.MessageKind
	if kind == "" {
		kind = "Syntax"
	}
	var b strings.Builder
	// strings.Builder.Write never returns an error, so Fprintf can't fail here.
	_, _ = fmt.Fprintf(&b, "htee %s Error: %s", kind, e.Message)
	if e.Token != nil {
		b.WriteByte('\n')
		b.WriteString(e.Source)
		b.WriteByte('\n')
		b.WriteString(strings.Repeat(" ", e.Token.Start))
		b.WriteString(strings.Repeat("^", e.Token.End-e.Token.Start))
	}
	return b.String()
}

func newSyntaxError(source string, tok *Token, message string) *SyntaxError {
	return &SyntaxError{Source: source, Token: tok, Message: message}
}

func newTypeError(source string, tok *Token, message string) *SyntaxError {
	return &SyntaxError{Source: source, Token: tok, Message: message, MessageKind: "Type"}
}

func newValueError(source string, tok *Token, message string) *SyntaxError {
	return &SyntaxError{Source: source, Token: tok, Message: message, MessageKind: "Value"}
}
