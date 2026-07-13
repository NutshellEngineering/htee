package output

import (
	"testing"

	"app.getnutshell/htee/internal/message"
)

func TestSortHeadersStable(t *testing.T) {
	in := []message.Header{
		{Name: "Zeta", Value: "1"},
		{Name: "Alpha", Value: "1"},
		{Name: "Alpha", Value: "2"},
		{Name: "Beta", Value: "1"},
	}
	got := sortHeaders(in)
	want := []string{"Alpha", "Alpha", "Beta", "Zeta"}
	for i, name := range want {
		if got[i].Name != name {
			t.Fatalf("got[%d]=%s, want %s (full: %+v)", i, got[i].Name, name, got)
		}
	}
	// Relative order of same-name headers preserved.
	if got[0].Value != "1" || got[1].Value != "2" {
		t.Fatalf("expected stable relative order for repeated Alpha headers, got %+v", got)
	}
	// Original slice untouched.
	if in[0].Name != "Zeta" {
		t.Fatalf("sortHeaders must not mutate its input, got %+v", in)
	}
}

func TestMimeOf(t *testing.T) {
	cases := []struct {
		headers []message.Header
		want    string
	}{
		{[]message.Header{{Name: "Content-Type", Value: "application/json; charset=utf-8"}}, "application/json"},
		{[]message.Header{{Name: "content-type", Value: "text/xml"}}, "text/xml"},
		{[]message.Header{{Name: "X-Other", Value: "nope"}}, ""},
		{nil, ""},
	}
	for _, c := range cases {
		if got := mimeOf(c.headers); got != c.want {
			t.Errorf("mimeOf(%+v) = %q, want %q", c.headers, got, c.want)
		}
	}
}
