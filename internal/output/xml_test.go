package output

import "testing"

func TestFormatXMLBodyIndent(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.XMLIndent = 2
	got := string(formatXMLBody([]byte(`<a><b>1</b><c>2</c></a>`), opts))
	want := "<a>\n  <b>1</b>\n  <c>2</c>\n</a>"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatXMLBodyDisabled(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.XMLFormat = false
	body := []byte(`<a><b>1</b></a>`)
	got := formatXMLBody(body, opts)
	if string(got) != string(body) {
		t.Fatalf("expected passthrough when xml.format=false, got %s", got)
	}
}

func TestFormatXMLBodyInvalidPassesThrough(t *testing.T) {
	opts := DefaultFormatOptions()
	body := []byte(`<a><b>unterminated`)
	got := formatXMLBody(body, opts)
	if string(got) != string(body) {
		t.Fatalf("expected invalid XML to pass through unchanged, got %q", got)
	}
}
