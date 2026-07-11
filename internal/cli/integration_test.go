package cli

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func runCLI(t *testing.T, verbPreset string, args ...string) (stdout string, err error) {
	t.Helper()
	cmd := NewRootCommand(Config{VerbPreset: verbPreset})
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return buf.String(), err
}

func TestGetBinaryPlainRequest(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	// Non-interactive stdout (as in this test) defaults to body-only output,
	// matching httpie; -v forces headers too so we can check the status line.
	out, err := runCLI(t, "GET", "-v", srv.URL+"/ping")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if gotMethod != "GET" || gotPath != "/ping" {
		t.Fatalf("server saw method=%q path=%q", gotMethod, gotPath)
	}
	if !strings.Contains(out, "200") || !strings.Contains(out, "hello") {
		t.Fatalf("output missing expected content: %s", out)
	}
}

func TestPostBinarySendsJSON(t *testing.T) {
	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotContentType = r.Header.Get("Content-Type")
		w.WriteHeader(201)
	}))
	defer srv.Close()

	_, err := runCLI(t, "POST", srv.URL, "foo=bar", "n:=1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotContentType != "application/json" {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if string(gotBody) != `{"foo":"bar","n":1}` {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestMethodInference(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
	}))
	defer srv.Close()

	// `ht URL foo=bar` (no explicit METHOD) should infer POST since foo=bar is body data.
	if _, err := runCLI(t, "", srv.URL, "foo=bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != "POST" {
		t.Fatalf("expected inferred POST, got %q", gotMethod)
	}

	// `ht URL` alone should infer GET.
	if _, err := runCLI(t, "", srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != "GET" {
		t.Fatalf("expected inferred GET, got %q", gotMethod)
	}

	// `ht PUT URL foo=bar` explicit method.
	if _, err := runCLI(t, "", "PUT", srv.URL, "foo=bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != "PUT" {
		t.Fatalf("expected explicit PUT, got %q", gotMethod)
	}
}

func TestHtAuthEnvVarInjectsBearer(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	t.Setenv("HT_AUTH", "supersecret")
	out, err := runCLI(t, "GET", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer supersecret" {
		t.Fatalf("Authorization sent to server = %q", gotAuth)
	}
	// The displayed request must show the exact same header that was sent
	// over the wire - regression test for a bug where the Authorization
	// header was applied to a cloned request at send time and so never
	// appeared in the rendered output, even though the server received it.
	if !strings.Contains(out, "Authorization: Bearer supersecret") {
		t.Fatalf("expected displayed request to include Authorization header, got:\n%s", out)
	}
}

func TestAuthTokenEnvVarUsedWhenHtAuthUnset(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	t.Setenv("AUTH_TOKEN", "fallbacktoken")
	if _, err := runCLI(t, "GET", srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer fallbacktoken" {
		t.Fatalf("Authorization sent to server = %q", gotAuth)
	}
}

func TestHtAuthBeatsAuthToken(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	t.Setenv("HT_AUTH", "supersecret")
	t.Setenv("AUTH_TOKEN", "fallbacktoken")
	if _, err := runCLI(t, "GET", srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer supersecret" {
		t.Fatalf("Authorization sent to server = %q", gotAuth)
	}
}

func TestExplicitAuthOverridesHtAuth(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	t.Setenv("HT_AUTH", "supersecret")
	if _, err := runCLI(t, "GET", "-a", "alice:s3cret", srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req := &http.Request{Header: http.Header{"Authorization": []string{gotAuth}}}
	user, pass, ok := req.BasicAuth()
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
}

func TestFormFlag(t *testing.T) {
	var gotBody []byte
	var gotContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		gotContentType = r.Header.Get("Content-Type")
	}))
	defer srv.Close()

	if _, err := runCLI(t, "POST", "-f", srv.URL, "foo=bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(gotContentType, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type = %q", gotContentType)
	}
	if string(gotBody) != "foo=bar" {
		t.Fatalf("body = %s", gotBody)
	}
}

func TestOfflineDoesNotSend(t *testing.T) {
	served := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	}))
	defer srv.Close()

	out, err := runCLI(t, "GET", "--offline", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if served {
		t.Fatal("server should not have been contacted in --offline mode")
	}
	if !strings.Contains(out, "GET") {
		t.Fatalf("expected request line in offline output: %s", out)
	}
}

// TestOfflineWithUnsortedDoesNotSend guards against a regression where
// --unsorted (registered as a custom pflag.Value, not BoolVar) swallowed
// the following "--offline" token as its own argument instead of leaving
// it for the parser, silently defaulting Offline to false and actually
// contacting the server.
func TestOfflineWithUnsortedDoesNotSend(t *testing.T) {
	served := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served = true
	}))
	defer srv.Close()

	out, err := runCLI(t, "POST", "--pretty=format", "--unsorted", "--offline", srv.URL, "b=2", "a=1")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if served {
		t.Fatal("server should not have been contacted in --offline mode")
	}
	// --unsorted disables json.sort_keys, so declaration order ("b" then "a") is kept.
	if !strings.Contains(out, `"b": "2"`) || strings.Index(out, `"b"`) > strings.Index(out, `"a"`) {
		t.Fatalf("expected unsorted body (b before a): %s", out)
	}
}

func TestSortedUnsortedFlagsOverrideInOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	// --sorted then --unsorted: unsorted applied last, wins.
	out, err := runCLI(t, "POST", "--pretty=format", "--offline", "--sorted", "--unsorted", srv.URL, "b=2", "a=1")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if strings.Index(out, `"b"`) > strings.Index(out, `"a"`) {
		t.Fatalf("expected unsorted (declaration order) to win when given last: %s", out)
	}

	// Reversed: sorted applied last, wins.
	out, err = runCLI(t, "POST", "--pretty=format", "--offline", "--unsorted", "--sorted", srv.URL, "b=2", "a=1")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if strings.Index(out, `"a"`) > strings.Index(out, `"b"`) {
		t.Fatalf("expected sorted (alphabetical) to win when given last: %s", out)
	}
}

func TestFollowChasesRedirect(t *testing.T) {
	var final http.HandlerFunc
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/end")
			w.WriteHeader(302)
			return
		}
		final(w, r)
	}))
	defer srv.Close()
	final = func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("landed"))
	}

	out, err := runCLI(t, "GET", "-F", srv.URL+"/start")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "landed") {
		t.Fatalf("expected final body in output: %s", out)
	}
}

func TestWithoutFollowRedirectIsNotChased(t *testing.T) {
	visited := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		visited++
		w.Header().Set("Location", "/end")
		w.WriteHeader(302)
	}))
	defer srv.Close()

	out, err := runCLI(t, "GET", srv.URL+"/start")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if visited != 1 {
		t.Fatalf("server hit %d times, want 1 (no following)", visited)
	}
	if !strings.Contains(out, "302") {
		t.Fatalf("expected 302 status in output: %s", out)
	}
}

func TestMaxRedirectsExceeded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/loop")
		w.WriteHeader(302)
	}))
	defer srv.Close()

	_, err := runCLI(t, "GET", "-F", "--max-redirects", "2", srv.URL+"/loop")
	if err == nil {
		t.Fatal("expected error for exceeding --max-redirects")
	}
}

func TestAllShowsIntermediateHop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/end")
			w.WriteHeader(302)
			return
		}
		w.Write([]byte("landed"))
	}))
	defer srv.Close()

	// --print Hh (request+response headers, no body) isolates the request-
	// vs-response distinction: with --all, both status lines (302 and 200)
	// should appear alongside both request lines.
	out, err := runCLI(t, "GET", "-F", "--all", "--print", "Hh", srv.URL+"/start")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "302 Found") {
		t.Fatalf("expected intermediate 302 response in --all output: %s", out)
	}
	if !strings.Contains(out, "200 OK") {
		t.Fatalf("expected final 200 response in --all output: %s", out)
	}
	if strings.Count(out, "GET /") < 2 {
		t.Fatalf("expected both hop requests (GET /start and GET /end) in --all output: %s", out)
	}
}

func TestWithoutAllOnlyShowsFinalResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			w.Header().Set("Location", "/end")
			w.WriteHeader(302)
			return
		}
		w.Write([]byte("landed"))
	}))
	defer srv.Close()

	out, err := runCLI(t, "GET", "-F", "--print", "Hh", srv.URL+"/start")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if strings.Contains(out, "302 Found") {
		t.Fatalf("did not expect intermediate 302 response without --all: %s", out)
	}
	if !strings.Contains(out, "200 OK") {
		t.Fatalf("expected final 200 response: %s", out)
	}
	// Both hop requests are still shown (httpie always shows every request
	// it made), only the intermediate *response* is suppressed.
	if strings.Count(out, "GET /") < 2 {
		t.Fatalf("expected both hop requests even without --all: %s", out)
	}
}
