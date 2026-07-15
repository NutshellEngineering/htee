// Package config reads and writes the project-local .ht/conf.toml file
// created by `ht init`.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	dirName  = ".ht"
	fileName = "conf.toml"

	fileHeader = "# htee configuration, created by `ht init`.\n\n"
)

// Config is the on-disk shape of .ht/conf.toml.
//
// Entrypoint and OpenAPISpec are declared before Servers so they marshal as
// root-level keys ahead of the [servers] table: TOML keys following a table
// header belong to that table, so the map field has to come last.
type Config struct {
	Entrypoint  string            `toml:"entrypoint" comment:"Optional entry point URI path for HATEOAS-style APIs, where a client\nfollows links from a single starting resource instead of knowing every\nURL in advance. Not currently used automatically by ht."`
	OpenAPISpec string            `toml:"openapispec" comment:"Optional path to an OpenAPI 3 spec file describing this API."`
	Servers     map[string]string `toml:"servers" comment:"Named servers as scheme + authority, e.g. local = \"http://localhost:8080\".\nReference a server by name instead of typing its full URL."`
}

// Path returns the config file path for the project rooted at dir.
func Path(dir string) string {
	return filepath.Join(dir, dirName, fileName)
}

// Exists reports whether a config file already exists for the project
// rooted at dir.
func Exists(dir string) bool {
	_, err := os.Stat(Path(dir))
	return err == nil
}

// Read parses the project's .ht/conf.toml. Callers should check Exists
// first if a missing file shouldn't be treated as an error.
func Read(dir string) (Config, error) {
	raw, err := os.ReadFile(Path(dir))
	if err != nil {
		return Config{}, fmt.Errorf("reading %s: %w", fileName, err)
	}
	var cfg Config
	if err := toml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", fileName, err)
	}
	return cfg, nil
}

// Write creates .ht/ (if needed) and writes cfg to conf.toml, overwriting
// any existing file. Callers are responsible for confirming the overwrite
// with the user first.
func Write(dir string, cfg Config) error {
	if err := os.MkdirAll(filepath.Join(dir, dirName), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dirName, err)
	}

	body, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	out := append([]byte(fileHeader), body...)
	if err := os.WriteFile(Path(dir), out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", fileName, err)
	}
	return nil
}
