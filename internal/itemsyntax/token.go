package itemsyntax

import "strings"

// token is one piece of a tokenized REQUEST_ITEM string: either a plain
// literal run of characters, or a single escaped separator character (from
// e.g. `\=`). Mirrors httpie's `str` vs `Escaped(str)` token types.
type token struct {
	text    string
	escaped bool
}

// specialChars returns the set of runes that appear in any of the given
// separators - these are the only characters a backslash can escape.
func specialChars(separators []Separator) map[rune]bool {
	set := map[rune]bool{}
	for _, sep := range separators {
		for _, r := range string(sep) {
			set[r] = true
		}
	}
	return set
}

// tokenize splits s into a sequence of plain-text and escaped-character
// tokens. A backslash immediately followed by a special character escapes
// that character (removing it from separator consideration); a backslash
// followed by anything else is kept as a literal two-character sequence.
//
// Mirrors KeyValueArgType.tokenize in httpie/cli/argtypes.py.
func tokenize(s string, special map[rune]bool) []token {
	tokens := []token{{text: ""}}
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		if ch == '\\' {
			var next rune
			hasNext := i+1 < len(runes)
			if hasNext {
				next = runes[i+1]
			}
			if !hasNext || !special[next] {
				last := &tokens[len(tokens)-1]
				if hasNext {
					last.text += "\\" + string(next)
					i++
				} else {
					last.text += "\\"
				}
			} else {
				tokens = append(tokens, token{text: string(next), escaped: true}, token{text: ""})
				i++
			}
			continue
		}
		last := &tokens[len(tokens)-1]
		last.text += string(ch)
	}
	return tokens
}

func joinTokens(tokens []token) string {
	var b strings.Builder
	for _, t := range tokens {
		b.WriteString(t.text)
	}
	return b.String()
}
