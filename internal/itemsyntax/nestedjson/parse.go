package nestedjson

import (
	"slices"
	"strconv"
)

// Parse parses a bracket-path key string into its sequence of Path
// segments. Mirrors httpie/cli/nested_json/parse.py:parse.
//
// Grammar:
//
//	start: root_path path*
//	root_path: (literal | index_path | append_path)?
//	literal: TEXT | NUMBER
//	path: key_path | index_path | append_path
//	key_path: LEFT_BRACKET TEXT RIGHT_BRACKET
//	index_path: LEFT_BRACKET NUMBER RIGHT_BRACKET
//	append_path: LEFT_BRACKET RIGHT_BRACKET
func Parse(source string) ([]Path, error) {
	p := &parser{source: source, tokens: tokenize(source)}
	root, err := p.parseRoot()
	if err != nil {
		return nil, err
	}
	paths := []Path{root}
	for p.canAdvance() {
		path, err := p.parsePath()
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

type parser struct {
	source string
	tokens []Token
	cursor int
}

func (p *parser) canAdvance() bool { return p.cursor < len(p.tokens) }

// expect consumes and returns the next token if its kind is one of kinds,
// otherwise returns a SyntaxError describing what was expected.
func (p *parser) expect(kinds ...TokenKind) (Token, error) {
	var tok Token
	var haveTok bool
	if p.canAdvance() {
		tok = p.tokens[p.cursor]
		p.cursor++
		haveTok = true
		if slices.Contains(kinds, tok.Kind) {
			return tok, nil
		}
	} else if len(p.tokens) > 0 {
		last := p.tokens[len(p.tokens)-1]
		tok = Token{Start: last.End, End: last.End + 1}
		haveTok = true
	}

	names := make([]string, len(kinds))
	for i, k := range kinds {
		names[i] = k.String()
	}
	suffix := names[0]
	if len(names) > 1 {
		suffix = ""
		for i, n := range names {
			if i == len(names)-1 {
				suffix += " or " + n
			} else {
				if i > 0 {
					suffix += ", "
				}
				suffix += n
			}
		}
	}
	message := "Expecting " + suffix
	if haveTok {
		return Token{}, newSyntaxError(p.source, &tok, message)
	}
	return Token{}, newSyntaxError(p.source, nil, message)
}

func (p *parser) parseRoot() (Path, error) {
	if !p.canAdvance() {
		return Path{Kind: ActionKey, KeyStr: emptyString, IsRoot: true}, nil
	}

	tok, err := p.expect(TokenText, TokenNumber, TokenLeftBracket)
	if err != nil {
		return Path{}, err
	}
	pathTokens := []Token{tok}

	switch tok.Kind {
	case TokenText:
		return Path{Kind: ActionKey, KeyStr: tok.Text, Tokens: pathTokens, IsRoot: true}, nil
	case TokenNumber:
		// A bare (unbracketed) root literal is always a KEY, even if it
		// looks numeric - only a bracketed root ([N]) is an array index.
		return Path{Kind: ActionKey, KeyStr: strconv.Itoa(tok.Num), Tokens: pathTokens, IsRoot: true}, nil
	case TokenLeftBracket:
		inner, err := p.expect(TokenNumber, TokenRightBracket)
		if err != nil {
			return Path{}, err
		}
		pathTokens = append(pathTokens, inner)
		if inner.Kind == TokenNumber {
			closeTok, err := p.expect(TokenRightBracket)
			if err != nil {
				return Path{}, err
			}
			pathTokens = append(pathTokens, closeTok)
			return Path{Kind: ActionIndex, Index: inner.Num, Tokens: pathTokens, IsRoot: true}, nil
		}
		return Path{Kind: ActionAppend, Tokens: pathTokens, IsRoot: true}, nil
	case TokenRightBracket:
		// unreachable: the expect() call above only allows TokenText,
		// TokenNumber, or TokenLeftBracket.
	}
	panic("unreachable")
}

func (p *parser) parsePath() (Path, error) {
	lb, err := p.expect(TokenLeftBracket)
	if err != nil {
		return Path{}, err
	}
	pathTokens := []Token{lb}

	tok, err := p.expect(TokenText, TokenNumber, TokenRightBracket)
	if err != nil {
		return Path{}, err
	}
	pathTokens = append(pathTokens, tok)

	switch tok.Kind {
	case TokenRightBracket:
		return Path{Kind: ActionAppend, Tokens: pathTokens}, nil
	case TokenText:
		rb, err := p.expect(TokenRightBracket)
		if err != nil {
			return Path{}, err
		}
		pathTokens = append(pathTokens, rb)
		return Path{Kind: ActionKey, KeyStr: tok.Text, Tokens: pathTokens}, nil
	case TokenNumber:
		rb, err := p.expect(TokenRightBracket)
		if err != nil {
			return Path{}, err
		}
		pathTokens = append(pathTokens, rb)
		return Path{Kind: ActionIndex, Index: tok.Num, Tokens: pathTokens}, nil
	case TokenLeftBracket:
		// unreachable: the expect() call above only allows TokenText,
		// TokenNumber, or TokenRightBracket.
	}
	panic("unreachable")
}
