package output

import "strconv"

// Hand-rolled ANSI colorizer for the request/status line and headers.
// httpie uses Pygments' HttpLexer + TerminalFormatter for this under
// --style=auto (and a dedicated SimplifiedHTTPLexer otherwise); neither
// maps cleanly onto chroma's generic lexer set (per the plan), so this
// mirrors the same visual result directly with basic SGR codes, which
// - like httpie's "auto" style - just follows the terminal's own palette.
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiCyan   = "\033[36m"
	ansiGreen  = "\033[1;32m"
	ansiYellow = "\033[1;33m"
	ansiOrange = "\033[1;33m" // no distinct basic-ANSI orange; reuse yellow
	ansiRed    = "\033[1;31m"
	ansiDim    = "\033[2m"
)

func methodColor(method string) string {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return ansiGreen
	case "POST":
		return ansiYellow
	case "PUT", "PATCH":
		return ansiOrange
	case "DELETE":
		return ansiRed
	default:
		return ansiBold
	}
}

func statusColor(code int) string {
	switch {
	case code < 200:
		return ansiCyan
	case code < 300:
		return ansiGreen
	case code < 400:
		return ansiYellow
	default:
		return ansiRed
	}
}

// colorizeRequestLine returns the ANSI-colored "METHOD target PROTO" line.
func colorizeRequestLine(method, target, proto string) string {
	return methodColor(method) + method + ansiReset + " " + target + " " + ansiDim + proto + ansiReset
}

// colorizeStatusLine returns the ANSI-colored "PROTO code reason" line.
func colorizeStatusLine(proto string, code int, reason string) string {
	return ansiDim + proto + ansiReset + " " + statusColor(code) + strconv.Itoa(code) + " " + reason + ansiReset
}

// colorizeHeaderLine returns the ANSI-colored "Name: value" header line.
func colorizeHeaderLine(name, value string) string {
	return ansiCyan + name + ansiReset + ": " + value
}
