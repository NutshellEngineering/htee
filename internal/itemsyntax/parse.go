package itemsyntax

import (
	"fmt"
	"sort"
	"strings"
)

// KeyValueArg is a parsed REQUEST_ITEM: `key<sep>value`.
type KeyValueArg struct {
	Key   string
	Value string
	Sep   Separator
	Orig  string
}

// ParseItem parses a single REQUEST_ITEM string using the given candidate
// separators. The earliest-starting separator wins; ties are broken in
// favor of the longest separator. Backslash-escaped separator characters are
// never treated as separators.
//
// Mirrors KeyValueArgType.__call__ in httpie/cli/argtypes.py.
func ParseItem(s string, separators []Separator) (KeyValueArg, error) {
	special := specialChars(separators)
	tokens := tokenize(s, special)

	byLenAsc := append([]Separator(nil), separators...)
	sort.SliceStable(byLenAsc, func(i, j int) bool { return len(byLenAsc[i]) < len(byLenAsc[j]) })

	for i, tok := range tokens {
		if tok.escaped {
			continue
		}
		found := map[int]Separator{}
		for _, sep := range byLenAsc {
			if pos := strings.Index(tok.text, string(sep)); pos != -1 {
				found[pos] = sep // longer separators (processed later) overwrite shorter ones at the same position
			}
		}
		if len(found) == 0 {
			continue
		}
		minPos := -1
		for pos := range found {
			if minPos == -1 || pos < minPos {
				minPos = pos
			}
		}
		sep := found[minPos]
		key := tok.text[:minPos]
		value := tok.text[minPos+len(sep):]
		key = joinTokens(tokens[:i]) + key
		value = value + joinTokens(tokens[i+1:])
		return KeyValueArg{Key: key, Value: value, Sep: sep, Orig: s}, nil
	}

	return KeyValueArg{}, fmt.Errorf("%q is not a valid value", s)
}
