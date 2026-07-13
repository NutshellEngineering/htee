package auth

import (
	"net/http"

	"github.com/icholy/digest"
)

// digestTransport wraps base with RFC 7616 digest auth, retrying a 401 with
// the digest challenge response.
func digestTransport(base http.RoundTripper, user, pass string) http.RoundTripper {
	return &digest.Transport{
		Username:  user,
		Password:  pass,
		Transport: base,
	}
}
