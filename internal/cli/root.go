// Package cli wires the cobra command for the ht binary onto one shared
// flag set, item-syntax parser, request builder, auth resolver, transport,
// and output renderer.
package cli

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	"app.getnutshell/htee/internal/auth"
	"app.getnutshell/htee/internal/itemsyntax"
	"app.getnutshell/htee/internal/message"
	"app.getnutshell/htee/internal/netrc"
	"app.getnutshell/htee/internal/output"
	"app.getnutshell/htee/internal/request"
	"app.getnutshell/htee/internal/transport"
)

// Config selects the command's method-presetting behavior.
type Config struct {
	// VerbPreset is "" for the `ht` binary (METHOD is a positional
	// argument), or a fixed HTTP method ("GET", "POST", ...) to preset it,
	// used by tests to exercise per-verb behavior without a separate binary.
	VerbPreset string
}

// Execute builds and runs the cobra command for this binary and returns the
// process exit code.
func Execute(cfg Config) int {
	cmd := NewRootCommand(cfg)
	if err := cmd.Execute(); err != nil {
		if ue, ok := err.(*usageError); ok {
			fmt.Fprintf(os.Stderr,
				"usage:\n    %s\n\nerror:\n    %s\n\nfor more information:\n    run '%s --help'\n",
				ue.usage, ue.message, ue.progName)
		} else {
			fmt.Fprintln(os.Stderr, progNameFor(cfg)+":", err)
		}
		return 1
	}
	return 0
}

// progNameFor returns the binary name used in usage text and error prefixes:
// "ht" for the METHOD-positional binary, or the lowercased verb for a
// preset-method configuration (e.g. "get", "post").
func progNameFor(cfg Config) string {
	if cfg.VerbPreset == "" {
		return "ht"
	}
	return strings.ToLower(cfg.VerbPreset)
}

// NewRootCommand builds the cobra command for the given binary config.
func NewRootCommand(cfg Config) *cobra.Command {
	var flags sharedFlags

	progName := progNameFor(cfg)
	use := "ht [METHOD] URL [ITEM ...]"
	short := "ht - a first-class terminal HTTP client"
	if cfg.VerbPreset != "" {
		use = progName + " URL [ITEM ...]"
		short = fmt.Sprintf("%s - issue an HTTP %s request", progName, cfg.VerbPreset)
	}

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return &usageError{
					usage:    use,
					progName: progName,
					message:  "the following arguments are required: URL",
				}
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd, cfg, &flags, args)
		},
	}
	registerSharedFlags(cmd, &flags)
	return cmd
}

func run(cmd *cobra.Command, cfg Config, flags *sharedFlags, args []string) error {
	method, rawURL, rawItems := ResolvePositionals(cfg.VerbPreset, args)

	items := make([]itemsyntax.KeyValueArg, 0, len(rawItems))
	for _, raw := range rawItems {
		kv, err := itemsyntax.ParseItem(raw, itemsyntax.AllItemSeparators)
		if err != nil {
			return err
		}
		items = append(items, kv)
	}

	stdinBody, hasStdin := readStdin(flags.IgnoreStdin)

	if method == "" {
		method = inferMethod(items, flags.Raw != "" || hasStdin)
	}

	built, err := request.Build(request.Options{
		Method:        method,
		URL:           rawURL,
		DefaultScheme: flags.DefaultScheme,
		Items:         items,
		Form:          flags.Form,
		Multipart:     flags.Multipart,
		Boundary:      flags.Boundary,
		HasRaw:        flags.Raw != "",
		Raw:           flags.Raw,
		HasStdin:      hasStdin,
		Stdin:         stdinBody,
	})
	if err != nil {
		return err
	}

	viper.SetEnvPrefix("HT")
	viper.AutomaticEnv()
	envAuthToken := viper.GetString("AUTH")
	if envAuthToken == "" {
		envAuthToken = os.Getenv("AUTH_TOKEN")
	}

	var netrcUserinfo string
	if !flags.IgnoreNetrc {
		if user, pass, ok := netrc.Lookup(built.Request.URL.Hostname()); ok {
			netrcUserinfo = user + ":" + pass
		}
	}

	authOpts := auth.Options{
		Explicit:      flags.Auth != "",
		AuthType:      auth.Type(flags.AuthType),
		Credentials:   flags.Auth,
		URLUserinfo:   built.Request.URL.User.String(),
		EnvAuthToken:  envAuthToken,
		NetrcUserinfo: netrcUserinfo,
	}
	authDecision, err := auth.Resolve(authOpts)
	if err != nil {
		return err
	}
	if authDecision.Applied && !authDecision.Digest {
		built.Request.Header.Set("Authorization", authDecision.HeaderValue)
		built.HeaderOrder = append(built.HeaderOrder, "Authorization")
	}

	stdoutIsTTY := term.IsTerminal(int(os.Stdout.Fd()))

	printFlags, showAll := message.Resolve(message.DefaultOptions{
		ExplicitPrint: flags.Print,
		HasExplicit:   flags.Print != "",
		HeadersOnly:   flags.Headers,
		BodyOnly:      flags.Body,
		MetaOnly:      flags.Meta,
		VerboseCount:  flags.Verbose,
		All:           flags.All,
		Offline:       flags.Offline,
	})
	outOpts, err := resolveOutputOptions(cmd, flags, stdoutIsTTY)
	if err != nil {
		return err
	}

	reqMsg := message.FromRequest(built.Request, built.Body, built.HeaderOrder)
	output.RenderRequest(cmd.OutOrStdout(), reqMsg, printFlags, outOpts)

	if flags.Offline {
		return nil
	}

	baseTransport, err := resolveHTTPTransport(flags)
	if err != nil {
		return err
	}
	transportRT := auth.TransportFor(baseTransport, authDecision)
	client := transport.New()
	client.HTTP.Transport = transportRT
	client.HTTP.Timeout = time.Duration(flags.Timeout * float64(time.Second))

	chain, err := client.SendFollowing(built.Request, built.Body, built.HeaderOrder, transport.FollowOptions{
		Follow:       flags.Follow,
		MaxRedirects: flags.MaxRedirects,
	})
	if err != nil {
		return err
	}

	for i, hop := range chain.Hops {
		isFinal := i == len(chain.Hops)-1
		if i > 0 {
			hopReqMsg := message.FromRequest(hop.Request, hop.Body, hop.HeaderOrder)
			output.RenderRequest(cmd.OutOrStdout(), hopReqMsg, printFlags, outOpts)
		}
		if !isFinal && !showAll {
			continue
		}

		printedRequest := printFlags.ReqHeaders || printFlags.ReqBody
		printsResponse := printFlags.RespHeaders || printFlags.RespBody || printFlags.RespMeta
		if printedRequest && printsResponse {
			fmt.Fprint(cmd.OutOrStdout(), message.MessageSeparator)
		}

		respMsg := message.FromResponse(hop.Result.Response, hop.Result.Body, hop.Result.Elapsed.Seconds())
		output.RenderResponse(cmd.OutOrStdout(), respMsg, printFlags, outOpts)
	}

	return nil
}

// resolveHTTPTransport builds the *http.Transport carrying flags.Verify/
// SSLVersion/Ciphers/Cert/CertKey/CertKeyPass's TLS configuration. It's the
// `base` handed to auth.TransportFor, so auth (e.g. digest) wraps on top of
// TLS/proxy settings rather than replacing them.
func resolveHTTPTransport(flags *sharedFlags) (*http.Transport, error) {
	tlsCfg, err := transport.BuildTLSConfig(transport.TLSOptions{
		Verify:      flags.Verify,
		SSLVersion:  flags.SSLVersion,
		Ciphers:     flags.Ciphers,
		CertFile:    flags.Cert,
		CertKeyFile: flags.CertKey,
		CertKeyPass: flags.CertKeyPass,
	})
	if err != nil {
		return nil, err
	}

	t := http.DefaultTransport.(*http.Transport).Clone()
	t.TLSClientConfig = tlsCfg

	proxyFn, err := transport.ProxyFunc(flags.Proxy)
	if err != nil {
		return nil, err
	}
	if proxyFn != nil {
		t.Proxy = proxyFn
	}

	return t, nil
}

// resolveOutputOptions builds the output.Options controlling
// --pretty/--style/--format-options/--response-mime/--response-charset/
// --stream, validating the style name and format-options tokens up front.
func resolveOutputOptions(cmd *cobra.Command, flags *sharedFlags, stdoutIsTTY bool) (output.Options, error) {
	pretty, err := output.ResolvePretty(flags.Pretty, cmd.Flags().Changed("pretty"), stdoutIsTTY)
	if err != nil {
		return output.Options{}, err
	}
	if !output.ValidStyle(flags.Style) {
		return output.Options{}, fmt.Errorf("invalid --style %q", flags.Style)
	}
	formatOpts, err := output.ParseFormatOptions(flags.FormatOptions)
	if err != nil {
		return output.Options{}, err
	}
	return output.Options{
		Pretty:          pretty,
		Style:           flags.Style,
		FormatOptions:   formatOpts,
		ResponseMime:    flags.ResponseMime,
		ResponseCharset: flags.ResponseCharset,
		Stream:          flags.Stream,
	}, nil
}

// inferMethod infers GET vs POST when METHOD was omitted (the `ht`
// binary only): POST if the request has body data, else GET. Mirrors the
// tail end of HTTPieArgumentParser._guess_method.
func inferMethod(items []itemsyntax.KeyValueArg, hasOtherData bool) string {
	if hasOtherData {
		return "POST"
	}
	for _, it := range items {
		if itemsyntax.GroupDataItems[it.Sep] {
			return "POST"
		}
	}
	return "GET"
}

// readStdin reads piped stdin data (when stdin is not a terminal), used as
// the raw request body. Returns (nil, false) when stdin is a terminal or
// --ignore-stdin was given.
func readStdin(ignore bool) ([]byte, bool) {
	if ignore {
		return nil, false
	}
	if term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, false
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil || len(data) == 0 {
		return nil, false
	}
	return data, true
}
