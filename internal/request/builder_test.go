package request

import (
	"io"
	"mime"
	"os"
	"path/filepath"
	"testing"

	"app.getnutshell/htee/internal/itemsyntax"
)

func items(t *testing.T, raws ...string) []itemsyntax.KeyValueArg {
	t.Helper()
	var out []itemsyntax.KeyValueArg
	for _, r := range raws {
		kv, err := itemsyntax.ParseItem(r, itemsyntax.AllItemSeparators)
		if err != nil {
			t.Fatalf("ParseItem(%q): %v", r, err)
		}
		out = append(out, kv)
	}
	return out
}

func TestBuildJSONDefault(t *testing.T) {
	res, err := Build(Options{
		Method: "POST",
		URL:    "localhost:8080/",
		Items:  items(t, "foo=bar"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := string(res.Body); got != `{"foo":"bar"}` {
		t.Fatalf("body = %s", got)
	}
	if ct := res.Request.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
	if acc := res.Request.Header.Get("Accept"); acc != "application/json, */*;q=0.5" {
		t.Fatalf("accept = %q", acc)
	}
	if res.Request.URL.Scheme != "https" || res.Request.URL.Host != "localhost:8080" {
		t.Fatalf("url = %v", res.Request.URL)
	}
}

func TestBuildFormMode(t *testing.T) {
	res, err := Build(Options{
		Method: "POST",
		URL:    "example.org",
		Form:   true,
		Items:  items(t, "foo=bar", "n:=1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ct := res.Request.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if got := string(res.Body); got != "foo=bar&n=1" {
		t.Fatalf("body = %s", got)
	}
}

func TestBuildQueryAndHeaders(t *testing.T) {
	res, err := Build(Options{
		Method: "GET",
		URL:    "example.org/path",
		Items:  items(t, "search==term", "X-Foo:bar", "Accept;"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.URL.RawQuery != "search=term" {
		t.Fatalf("query = %q", res.Request.URL.RawQuery)
	}
	if got := res.Request.Header.Get("X-Foo"); got != "bar" {
		t.Fatalf("X-Foo = %q", got)
	}
	if got := res.Request.Header.Get("Accept"); got != "" {
		t.Fatalf("Accept should be removed by empty-header item, got %q", got)
	}
}

func TestBuildRawBody(t *testing.T) {
	res, err := Build(Options{
		Method: "POST",
		URL:    ":8080/",
		HasRaw: true,
		Raw:    `{"a":1}`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(res.Body) != `{"a":1}` {
		t.Fatalf("body = %s", res.Body)
	}
	if res.Request.URL.String() != "https://localhost:8080/" {
		t.Fatalf("url = %s", res.Request.URL)
	}
}

func TestDefaultSchemeIsHTTPSWhenUnset(t *testing.T) {
	res, err := Build(Options{
		Method: "GET",
		URL:    "example.org",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Request.URL.String() != "https://example.org" {
		t.Fatalf("url = %s, want https://example.org", res.Request.URL)
	}
}

func TestBuildBareFileBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	if err := os.WriteFile(path, []byte(`{"a":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Build(Options{
		Method: "POST",
		URL:    "example.org",
		Items:  items(t, "@"+path),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(res.Body) != `{"a":1}` {
		t.Fatalf("body = %s", res.Body)
	}
	if ct := res.Request.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestBuildMultipartWithFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Build(Options{
		Method: "POST",
		URL:    "example.org",
		Items:  items(t, "name=bob", "file@"+path),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != BodyModeMultipart {
		t.Fatalf("expected multipart mode, got %v", res.Mode)
	}
	ct := res.Request.Header.Get("Content-Type")
	if ct == "" {
		t.Fatal("expected multipart Content-Type with boundary")
	}
	mediaType, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("bad content type: %v", err)
	}
	if mediaType != "multipart/form-data" || params["boundary"] == "" {
		t.Fatalf("mediaType=%q params=%v", mediaType, params)
	}
	body, err := io.ReadAll(res.Request.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(body) == 0 {
		t.Fatal("expected non-empty multipart body")
	}
}

func TestBuildEmptyDataNoBody(t *testing.T) {
	res, err := Build(Options{Method: "GET", URL: "example.org"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Body) != 0 {
		t.Fatalf("expected no body, got %q", res.Body)
	}
	if ct := res.Request.Header.Get("Content-Type"); ct != "" {
		t.Fatalf("expected no Content-Type for empty body, got %q", ct)
	}
}

func TestBuildFormRejectsComplexJSON(t *testing.T) {
	_, err := Build(Options{
		Method: "POST",
		URL:    "example.org",
		Form:   true,
		Items:  items(t, `obj:={"a":1}`),
	})
	if err == nil {
		t.Fatal("expected error for complex JSON value in form mode")
	}
}
