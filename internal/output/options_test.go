package output

import "testing"

func TestResolvePrettyDefaults(t *testing.T) {
	p, err := ResolvePretty("", false, true)
	if err != nil || !p.Colors || !p.Format {
		t.Fatalf("tty default: got %+v, err=%v", p, err)
	}
	p, err = ResolvePretty("", false, false)
	if err != nil || p.Colors || p.Format {
		t.Fatalf("redirected default: got %+v, err=%v", p, err)
	}
}

func TestResolvePrettyExplicit(t *testing.T) {
	cases := []struct {
		value          string
		colors, format bool
	}{
		{"all", true, true},
		{"colors", true, false},
		{"format", false, true},
		{"none", false, false},
	}
	for _, c := range cases {
		p, err := ResolvePretty(c.value, true, false)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", c.value, err)
		}
		if p.Colors != c.colors || p.Format != c.format {
			t.Errorf("%s: got %+v, want colors=%v format=%v", c.value, p, c.colors, c.format)
		}
	}
}

func TestResolvePrettyInvalid(t *testing.T) {
	if _, err := ResolvePretty("bogus", true, false); err == nil {
		t.Fatal("expected error for invalid --pretty value")
	}
}

func TestParseFormatOptionsDefaults(t *testing.T) {
	opts, err := ParseFormatOptions(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts != DefaultFormatOptions() {
		t.Fatalf("got %+v, want defaults %+v", opts, DefaultFormatOptions())
	}
}

func TestParseFormatOptionsOverride(t *testing.T) {
	opts, err := ParseFormatOptions([]string{"json.sort_keys:false,json.indent:2", "xml.indent:4"})
	if err != nil {
		t.Fatal(err)
	}
	if opts.JSONSortKeys || opts.JSONIndent != 2 || opts.XMLIndent != 4 {
		t.Fatalf("got %+v", opts)
	}
	// Untouched fields keep their defaults.
	if !opts.HeadersSort || !opts.JSONFormat || !opts.XMLFormat {
		t.Fatalf("expected untouched fields to keep defaults, got %+v", opts)
	}
}

func TestParseFormatOptionsOrderMatters(t *testing.T) {
	// --sorted then --unsorted: unsorted should win (last wins).
	opts, err := ParseFormatOptions([]string{SortedFormatOptionsString, UnsortedFormatOptionsString})
	if err != nil {
		t.Fatal(err)
	}
	if opts.HeadersSort || opts.JSONSortKeys {
		t.Fatalf("expected unsorted to win when applied last, got %+v", opts)
	}

	// Reversed order: sorted should win.
	opts, err = ParseFormatOptions([]string{UnsortedFormatOptionsString, SortedFormatOptionsString})
	if err != nil {
		t.Fatal(err)
	}
	if !opts.HeadersSort || !opts.JSONSortKeys {
		t.Fatalf("expected sorted to win when applied last, got %+v", opts)
	}
}

func TestParseFormatOptionsInvalidToken(t *testing.T) {
	if _, err := ParseFormatOptions([]string{"not-a-valid-token"}); err == nil {
		t.Fatal("expected error for malformed token")
	}
	if _, err := ParseFormatOptions([]string{"json.indent:not-a-number"}); err == nil {
		t.Fatal("expected error for non-numeric json.indent")
	}
	if _, err := ParseFormatOptions([]string{"bogus.key:true"}); err == nil {
		t.Fatal("expected error for unknown key")
	}
}
