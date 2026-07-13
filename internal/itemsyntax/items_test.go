package itemsyntax

import (
	"encoding/json"
	"testing"

	"app.getnutshell/htee/internal/ordered"
)

func parseAll(t *testing.T, raws ...string) []KeyValueArg {
	t.Helper()
	var out []KeyValueArg
	for _, r := range raws {
		kv, err := ParseItem(r, AllItemSeparators)
		if err != nil {
			t.Fatalf("ParseItem(%q): %v", r, err)
		}
		out = append(out, kv)
	}
	return out
}

func TestFromArgsJSONMode(t *testing.T) {
	items := parseAll(t, "name=bob", "age:=30", "X-Foo:bar", "search==term")
	ri, err := FromArgs(items, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ri.Headers) != 1 || ri.Headers[0].Name != "X-Foo" || *ri.Headers[0].Value != "bar" {
		t.Fatalf("headers = %+v", ri.Headers)
	}
	if len(ri.Params) != 1 || ri.Params[0].Name != "search" || ri.Params[0].Value != "term" {
		t.Fatalf("params = %+v", ri.Params)
	}
	m, ok := ri.Data.(*ordered.Map)
	if !ok {
		t.Fatalf("expected *ordered.Map data, got %T", ri.Data)
	}
	b, _ := json.Marshal(m)
	if string(b) != `{"name":"bob","age":30}` {
		t.Fatalf("data = %s", b)
	}
}

func TestFromArgsFormModeRejectsComplexRawJSON(t *testing.T) {
	items := parseAll(t, `obj:={"a":1}`)
	_, err := FromArgs(items, false)
	if err == nil {
		t.Fatal("expected error for complex JSON value in form mode")
	}
}

func TestFromArgsFormModeFlatKeys(t *testing.T) {
	// In form mode, brackets in keys are NOT interpreted - the key is used literally.
	items := parseAll(t, "person[name]=bob", "age:=30")
	ri, err := FromArgs(items, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := ri.Data.(*ordered.Map)
	v, ok := m.Get("person[name]")
	if !ok || v != "bob" {
		t.Fatalf("expected literal key 'person[name]', got %+v", m.Keys())
	}
	age, _ := m.Get("age")
	if age != float64(30) {
		t.Fatalf("age = %v", age)
	}
}

func TestFromArgsEmptyHeaderRemoves(t *testing.T) {
	items := parseAll(t, "Accept;")
	ri, err := FromArgs(items, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ri.Headers) != 1 || ri.Headers[0].Value != nil {
		t.Fatalf("headers = %+v", ri.Headers)
	}
}
