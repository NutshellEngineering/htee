package request

import (
	"regexp"
	"strings"
)

var urlSchemeRe = regexp.MustCompile(`^[a-z][a-z0-9.+-]*://`)

// NormalizeURL applies httpie's URL normalization rules, plus htee's own
// host-aware scheme default:
//   - a leading "://" is stripped (paste-URL shortcut: "http ://pie.dev" -> "http://pie.dev")
//   - if there's still no scheme, curl-style localhost shorthand (":8080/foo") becomes
//     "<scheme>://localhost:8080/foo"
//   - otherwise "<scheme>://" is prepended
//
// defaultScheme, if non-empty, is used uniformly (an explicit
// --default-scheme). If empty, the scheme is chosen per host: "http" for
// localhost/127.0.0.0/8/::1, "https" for everything else - on the
// assumption that a bare hostname typically means a real, TLS-serving
// host, while a bare local address typically means a plaintext dev server.
//
// Mirrors HTTPieArgumentParser._process_url in httpie/cli/argparser.py,
// with the host-aware default as htee's own addition.
func NormalizeURL(raw, defaultScheme string) string {
	raw = strings.TrimPrefix(raw, "://")
	if urlSchemeRe.MatchString(raw) {
		return raw
	}
	if port, rest, ok := localhostShorthand(raw); ok {
		scheme := schemeFor(defaultScheme, "localhost")
		if port != "" {
			return scheme + "://localhost:" + port + rest
		}
		return scheme + "://localhost" + rest
	}
	scheme := schemeFor(defaultScheme, hostOf(raw))
	return scheme + "://" + raw
}

// schemeFor returns defaultScheme unchanged if it's non-empty (an explicit
// --default-scheme, applied uniformly); otherwise it auto-selects "http"
// for a local host or "https" for anything else.
func schemeFor(defaultScheme, host string) string {
	if defaultScheme != "" {
		return defaultScheme
	}
	if isLocalHost(host) {
		return "http"
	}
	return "https"
}

// hostOf extracts the host (without userinfo, port, or IPv6 brackets) from
// the authority section - the part of a not-yet-schemed URL string before
// its first "/" or "?" - for scheme auto-selection by schemeFor/isLocalHost.
func hostOf(raw string) string {
	authority := raw
	if i := strings.IndexAny(raw, "/?"); i >= 0 {
		authority = raw[:i]
	}
	if i := strings.LastIndex(authority, "@"); i >= 0 {
		authority = authority[i+1:]
	}
	if strings.HasPrefix(authority, "[") {
		if i := strings.Index(authority, "]"); i >= 0 {
			return strings.ToLower(authority[1:i])
		}
		return strings.ToLower(authority)
	}
	if i := strings.LastIndex(authority, ":"); i >= 0 {
		authority = authority[:i]
	}
	return strings.ToLower(authority)
}

// isLocalHost reports whether host (as returned by hostOf, already
// lowercased) refers to the local machine: "localhost", the IPv4 loopback
// range 127.0.0.0/8, or the IPv6 loopback address.
func isLocalHost(host string) bool {
	if host == "localhost" || host == "::1" {
		return true
	}
	return strings.HasPrefix(host, "127.")
}

// localhostShorthand recognizes curl's `:PORT/path` shorthand for
// `localhost:PORT/path`, e.g. ":8080/foo" or ":/foo" (no port). Mirrors the
// Python regex `^:(?!:)(\d*)(/?.*)$` (rewritten without lookahead, since
// Go's RE2 doesn't support it).
func localhostShorthand(url string) (port, rest string, ok bool) {
	if !strings.HasPrefix(url, ":") || strings.HasPrefix(url, "::") {
		return "", "", false
	}
	remainder := url[1:]
	i := 0
	for i < len(remainder) && remainder[i] >= '0' && remainder[i] <= '9' {
		i++
	}
	return remainder[:i], remainder[i:], true
}
