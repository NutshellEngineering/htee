package cli

import (
	"strings"
	"testing"
)

func TestInitCommandIsRegistered(t *testing.T) {
	cmd := NewRootCommand(Config{VerbPreset: ""})
	sub, _, err := cmd.Find([]string{"init"})
	if err != nil {
		t.Fatalf("Find(init): %v", err)
	}
	if sub.Name() != "init" {
		t.Fatalf("expected the init subcommand, got %q", sub.Name())
	}
}

func TestInitCommandRejectsExtraArgs(t *testing.T) {
	out, err := runCLI(t, "", "init", "extra", "another")
	if err == nil {
		t.Fatalf("expected an error for `ht init extra another`, out=%s", out)
	}
}

func TestInitCommandRejectsUnreadableSwaggerFile(t *testing.T) {
	out, err := runCLI(t, "", "init", "/nonexistent/openapi.yaml")
	if err == nil {
		t.Fatalf("expected an error for a missing swagger file, out=%s", out)
	}
}

func TestInitCommandHelp(t *testing.T) {
	out, err := runCLI(t, "", "init", "--help")
	if err != nil {
		t.Fatalf("unexpected error: %v; out=%s", err, out)
	}
	if !strings.Contains(out, "conf.toml") {
		t.Fatalf("expected help text to mention conf.toml, got: %s", out)
	}
}
