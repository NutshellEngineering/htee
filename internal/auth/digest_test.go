package auth

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newDigestServer builds a minimal RFC 7616 (qop=auth, MD5) digest server
// that challenges every unauthenticated request and validates the
// client's computed response server-side, so the test actually proves the
// challenge/response round-trip works rather than just checking that some
// "Digest ..." header was sent.
func newDigestServer(t *testing.T, realm, nonce, username, password string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authz := r.Header.Get("Authorization")
		if authz == "" || !strings.HasPrefix(authz, "Digest ") {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm=%q, nonce=%q, qop="auth"`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		params := parseDigestHeader(authz)
		ha1 := md5hex(username + ":" + realm + ":" + password)
		ha2 := md5hex(r.Method + ":" + params["uri"])
		want := md5hex(strings.Join([]string{ha1, params["nonce"], params["nc"], params["cnonce"], params["qop"], ha2}, ":"))

		if params["username"] != username || params["response"] != want {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Digest realm=%q, nonce=%q, qop="auth"`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
}

func md5hex(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}

// parseDigestHeader parses `Digest key="value", key2=value2, ...` into a map.
func parseDigestHeader(header string) map[string]string {
	header = strings.TrimPrefix(header, "Digest ")
	out := map[string]string{}
	for part := range strings.SplitSeq(header, ",") {
		part = strings.TrimSpace(part)
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		out[k] = strings.Trim(v, `"`)
	}
	return out
}

func TestDigestAuthRoundTrip(t *testing.T) {
	srv := newDigestServer(t, "testrealm", "abc123nonce", "alice", "s3cret")
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{
		Explicit:    true,
		AuthType:    TypeDigest,
		Credentials: "alice:s3cret",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := doRequest(t, rt, srv.URL)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 after digest round-trip, got %d", resp.StatusCode)
	}
}

func TestDigestAuthWrongPasswordFails(t *testing.T) {
	srv := newDigestServer(t, "testrealm", "abc123nonce", "alice", "s3cret")
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{
		Explicit:    true,
		AuthType:    TypeDigest,
		Credentials: "alice:wrongpass",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := doRequest(t, rt, srv.URL)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong password, got %d", resp.StatusCode)
	}
}
