// Package auth resolves and applies request authentication: httpie's
// -a/--auth + -A/--auth-type (basic/digest/bearer), URL-embedded userinfo,
// and htee's own env var fallback (auto Bearer injection) - HT_AUTH, or
// AUTH_TOKEN if that's what's set.
//
// Static schemes (basic/bearer/URL-userinfo/env token/netrc) resolve to a
// concrete header value up front via Resolve, so callers can set it
// directly on the request that's about to be displayed *and* sent - one
// source of truth, so what's printed always matches what's on the wire.
// Digest auth stays dynamic (its header can't be known until a 401
// challenge is seen), so it's still applied via an http.RoundTripper
// decorator through TransportFor.
package auth

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

// Type is an authentication scheme, selected via -A/--auth-type.
type Type string

const (
	TypeBasic  Type = "basic"
	TypeDigest Type = "digest"
	TypeBearer Type = "bearer"
)

// Options configures auth resolution.
type Options struct {
	Explicit    bool   // -a/--auth was given on the command line
	AuthType    Type   // -A/--auth-type; defaults to TypeBasic
	Credentials string // raw -a value: "user:pass", "user", or a bearer token

	URLUserinfo string // "user:pass" (or "user") embedded in the request URL, if any

	EnvAuthToken string // HT_AUTH (or AUTH_TOKEN) env var value, if set

	NetrcUserinfo string // "user:pass" resolved from .netrc, if any (skipped when --ignore-netrc)
}

// Decision is the resolved outcome of applying auth precedence: either a
// concrete "Authorization" header value (basic/bearer/URL-userinfo/
// env token/netrc), or a request for dynamic digest auth (whose header
// can only be computed after seeing the server's challenge).
type Decision struct {
	Applied bool // some auth scheme matched

	// Set when Applied && !Digest: the literal "Authorization" header value.
	HeaderValue string

	// Set when Applied && Digest: credentials for the digest challenge/response
	// round-trip, applied via TransportFor rather than a static header.
	Digest     bool
	DigestUser string
	DigestPass string
}

// Resolve computes the auth Decision for opts, in precedence order:
// explicit -a/-A wins; else URL-embedded userinfo (as basic auth); else
// the env token (htee-specific: `Authorization: Bearer $token`, from
// HT_AUTH or AUTH_TOKEN); else a matching .netrc entry (as basic auth);
// else no auth applies.
func Resolve(opts Options) (Decision, error) {
	if opts.Explicit {
		return resolveExplicit(opts)
	}
	if opts.URLUserinfo != "" {
		user, pass, _ := strings.Cut(opts.URLUserinfo, ":")
		return Decision{Applied: true, HeaderValue: basicHeader(user, pass)}, nil
	}
	if opts.EnvAuthToken != "" {
		return Decision{Applied: true, HeaderValue: "Bearer " + opts.EnvAuthToken}, nil
	}
	if opts.NetrcUserinfo != "" {
		user, pass, _ := strings.Cut(opts.NetrcUserinfo, ":")
		return Decision{Applied: true, HeaderValue: basicHeader(user, pass)}, nil
	}
	return Decision{}, nil
}

func resolveExplicit(opts Options) (Decision, error) {
	authType := opts.AuthType
	if authType == "" {
		authType = TypeBasic
	}
	switch authType {
	case TypeBearer:
		return Decision{Applied: true, HeaderValue: "Bearer " + opts.Credentials}, nil
	case TypeBasic, TypeDigest:
		user, pass, hasPass := strings.Cut(opts.Credentials, ":")
		if !hasPass {
			var err error
			pass, err = PromptPassword(fmt.Sprintf("password for %s: ", user))
			if err != nil {
				return Decision{}, err
			}
		}
		if authType == TypeDigest {
			return Decision{Applied: true, Digest: true, DigestUser: user, DigestPass: pass}, nil
		}
		return Decision{Applied: true, HeaderValue: basicHeader(user, pass)}, nil
	default:
		return Decision{}, fmt.Errorf("invalid --auth-type %q", authType)
	}
}

// basicHeader builds a `Basic <base64(user:pass)>` header value, matching
// http.Request.SetBasicAuth's encoding.
func basicHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

// TransportFor returns an http.RoundTripper applying d on top of base: a
// digest challenge/response wrapper for dynamic digest auth, a static
// Authorization-header setter for everything else, or base unchanged if
// no auth applies.
func TransportFor(base http.RoundTripper, d Decision) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if !d.Applied {
		return base
	}
	if d.Digest {
		return digestTransport(base, d.DigestUser, d.DigestPass)
	}
	return headerRoundTripper{base: base, set: func(req *http.Request) {
		req.Header.Set("Authorization", d.HeaderValue)
	}}
}

// Wrap resolves opts and returns an http.RoundTripper applying the result
// on top of base. Kept for callers that only need a transport (e.g. tests
// exercising precedence purely over the wire); the CLI itself uses Resolve
// directly so the same decision can also be reflected in displayed output.
func Wrap(base http.RoundTripper, opts Options) (http.RoundTripper, error) {
	d, err := Resolve(opts)
	if err != nil {
		return nil, err
	}
	return TransportFor(base, d), nil
}

// headerRoundTripper wraps base, applying set to a cloned request before
// forwarding it - used for auth schemes that just need one header set.
type headerRoundTripper struct {
	base http.RoundTripper
	set  func(*http.Request)
}

func (h headerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	h.set(clone)
	return h.base.RoundTrip(clone)
}
