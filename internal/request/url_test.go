package request

import "testing"

func TestNormalizeURLAutoSchemeForLocalhost(t *testing.T) {
	cases := map[string]string{
		"localhost:8080/foo":  "http://localhost:8080/foo",
		"localhost":           "http://localhost",
		"127.0.0.1:5000":      "http://127.0.0.1:5000",
		"127.5.5.5":           "http://127.5.5.5",
		"[::1]:8080/foo":      "http://[::1]:8080/foo",
		":8080/foo":           "http://localhost:8080/foo",
		":/foo":               "http://localhost/foo",
		"user:pass@localhost": "http://user:pass@localhost",
	}
	for raw, want := range cases {
		if got := NormalizeURL(raw, ""); got != want {
			t.Errorf("NormalizeURL(%q, \"\") = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeURLAutoSchemeForNonLocalhost(t *testing.T) {
	cases := map[string]string{
		"example.org":            "https://example.org",
		"google.com/search?q=x":  "https://google.com/search?q=x",
		"user:pass@example.org":  "https://user:pass@example.org",
		"sub.example.org:9000/p": "https://sub.example.org:9000/p",
		"0.0.0.0:8080":           "https://0.0.0.0:8080",
	}
	for raw, want := range cases {
		if got := NormalizeURL(raw, ""); got != want {
			t.Errorf("NormalizeURL(%q, \"\") = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeURLExplicitDefaultSchemeOverridesAutoDetection(t *testing.T) {
	// An explicit --default-scheme forces that scheme uniformly, even for
	// what would otherwise auto-detect as localhost or non-local.
	if got := NormalizeURL("localhost:8080", "https"); got != "https://localhost:8080" {
		t.Errorf("NormalizeURL(localhost, explicit https) = %q, want https://localhost:8080", got)
	}
	if got := NormalizeURL("example.org", "http"); got != "http://example.org" {
		t.Errorf("NormalizeURL(example.org, explicit http) = %q, want http://example.org", got)
	}
}

func TestNormalizeURLAlreadySchemedIsUnchanged(t *testing.T) {
	if got := NormalizeURL("http://localhost:8080/foo", ""); got != "http://localhost:8080/foo" {
		t.Errorf("NormalizeURL(already schemed) = %q, want unchanged", got)
	}
	if got := NormalizeURL("https://example.org", ""); got != "https://example.org" {
		t.Errorf("NormalizeURL(already schemed) = %q, want unchanged", got)
	}
}
