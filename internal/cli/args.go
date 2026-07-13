package cli

import (
	"regexp"
	"strings"
)

var bareMethodRe = regexp.MustCompile(`^[a-zA-Z]+$`)

// ResolvePositionals splits the leftover positional args (after flags are
// stripped) into (method, url, rawItems).
//
// For a preset method (verbPreset != ""), there's no METHOD positional at
// all: args[0] is always the URL.
//
// For `ht` (verbPreset == ""), METHOD is an optional positional that
// argparse-style greedily takes args[0] whenever 2+ positionals are given
// (regardless of whether it "looks like" a method) - httpie then corrects
// for this after the fact: if args[0] doesn't match ^[a-zA-Z]+$, it wasn't
// really a method, so it's reinterpreted as the URL and args[1] becomes the
// first ITEM instead. With exactly 1 positional, it's unambiguously just
// the URL (method left unresolved, "" here - the caller infers GET/POST
// from whether the request has body data).
//
// Mirrors HTTPieArgumentParser._guess_method in httpie/cli/argparser.py.
func ResolvePositionals(verbPreset string, args []string) (method, url string, rawItems []string) {
	if verbPreset != "" {
		return verbPreset, args[0], args[1:]
	}
	if len(args) == 1 {
		return "", args[0], nil
	}
	candidate := args[0]
	if !bareMethodRe.MatchString(candidate) {
		items := append([]string{args[1]}, args[2:]...)
		return "", candidate, items
	}
	return strings.ToUpper(candidate), args[1], args[2:]
}
