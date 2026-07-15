package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

func TestWriteCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Entrypoint:  "/v1",
		OpenAPISpec: "openapi.yaml",
		Servers: map[string]string{
			"local": "http://localhost:8080",
			"prod":  "https://api.example.com",
		},
	}

	if err := Write(dir, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	path := Path(dir)
	if path != filepath.Join(dir, ".ht", "conf.toml") {
		t.Fatalf("Path = %q", path)
	}
	if !Exists(dir) {
		t.Fatalf("Exists = false after Write")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got Config
	if err := toml.Unmarshal(raw, &got); err != nil {
		t.Fatalf("Unmarshal round-trip: %v\n%s", err, raw)
	}
	if got.Entrypoint != cfg.Entrypoint {
		t.Errorf("Entrypoint = %q, want %q", got.Entrypoint, cfg.Entrypoint)
	}
	if got.OpenAPISpec != cfg.OpenAPISpec {
		t.Errorf("OpenAPISpec = %q, want %q", got.OpenAPISpec, cfg.OpenAPISpec)
	}
	if len(got.Servers) != len(cfg.Servers) {
		t.Errorf("Servers = %v, want %v", got.Servers, cfg.Servers)
	}
	for name, url := range cfg.Servers {
		if got.Servers[name] != url {
			t.Errorf("Servers[%q] = %q, want %q", name, got.Servers[name], url)
		}
	}

	content := string(raw)
	for _, name := range []string{"entrypoint", "openapispec", "servers", "[servers]"} {
		if !strings.Contains(content, name) {
			t.Errorf("output missing %q:\n%s", name, content)
		}
	}
	for _, comment := range []string{
		"Optional entry point URI path for HATEOAS-style APIs",
		"Optional path to an OpenAPI 3 spec file",
		"Named servers as scheme + authority",
	} {
		if !strings.Contains(content, comment) {
			t.Errorf("output missing comment %q:\n%s", comment, content)
		}
	}

	// entrypoint/openapispec must be root-level keys, declared before the
	// [servers] table opens - otherwise they'd be (mis)parsed as belonging
	// to the servers table.
	entrypointIdx := strings.Index(content, "entrypoint")
	openapispecIdx := strings.Index(content, "openapispec")
	serversTableIdx := strings.Index(content, "[servers]")
	if entrypointIdx < 0 || openapispecIdx < 0 || serversTableIdx < 0 {
		t.Fatalf("expected all three keys present:\n%s", content)
	}
	if !(entrypointIdx < serversTableIdx && openapispecIdx < serversTableIdx) {
		t.Errorf("entrypoint/openapispec must precede [servers]:\n%s", content)
	}
}

func TestWriteOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	first := Config{Servers: map[string]string{"a": "http://a"}}
	second := Config{Servers: map[string]string{"b": "http://b"}}

	if err := Write(dir, first); err != nil {
		t.Fatalf("Write(first): %v", err)
	}
	if err := Write(dir, second); err != nil {
		t.Fatalf("Write(second): %v", err)
	}

	raw, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), "http://a") {
		t.Errorf("expected overwrite to drop the first config, got:\n%s", raw)
	}
	if !strings.Contains(string(raw), "http://b") {
		t.Errorf("expected overwrite to contain the second config, got:\n%s", raw)
	}
}

func TestExistsFalseWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	if Exists(dir) {
		t.Fatalf("Exists = true for a directory with no config written")
	}
}

func TestReadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{
		Entrypoint:  "/v1",
		OpenAPISpec: "openapi.yaml",
		Servers: map[string]string{
			"local": "http://localhost:8080",
			"prod":  "https://api.example.com",
		},
	}
	if err := Write(dir, cfg); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := Read(dir)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Entrypoint != cfg.Entrypoint {
		t.Errorf("Entrypoint = %q, want %q", got.Entrypoint, cfg.Entrypoint)
	}
	if got.OpenAPISpec != cfg.OpenAPISpec {
		t.Errorf("OpenAPISpec = %q, want %q", got.OpenAPISpec, cfg.OpenAPISpec)
	}
	if len(got.Servers) != len(cfg.Servers) {
		t.Errorf("Servers = %v, want %v", got.Servers, cfg.Servers)
	}
	for name, url := range cfg.Servers {
		if got.Servers[name] != url {
			t.Errorf("Servers[%q] = %q, want %q", name, got.Servers[name], url)
		}
	}
}

func TestReadMissingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Read(dir); err == nil {
		t.Fatalf("expected an error reading a config that was never written")
	}
}
