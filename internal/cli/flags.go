package cli

import (
	"github.com/spf13/cobra"

	"app.getnutshell/htee/internal/output"
)

// sharedFlags holds the pflag-bound values for the ht command, mirroring
// httpie's cli/definition.py flag set.
type sharedFlags struct {
	JSON      bool
	Form      bool
	Multipart bool
	Boundary  string
	Raw       string

	Print   string
	Headers bool
	Body    bool
	Meta    bool
	Verbose int
	All     bool
	Offline bool

	Follow       bool
	MaxRedirects int

	Verify      string
	SSLVersion  string
	Ciphers     string
	Cert        string
	CertKey     string
	CertKeyPass string

	Timeout float64
	Proxy   []string

	// MaxHeaders is parsed for httpie CLI compatibility but not enforced:
	// Go's net/http client has no equivalent to Python's
	// http.client._MAXHEADERS guard. See docs/superpowers/plans/
	// 2026-07-11-phase4-redirects-ssl.md's Global Constraints for why.
	MaxHeaders int

	DefaultScheme string

	Auth        string
	AuthType    string
	IgnoreNetrc bool

	IgnoreStdin bool

	Pretty          string
	Style           string
	FormatOptions   []string
	ResponseCharset string
	ResponseMime    string
	Stream          bool
}

// registerSharedFlags registers every flag onto cmd, and installs a bare
// `--help` (no `-h` shorthand, since httpie uses `-h` for "print response
// headers only").
func registerSharedFlags(cmd *cobra.Command, f *sharedFlags) {
	cmd.Flags().Bool("help", false, "help for "+cmd.Name())

	fl := cmd.Flags()
	fl.BoolVarP(&f.JSON, "json", "j", true, "(default) Serialize data items as a JSON object")
	fl.BoolVarP(&f.Form, "form", "f", false, "Serialize data items as form fields")
	fl.BoolVar(&f.Multipart, "multipart", false, "Always send as multipart/form-data")
	fl.StringVar(&f.Boundary, "boundary", "", "Custom multipart boundary (only with --form)")
	fl.StringVar(&f.Raw, "raw", "", "Pass raw request body, bypassing item parsing")

	fl.StringVarP(&f.Print, "print", "p", "", "String specifying what the output should contain: H,B,h,b,m")
	fl.BoolVarP(&f.Headers, "headers", "h", false, "Print only the response headers (shortcut for -p h)")
	fl.BoolVarP(&f.Body, "body", "b", false, "Print only the response body (shortcut for -p b)")
	fl.BoolVarP(&f.Meta, "meta", "m", false, "Print only the response metadata (shortcut for -p m)")
	fl.CountVarP(&f.Verbose, "verbose", "v", "Show intermediary requests and full request/response")
	fl.BoolVar(&f.All, "all", false, "Show any intermediary requests/responses")
	fl.BoolVar(&f.Offline, "offline", false, "Build the request and print it, but don't send it")

	fl.BoolVarP(&f.Follow, "follow", "F", false, "Follow 30x Location redirects")
	fl.IntVar(&f.MaxRedirects, "max-redirects", 30, "Maximum number of redirects to follow (with --follow); 0 means unlimited")

	fl.StringVar(&f.Verify, "verify", "yes", `If "no", skip SSL certificate verification; if a file path, use it as a CA bundle`)
	fl.StringVar(&f.SSLVersion, "ssl", "", "TLS protocol version to use: ssl2.3 (negotiate highest, default), tls1, tls1.1, tls1.2, or tls1.3")
	fl.StringVar(&f.Ciphers, "ciphers", "", "Colon- or comma-separated list of Go crypto/tls cipher suite names")
	fl.StringVar(&f.Cert, "cert", "", "Client-side SSL certificate file (PEM); may also contain the private key")
	fl.StringVar(&f.CertKey, "cert-key", "", "Private key for --cert, if not included in the cert file")
	fl.StringVar(&f.CertKeyPass, "cert-key-pass", "", "Passphrase for --cert-key, if it's encrypted (prompted for on a TTY if omitted)")

	fl.Float64Var(&f.Timeout, "timeout", 0, "Connection timeout in seconds; 0 means no timeout. Unlike httpie's inactivity-based timeout, this caps the whole request (connect through full body read)")
	fl.StringArrayVar(&f.Proxy, "proxy", nil, `Proxy as protocol:URL (e.g. http:http://localhost:8080); repeatable, or use protocol "all" as a catch-all`)
	fl.IntVar(&f.MaxHeaders, "max-headers", 0, "Accepted for httpie compatibility; not enforced (Go's HTTP client has no equivalent limit)")

	fl.StringVar(&f.DefaultScheme, "default-scheme", "https", "Default URL scheme when none is given")

	fl.StringVarP(&f.Auth, "auth", "a", "", "Credentials: user:pass, user, or a bearer token")
	fl.StringVarP(&f.AuthType, "auth-type", "A", "basic", "Auth type: basic, digest, or bearer")
	fl.BoolVar(&f.IgnoreNetrc, "ignore-netrc", false, "Ignore credentials from .netrc")

	fl.BoolVarP(&f.IgnoreStdin, "ignore-stdin", "I", false, "Do not read stdin for request data")

	fl.StringVar(&f.Pretty, "pretty", "", "Controls output processing: none, colors, format, or all (default: all on a TTY, none otherwise)")
	fl.StringVarP(&f.Style, "style", "s", output.AutoStyle, "Output coloring style (default: auto, follows the terminal's palette)")
	fl.Var(&rawAppender{target: &f.FormatOptions}, "format-options", "Controls output formatting, e.g. json.indent:2,json.sort_keys:false")
	fl.Var(&constAppender{target: &f.FormatOptions, value: output.SortedFormatOptionsString}, "sorted", "Re-enables all sorting options while formatting output")
	fl.Var(&constAppender{target: &f.FormatOptions, value: output.UnsortedFormatOptionsString}, "unsorted", "Disables all sorting while formatting output")
	fl.Var(&constAppender{target: &f.FormatOptions, value: output.UnsortedFormatOptionsString}, "no-sorted", "")
	fl.Var(&constAppender{target: &f.FormatOptions, value: output.SortedFormatOptionsString}, "no-unsorted", "")
	// constAppender takes no argument (like action='append_const'); pflag
	// only honors that for a plain Var flag via NoOptDefVal, not the
	// IsBoolFlag() method (that's only consulted for flags imported from
	// the stdlib "flag" package). Without this, `--unsorted --offline`
	// would swallow "--offline" as unsorted's value instead of leaving it
	// for the parser to see as its own flag.
	for _, name := range []string{"sorted", "unsorted", "no-sorted", "no-unsorted"} {
		fl.Lookup(name).NoOptDefVal = "true"
	}
	fl.Lookup("no-sorted").Hidden = true
	fl.Lookup("no-unsorted").Hidden = true
	fl.StringVar(&f.ResponseCharset, "response-charset", "", "Override the response encoding for terminal display, e.g. utf8, iso-8859-1")
	fl.StringVar(&f.ResponseMime, "response-mime", "", "Override the response mime type for coloring/formatting, e.g. application/json")
	fl.BoolVarP(&f.Stream, "stream", "S", false, "Always stream the response body, skipping body re-formatting")
}
