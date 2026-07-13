package output

import "testing"

func TestFormatJSONBodyIndentAndOrder(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.JSONSortKeys = false
	got := string(formatJSONBody([]byte(`{"b":1,"a":2}`), opts))
	want := "{\n    \"b\": 1,\n    \"a\": 2\n}"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatJSONBodySortKeys(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.JSONIndent = 2
	got := string(formatJSONBody([]byte(`{"b":1,"a":{"z":1,"y":2}}`), opts))
	want := "{\n  \"a\": {\n    \"y\": 2,\n    \"z\": 1\n  },\n  \"b\": 1\n}"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatJSONBodyDisabled(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.JSONFormat = false
	body := []byte(`{"b":1,"a":2}`)
	got := formatJSONBody(body, opts)
	if string(got) != string(body) {
		t.Fatalf("expected passthrough when json.format=false, got %s", got)
	}
}

func TestFormatJSONBodyInvalidPassesThrough(t *testing.T) {
	opts := DefaultFormatOptions()
	body := []byte(`not json at all`)
	got := formatJSONBody(body, opts)
	if string(got) != string(body) {
		t.Fatalf("expected invalid JSON to pass through unchanged, got %q", got)
	}
}

func TestFormatJSONBodyXSSIPrefix(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.JSONIndent = 2
	opts.JSONSortKeys = false
	got := string(formatJSONBody([]byte(`while(1);{"a":1}`), opts))
	want := "while(1);{\n  \"a\": 1\n}"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatJSONBodyArray(t *testing.T) {
	opts := DefaultFormatOptions()
	opts.JSONIndent = 2
	opts.JSONSortKeys = false
	got := string(formatJSONBody([]byte(`[1,2,3]`), opts))
	want := "[\n  1,\n  2,\n  3\n]"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}
