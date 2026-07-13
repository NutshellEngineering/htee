package output

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// autoUnderlyingStyle backs --style=auto: any style works as input to the
// 16-color terminal formatter, since it quantizes down to the terminal's
// own ANSI palette regardless of the style's exact colors.
const autoUnderlyingStyle = "monokai"

// ValidStyle reports whether name is "auto" or a style chroma knows about,
// for validating --style up front (mirrors httpie's lazy_choices).
func ValidStyle(name string) bool {
	if name == "" || name == AutoStyle {
		return true
	}
	_, ok := styles.Registry[strings.ToLower(name)]
	return ok
}

// resolveStyleAndFormatter implements the plan's chroma mapping: --style=auto
// (default) uses the basic 16-color terminal formatter (follows the
// terminal's own palette); a named style uses the 256-color formatter.
func resolveStyleAndFormatter(name string) (*chroma.Style, chroma.Formatter) {
	if name == "" || name == AutoStyle {
		return styles.Get(autoUnderlyingStyle), formatters.TTY16
	}
	return styles.Get(name), formatters.TTY256
}

// colorizeBody syntax-highlights body per its MIME type. If no lexer
// matches, body is returned unchanged (uncolored) - matches httpie's
// behavior when Pygments has no lexer for the content type.
func colorizeBody(body []byte, mime string, style *chroma.Style, formatter chroma.Formatter) []byte {
	lexer := lexerForMime(mime, body)
	if lexer == nil {
		return body
	}
	lexer = chroma.Coalesce(lexer)
	iterator, err := lexer.Tokenise(nil, string(body))
	if err != nil {
		return body
	}
	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return body
	}
	return buf.Bytes()
}

// lexerForMime mirrors output/formatters/colors.py's get_lexer: try the
// full MIME type, then subtype (splitting a "+suffix" structured syntax
// suffix into its own candidates), then fall back to "json" if the
// subtype mentions it at all.
func lexerForMime(mime string, body []byte) chroma.Lexer {
	if mime == "" {
		if l := lexers.Analyse(string(body)); l != nil {
			return l
		}
		return nil
	}
	typ, subtype, ok := strings.Cut(mime, "/")
	if !ok {
		return nil
	}

	mimeCandidates := []string{mime}
	var nameCandidates []string
	if base, suffix, hasSuffix := strings.Cut(subtype, "+"); hasSuffix {
		nameCandidates = append(nameCandidates, base, suffix)
		mimeCandidates = append(mimeCandidates, typ+"/"+base, typ+"/"+suffix)
	} else {
		nameCandidates = append(nameCandidates, subtype)
	}
	if strings.Contains(subtype, "json") {
		nameCandidates = append(nameCandidates, "json")
	}

	for _, m := range mimeCandidates {
		if l := lexers.MatchMimeType(m); l != nil {
			return l
		}
	}
	for _, n := range nameCandidates {
		if l := lexers.Get(n); l != nil {
			return l
		}
	}
	return nil
}
