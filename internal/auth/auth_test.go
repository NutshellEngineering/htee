package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequest(t *testing.T, rt http.RoundTripper, url string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestEnvAuthTokenAppliesWhenNoExplicitAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{EnvAuthToken: "secrettoken"})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	if gotAuth != "Bearer secrettoken" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestExplicitAuthBeatsEnvAuthToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{
		Explicit:     true,
		AuthType:     TypeBasic,
		Credentials:  "alice:s3cret",
		EnvAuthToken: "secrettoken", // should be ignored
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := doRequest(t, rt, srv.URL)
	_ = resp
	if gotAuth == "Bearer secrettoken" {
		t.Fatalf("env auth token should not apply when -a is explicit, got %q", gotAuth)
	}
	user, pass, ok := parseBasicHeader(gotAuth)
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestURLUserinfoUsedWhenNoExplicitOrEnvAuthToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{URLUserinfo: "bob:hunter2"})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	user, pass, ok := parseBasicHeader(gotAuth)
	if !ok || user != "bob" || pass != "hunter2" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestNoAuthWhenNothingConfigured(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	if gotAuth != "" {
		t.Fatalf("expected no Authorization header, got %q", gotAuth)
	}
}

func TestBearerAuthType(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{
		Explicit:    true,
		AuthType:    TypeBearer,
		Credentials: "mytoken",
	})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	if gotAuth != "Bearer mytoken" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestNetrcUsedWhenNothingElseConfigured(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{NetrcUserinfo: "carol:swordfish"})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	user, pass, ok := parseBasicHeader(gotAuth)
	if !ok || user != "carol" || pass != "swordfish" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestEnvAuthTokenBeatsNetrc(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	rt, err := Wrap(http.DefaultTransport, Options{
		EnvAuthToken:  "secrettoken",
		NetrcUserinfo: "carol:swordfish", // should be ignored
	})
	if err != nil {
		t.Fatal(err)
	}
	doRequest(t, rt, srv.URL)
	if gotAuth != "Bearer secrettoken" {
		t.Fatalf("expected env auth token to beat netrc, got %q", gotAuth)
	}
}

func parseBasicHeader(h string) (user, pass string, ok bool) {
	req := &http.Request{Header: http.Header{"Authorization": []string{h}}}
	return req.BasicAuth()
}
