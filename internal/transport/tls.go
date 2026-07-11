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
