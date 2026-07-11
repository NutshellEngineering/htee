// Package transport (this file) builds the *tls.Config used by the
// underlying http.Transport, mirroring httpie's SSL flag group
// (--verify/--ssl/--ciphers/--cert/--cert-key/--cert-key-pass) from
// httpie/ssl_.py's HTTPieHTTPSAdapter._create_ssl_context.
package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
)

// TLSOptions configures BuildTLSConfig, one field per SSL-group flag.
type TLSOptions struct {
	Verify      string // "yes"/"no"/"true"/"false", or a CA bundle file path; "" means "yes"
	SSLVersion  string // "", ssl2.3, ssl3, tls1, tls1.1, tls1.2, tls1.3
	Ciphers     string // colon- or comma-separated Go tls cipher suite names
	CertFile    string // --cert
	CertKeyFile string // --cert-key
	CertKeyPass string // --cert-key-pass
}

// BuildTLSConfig resolves opts into a *tls.Config for the client transport.
func BuildTLSConfig(opts TLSOptions) (*tls.Config, error) {
	cfg := &tls.Config{}

	insecure, caPool, err := resolveVerify(opts.Verify)
	if err != nil {
		return nil, err
	}
	cfg.InsecureSkipVerify = insecure
	cfg.RootCAs = caPool

	minV, maxV, err := resolveSSLVersion(opts.SSLVersion)
	if err != nil {
		return nil, err
	}
	cfg.MinVersion, cfg.MaxVersion = minV, maxV

	return cfg, nil
}

// resolveVerify implements --verify: "no"/"false" skips verification
// entirely; "yes"/"false"/"" (default) verifies against the system trust
// store; anything else is treated as a CA bundle file path.
func resolveVerify(raw string) (insecureSkipVerify bool, caBundle *x509.CertPool, err error) {
	switch strings.ToLower(raw) {
	case "", "yes", "true":
		return false, nil, nil
	case "no", "false":
		return true, nil, nil
	default:
		pemBytes, err := os.ReadFile(raw)
		if err != nil {
			return false, nil, fmt.Errorf("--verify: reading CA bundle %q: %w", raw, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pemBytes) {
			return false, nil, fmt.Errorf("--verify: no certificates found in CA bundle %q", raw)
		}
		return false, pool, nil
	}
}

var sslVersions = map[string]uint16{
	"tls1":   tls.VersionTLS10,
	"tls1.1": tls.VersionTLS11,
	"tls1.2": tls.VersionTLS12,
	"tls1.3": tls.VersionTLS13,
}

// resolveSSLVersion implements --ssl. "" and "ssl2.3" (httpie's "negotiate
// the highest mutually supported protocol") both return (0, 0): an unset
// range, letting crypto/tls pick its own default negotiation window.
// "ssl3" errors - Go's crypto/tls has never implemented SSLv3.
func resolveSSLVersion(raw string) (min, max uint16, err error) {
	switch raw {
	case "", "ssl2.3":
		return 0, 0, nil
	case "ssl3":
		return 0, 0, fmt.Errorf("--ssl=ssl3: SSLv3 is not supported (Go's crypto/tls has no SSLv3 support; the minimum available protocol is TLS 1.0)")
	}
	v, ok := sslVersions[raw]
	if !ok {
		return 0, 0, fmt.Errorf("invalid --ssl version %q (expected one of: ssl2.3, tls1, tls1.1, tls1.2, tls1.3)", raw)
	}
	return v, v, nil
}
