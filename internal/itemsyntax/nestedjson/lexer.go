package nestedjson

import "strconv"

// tokenize lexes a bracket-path key string into Tokens. Mirrors
// httpie/cli/nested_json/parse.py:tokenize.
func tokenize(source string) []Token {
	runes := []rune(source)
	var tokens []Token
	cursor := 0
	backslashes := 0
	var buffer []rune
	bufStart := 0

	sendBuffer := func() {
		if len(buffer) == 0 {
			return
		}
		value := string(buffer)
		kind := TokenText
		numVal := 0
		text := value
		if backslashes == 0 {
			if n, err := strconv.Atoi(value); err == nil {
				kind = TokenNumber
				numVal = n
			} else if unescaped, ok := checkEscapedInt(value); ok {
				kind = TokenText
				text = unescaped
			}
		}
		tokens = append(tokens, Token{Kind: kind, Text: text, Num: numVal, Start: bufStart, End: cursor})
		buffer = nil
		backslashes = 0
	}

	for cursor < len(runes) {
		ch := runes[cursor]
		if kind, ok := operators[ch]; ok {
			sendBuffer()
			tokens = append(tokens, Token{Kind: kind, Text: string(ch), Start: cursor, End: cursor + 1})
		} else if ch == backslash && cursor+1 < len(runes) {
			next := runes[cursor+1]
			if len(buffer) == 0 {
				bufStart = cursor
			}
			if specialChars[next] {
				backslashes++
			} else {
				buffer = append(buffer, ch)
			}
			buffer = append(buffer, next)
			cursor++
		} else {
			if len(buffer) == 0 {
				bufStart = cursor
			}
			buffer = append(buffer, ch)
		}
		cursor++
	}
	sendBuffer()
	return tokens
}

// checkEscapedInt reports whether value is a backslash followed by a valid
// integer (e.g. `\0`), in which case it was deliberately escaped to be kept
// as a literal string key rather than coerced into an array index. Returns
// the unescaped string (without the leading backslash) on success.
func checkEscapedInt(value string) (string, bool) {
	if len(value) == 0 || rune(value[0]) != backslash {
		return "", false
	}
	rest := value[1:]
	if _, err := strconv.Atoi(rest); err != nil {
		return "", false
	}
	return rest, true
}
