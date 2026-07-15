package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSpec(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

const specWithServersAndPaths = `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: https://api.example.com
    description: Production
  - url: https://staging.example.com
    description: Staging
paths:
  /pets:
    get:
      responses:
        '200':
          description: OK
    post:
      responses:
        '201':
          description: Created
  /pets/{id}:
    get:
      responses:
        '200':
          description: OK
    parameters:
      - name: id
        in: path
        required: true
        schema:
          type: string
  /v1/health:
    get:
      responses:
        '200':
          description: OK
  /admin:
    post:
      responses:
        '200':
          description: OK
`

func TestLoadExtractsServersAndShortestGetPath(t *testing.T) {
	path := writeSpec(t, specWithServersAndPaths)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	want := map[string]string{
		"production": "https://api.example.com",
		"staging":    "https://staging.example.com",
	}
	if len(got.Servers) != len(want) {
		t.Fatalf("Servers = %v, want %v", got.Servers, want)
	}
	for name, url := range want {
		if got.Servers[name] != url {
			t.Errorf("Servers[%q] = %q, want %q", name, got.Servers[name], url)
		}
	}

	if got.Entrypoint != "/pets" {
		t.Errorf("Entrypoint = %q, want %q", got.Entrypoint, "/pets")
	}
}

func TestLoadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "openapi.json")
	content := `{
		"openapi": "3.0.0",
		"info": {"title": "Test API", "version": "1.0"},
		"servers": [{"url": "http://localhost:8080"}],
		"paths": {
			"/status": {"get": {"responses": {"200": {"description": "OK"}}}}
		}
	}`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Servers["server1"] != "http://localhost:8080" {
		t.Errorf("Servers = %v", got.Servers)
	}
	if got.Entrypoint != "/status" {
		t.Errorf("Entrypoint = %q", got.Entrypoint)
	}
}

func TestServerNamingFallsBackWhenNoDescription(t *testing.T) {
	path := writeSpec(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: https://one.example.com
  - url: https://two.example.com
paths: {}
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Servers["server1"] != "https://one.example.com" {
		t.Errorf("Servers = %v", got.Servers)
	}
	if got.Servers["server2"] != "https://two.example.com" {
		t.Errorf("Servers = %v", got.Servers)
	}
}

func TestServerNamingDedupesSlugCollisions(t *testing.T) {
	path := writeSpec(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
servers:
  - url: https://one.example.com
    description: "Prod!"
  - url: https://two.example.com
    description: "Prod?"
paths: {}
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Servers["prod"] != "https://one.example.com" {
		t.Errorf("Servers = %v", got.Servers)
	}
	if got.Servers["prod-2"] != "https://two.example.com" {
		t.Errorf("Servers = %v", got.Servers)
	}
}

func TestLoadNoServersNoGetPaths(t *testing.T) {
	path := writeSpec(t, `
openapi: 3.0.0
info:
  title: Test API
  version: "1.0"
paths:
  /admin:
    post:
      responses:
        '200':
          description: OK
`)

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Servers) != 0 {
		t.Errorf("Servers = %v, want empty", got.Servers)
	}
	if got.Entrypoint != "" {
		t.Errorf("Entrypoint = %q, want empty", got.Entrypoint)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatalf("expected an error for a missing file")
	}
}
