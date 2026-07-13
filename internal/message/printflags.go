package message

import "strings"

// PrintFlags selects which parts of a request/response pair to render,
// mirroring httpie's -p/--print letters: H=request headers, B=request body,
// h=response headers, b=response body, m=response metadata (timing).
type PrintFlags struct {
	ReqHeaders  bool
	ReqBody     bool
	RespHeaders bool
	RespBody    bool
	RespMeta    bool
}

// ParsePrintString builds PrintFlags from a raw --print argument, e.g. "hb".
func ParsePrintString(s string) PrintFlags {
	return PrintFlags{
		ReqHeaders:  strings.Contains(s, "H"),
		ReqBody:     strings.Contains(s, "B"),
		RespHeaders: strings.Contains(s, "h"),
		RespBody:    strings.Contains(s, "b"),
		RespMeta:    strings.Contains(s, "m"),
	}
}

// DefaultOptions captures the inputs needed to resolve the default print
// flags when the user didn't pass an explicit --print/-h/-b string.
type DefaultOptions struct {
	ExplicitPrint string
	HasExplicit   bool
	HeadersOnly   bool // -h
	BodyOnly      bool // -b
	MetaOnly      bool // -m
	VerboseCount  int  // -v repeated
	All           bool // --all (independent of -v)
	Offline       bool
}

// Resolve implements htee's print-flag defaulting precedence: explicit
// --print > -h/-b/-m shortcuts > -vv > --offline > default. Unlike httpie
// (whose bare default is TTY-dependent: "hb" interactive, "b" redirected),
// htee's own default is always full request+response headers+body (as if
// -v were given), since the user wants verbose the default behavior.
// Returns the resolved flags and whether intermediate (redirect/
// auth-preflight) messages should also be printed.
func Resolve(opts DefaultOptions) (flags PrintFlags, all bool) {
	all = opts.All
	switch {
	case opts.HasExplicit:
		return ParsePrintString(opts.ExplicitPrint), all
	case opts.HeadersOnly:
		return ParsePrintString("h"), all
	case opts.BodyOnly:
		return ParsePrintString("b"), all
	case opts.MetaOnly:
		return ParsePrintString("m"), all
	case opts.VerboseCount >= 2:
		return ParsePrintString("HBhbm"), true
	case opts.Offline:
		return ParsePrintString("HB"), all
	default:
		return ParsePrintString("HBhb"), true
	}
}
