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

	cipherIDs, err := resolveCiphers(opts.Ciphers)
	if err != nil {
		return nil, err
	}
	cfg.CipherSuites = cipherIDs

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

// resolveCiphers implements --ciphers: a colon- or comma-separated list of
// Go crypto/tls cipher suite names (e.g. TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256).
// Unlike httpie (which accepts OpenSSL cipher-list syntax via urllib3),
// there's no OpenSSL-name-to-Go-suite mapping in the stdlib, so names must
// match tls.CipherSuiteName exactly - see tls.CipherSuites()/
// tls.InsecureCipherSuites() for the full list.
func resolveCiphers(raw string) ([]uint16, error) {
	if raw == "" {
		return nil, nil
	}
	names := strings.FieldsFunc(raw, func(r rune) bool { return r == ':' || r == ',' })
	lookup := make(map[string]uint16)
	for _, cs := range tls.CipherSuites() {
		lookup[cs.Name] = cs.ID
	}
	for _, cs := range tls.InsecureCipherSuites() {
		lookup[cs.Name] = cs.ID
	}
	ids := make([]uint16, 0, len(names))
	for _, name := range names {
		id, ok := lookup[name]
		if !ok {
			return nil, fmt.Errorf("--ciphers: unknown cipher suite %q (expected a Go crypto/tls suite name, e.g. TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256)", name)
		}
		ids = append(ids, id)
	}
	return ids, nil
}
