package netrc

import (
	"os"
	"path/filepath"
	"testing"
)

func withNetrcFile(t *testing.T, content string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".netrc")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETRC", path)
}

func TestLookupMachineMatch(t *testing.T) {
	withNetrcFile(t, `
machine example.com
login alice
password s3cret

machine other.com
login bob
password hunter2
`)
	user, pass, ok := Lookup("example.com")
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("got user=%q pass=%q ok=%v", user, pass, ok)
	}

	user, pass, ok = Lookup("other.com")
	if !ok || user != "bob" || pass != "hunter2" {
		t.Fatalf("got user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestLookupNoMatch(t *testing.T) {
	withNetrcFile(t, `
machine example.com
login alice
password s3cret
`)
	if _, _, ok := Lookup("nowhere.com"); ok {
		t.Fatal("expected no match for unrelated host")
	}
}

func TestLookupDefaultFallback(t *testing.T) {
	withNetrcFile(t, `
machine example.com
login alice
password s3cret

default
login anon
password anonpass
`)
	user, pass, ok := Lookup("nowhere.com")
	if !ok || user != "anon" || pass != "anonpass" {
		t.Fatalf("expected default entry, got user=%q pass=%q ok=%v", user, pass, ok)
	}

	// An explicit machine match still wins over default.
	user, pass, ok = Lookup("example.com")
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("expected machine entry to win over default, got user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestLookupSingleLineEntry(t *testing.T) {
	withNetrcFile(t, `machine example.com login alice password s3cret`)
	user, pass, ok := Lookup("example.com")
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("got user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestLookupIgnoresMacdefBody(t *testing.T) {
	withNetrcFile(t, `
macdef init
machine should-not-match
login should-not-appear
password should-not-appear

machine example.com
login alice
password s3cret
`)
	if _, _, ok := Lookup("should-not-match"); ok {
		t.Fatal("macdef body should not be parsed as a machine entry")
	}
	user, pass, ok := Lookup("example.com")
	if !ok || user != "alice" || pass != "s3cret" {
		t.Fatalf("got user=%q pass=%q ok=%v", user, pass, ok)
	}
}

func TestLookupNoFile(t *testing.T) {
	t.Setenv("NETRC", filepath.Join(t.TempDir(), "does-not-exist"))
	if _, _, ok := Lookup("example.com"); ok {
		t.Fatal("expected no match when netrc file is missing")
	}
}
