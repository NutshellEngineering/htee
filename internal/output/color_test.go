package output

import (
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/formatters"
)

func TestValidStyle(t *testing.T) {
	if !ValidStyle("") || !ValidStyle(AutoStyle) {
		t.Fatal("expected empty and auto to be valid")
	}
	if !ValidStyle("monokai") {
		t.Fatal("expected a known bundled chroma style to be valid")
	}
	if ValidStyle("definitely-not-a-real-style") {
		t.Fatal("expected an unknown style name to be invalid")
	}
}

func TestResolveStyleAndFormatterAutoUsesTTY16(t *testing.T) {
	_, formatter := resolveStyleAndFormatter(AutoStyle)
	if formatter != formatters.TTY16 {
		t.Fatal("expected auto style to use the 16-color terminal formatter")
	}
}

func TestResolveStyleAndFormatterNamedUsesTTY256(t *testing.T) {
	_, formatter := resolveStyleAndFormatter("monokai")
	if formatter != formatters.TTY256 {
		t.Fatal("expected a named style to use the 256-color terminal formatter")
	}
}

func TestColorizeBodyJSON(t *testing.T) {
	style, formatter := resolveStyleAndFormatter(AutoStyle)
	out := colorizeBody([]byte(`{"a":1}`), "application/json", style, formatter)
	if !strings.Contains(string(out), "\x1b[") {
		t.Fatalf("expected ANSI escape codes in colorized JSON output, got %q", out)
	}
}

func TestColorizeBodyUnknownMimePassesThrough(t *testing.T) {
	style, formatter := resolveStyleAndFormatter(AutoStyle)
	body := []byte("just some bytes")
	out := colorizeBody(body, "application/x-htee-test-unmatched-zzzz", style, formatter)
	if string(out) != string(body) {
		t.Fatalf("expected passthrough for unmatched mime, got %q", out)
	}
}
