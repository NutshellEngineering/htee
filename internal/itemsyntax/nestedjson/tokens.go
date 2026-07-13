// Package nestedjson implements httpie's nested-JSON bracket-path syntax:
// `person[name]=bob`, `person[addresses][0][street]=Main`, `tags[]=a`
// (append), and top-level arrays via a leading `[]`/`[0]` path.
//
// Mirrors httpie/cli/nested_json/{tokens,parse,interpret}.py.
package nestedjson

import "strconv"

// TokenKind identifies a lexical token kind in a bracket-path key string.
type TokenKind int

const (
	TokenText TokenKind = iota
	TokenNumber
	TokenLeftBracket
	TokenRightBracket
)

func (k TokenKind) String() string {
	switch k {
	case TokenText:
		return "TEXT"
	case TokenNumber:
		return "NUMBER"
	case TokenLeftBracket:
		return "["
	case TokenRightBracket:
		return "]"
	default:
		return "?"
	}
}

// Token is one lexical unit of a bracket-path key, with byte offsets into
// the original key string (used for error messages).
type Token struct {
	Kind  TokenKind
	Text  string // set when Kind == TokenText
	Num   int    // set when Kind == TokenNumber
	Start int
	End   int
}

// PathAction is what a Path segment does to the value it's applied to.
type PathAction int

const (
	ActionKey PathAction = iota
	ActionIndex
	ActionAppend
	ActionSet // terminal: the right-hand-side value itself
)

func (a PathAction) String() string {
	switch a {
	case ActionKey:
		return "key"
	case ActionIndex:
		return "index"
	case ActionAppend:
		return "append"
	case ActionSet:
		return "set"
	default:
		return "?"
	}
}

// Path is one parsed path segment: `[name]`, `[0]`, `[]`, a bare root key,
// or (for ActionSet) the terminal value itself.
type Path struct {
	Kind   PathAction
	KeyStr string // set when Kind == ActionKey
	Index  int    // set when Kind == ActionIndex
	Value  any    // set when Kind == ActionSet
	Tokens []Token
	IsRoot bool
}

// Reconstruct renders the path segment back to its source syntax, e.g.
// `[name]`, `[0]`, `[]`, or (for a root key) the bare key itself. Used to
// build human-readable "which key had the wrong type" error messages.
func (p Path) Reconstruct() string {
	switch p.Kind {
	case ActionKey:
		if p.IsRoot {
			return p.KeyStr
		}
		return "[" + p.KeyStr + "]"
	case ActionIndex:
		return "[" + strconv.Itoa(p.Index) + "]"
	case ActionAppend:
		return "[]"
	default:
		return ""
	}
}

// emptyString is the sentinel root key used to represent "no key at all",
// i.e. a bare `[]=v` / `[0]=v` item building a top-level JSON array.
const emptyString = ""

const backslash = '\\'

// specialChars are the characters `\` can escape within a bracket-path key:
// the brackets themselves and the backslash itself.
var specialChars = map[rune]bool{'[': true, ']': true, '\\': true}

var operators = map[rune]TokenKind{
	'[': TokenLeftBracket,
	']': TokenRightBracket,
}
