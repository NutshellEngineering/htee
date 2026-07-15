package cli

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app.getnutshell/htee/internal/config"
)

// writeProjectWithSpec writes .ht/conf.toml (pointing at a spec file next
// to it) into a fresh temp dir, chdirs the test into it, and returns the
// temp dir. The spec template's single %s is replaced with the server URL
// so the spec's servers entry matches the httptest server.
func writeProjectWithSpec(t *testing.T, srv *httptest.Server, specTemplate string) {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)

	specPath := filepath.Join(dir, "openapi.yaml")
	spec := fmt.Sprintf(specTemplate, srv.URL)
	if err := os.WriteFile(specPath, []byte(spec), 0o644); err != nil {
		t.Fatalf("WriteFile(spec): %v", err)
	}

	if err := config.Write(dir, config.Config{
		OpenAPISpec: specPath,
		Servers:     map[string]string{"local": srv.URL},
	}); err != nil {
		t.Fatalf("config.Write: %v", err)
	}
}

const petsSpecTemplate = `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: %s
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
          content:
            application/json:
              schema:
                type: object
                required: [name]
                properties:
                  name:
                    type: string
`

func TestRequestValidatesAgainstConfiguredSpecValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"name":"Fido"}`))
		if err != nil {
			t.Fatalf("w.Write failed unexpectedly")
		}
	}))
	defer srv.Close()

	writeProjectWithSpec(t, srv, petsSpecTemplate)

	out, err := runCLI(t, "GET", srv.URL+"/pets")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "openapi: response is valid") {
		t.Fatalf("expected a validation success note, got: %s", out)
	}
}

func TestRequestValidatesAgainstConfiguredSpecInvalid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{}`))
		if err != nil {
			t.Fatalf("w.Write failed unexpectedly")
		}
	}))
	defer srv.Close()

	writeProjectWithSpec(t, srv, petsSpecTemplate)

	out, err := runCLI(t, "GET", srv.URL+"/pets")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "openapi: response failed validation") {
		t.Fatalf("expected a validation failure note, got: %s", out)
	}
	if !strings.Contains(out, "name") {
		t.Fatalf("expected the failure list to mention the missing field, got: %s", out)
	}
}

func TestRequestNotCoveredBySpecWarns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			t.Fatalf("w.Write failed unexpectedly")
		}
	}))
	defer srv.Close()

	writeProjectWithSpec(t, srv, petsSpecTemplate)

	out, err := runCLI(t, "GET", srv.URL+"/unrelated")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "not found in the OpenAPI spec") {
		t.Fatalf("expected a not-found warning, got: %s", out)
	}
}

func TestRequestPrintsValidationEnabledNoticeBeforeRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"name":"Fido"}`))
		if err != nil {
			t.Fatalf("w.Write failed unexpectedly")
		}
	}))
	defer srv.Close()

	writeProjectWithSpec(t, srv, petsSpecTemplate)

	out, err := runCLI(t, "GET", srv.URL+"/pets")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}

	noticeIdx := strings.Index(out, "openapi: validating request and response")
	reqIdx := strings.Index(out, "GET /pets")
	if noticeIdx < 0 {
		t.Fatalf("expected a validating notice, got: %s", out)
	}
	if reqIdx < 0 || noticeIdx > reqIdx {
		t.Fatalf("expected the validating notice before the request line, got: %s", out)
	}
}

func TestRequestWithBrokenSpecReportsErrorAfterResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"ok":true}`))
		if err != nil {
			t.Fatalf("w.Write failed unexpectedly")
		}
	}))
	defer srv.Close()

	// An operation with no responses defined fails full OpenAPI validation,
	// so LoadDoc rejects it - unlike duplicate tags, this isn't safely
	// ignorable, so it should still surface as a load error.
	writeProjectWithSpec(t, srv, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: %s
paths:
  /pets:
    get:
      responses: {}
`)

	out, err := runCLI(t, "GET", srv.URL+"/pets")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "openapi: could not load OpenAPI spec") {
		t.Fatalf("expected a load-error note, got: %s", out)
	}

	bodyIdx := strings.Index(out, `"ok":true`)
	errIdx := strings.Index(out, "openapi: could not load OpenAPI spec")
	if bodyIdx < 0 || errIdx < 0 || errIdx < bodyIdx {
		t.Fatalf("expected the load-error note after the response body, got: %s", out)
	}
}
