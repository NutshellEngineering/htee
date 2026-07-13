package nestedjson

import (
	"encoding/json"
	"testing"

	"app.getnutshell/htee/internal/ordered"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestInterpretSimpleAndMerge(t *testing.T) {
	// person[name]=bob then person[age]:=30 should merge into one object.
	pairs := []KeyValuePair{
		{Key: "person[name]", Value: "bob"},
		{Key: "person[age]", Value: 30},
	}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(*ordered.Map)
	if !ok {
		t.Fatalf("expected *ordered.Map, got %T", result)
	}
	if got := mustJSON(t, m); got != `{"person":{"name":"bob","age":30}}` {
		t.Fatalf("got %s", got)
	}
}

func TestInterpretTopLevelArrayAppend(t *testing.T) {
	// tags[]=a tags[]=b
	pairs := []KeyValuePair{
		{Key: "tags[]", Value: "a"},
		{Key: "tags[]", Value: "b"},
	}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*ordered.Map)
	if got := mustJSON(t, m); got != `{"tags":["a","b"]}` {
		t.Fatalf("got %s", got)
	}
}

func TestInterpretBareTopLevelArray(t *testing.T) {
	// []=1 []=2 - a bare top-level array, not nested under any key.
	pairs := []KeyValuePair{
		{Key: "[]", Value: float64(1)},
		{Key: "[]", Value: float64(2)},
	}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	arr, ok := result.(*Array)
	if !ok {
		t.Fatalf("expected *Array (top-level array), got %T", result)
	}
	if got := mustJSON(t, arr.Items); got != `[1,2]` {
		t.Fatalf("got %s", got)
	}
}

func TestInterpretIndexedArray(t *testing.T) {
	// addresses[0][street]=Main addresses[1][street]=Oak
	pairs := []KeyValuePair{
		{Key: "addresses[0][street]", Value: "Main"},
		{Key: "addresses[1][street]", Value: "Oak"},
	}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*ordered.Map)
	if got := mustJSON(t, m); got != `{"addresses":[{"street":"Main"},{"street":"Oak"}]}` {
		t.Fatalf("got %s", got)
	}
}

func TestInterpretEscapedIntStaysLiteralKey(t *testing.T) {
	// foo[\0]=bar -> key path segment "0" as a literal string key, not index 0.
	pairs := []KeyValuePair{
		{Key: `foo[\0]`, Value: "bar"},
	}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*ordered.Map)
	if got := mustJSON(t, m); got != `{"foo":{"0":"bar"}}` {
		t.Fatalf("got %s", got)
	}
}

func TestInterpretTypeConflictErrors(t *testing.T) {
	// foo=bar then foo[0]=x - foo is already a string, can't index into it.
	pairs := []KeyValuePair{
		{Key: "foo", Value: "bar"},
		{Key: "foo[0]", Value: "x"},
	}
	_, err := Interpret(pairs)
	if err == nil {
		t.Fatal("expected a type-conflict error")
	}
	se, ok := err.(*SyntaxError)
	if !ok {
		t.Fatalf("expected *SyntaxError, got %T: %v", err, err)
	}
	if se.MessageKind != "Type" {
		t.Fatalf("expected Type error, got %q: %v", se.MessageKind, err)
	}
}

func TestInterpretNegativeIndexRejected(t *testing.T) {
	_, err := Interpret([]KeyValuePair{{Key: "a[-1]", Value: "x"}})
	if err == nil {
		t.Fatal("expected error for negative index")
	}
}

func TestInterpretRootNumericKeyIsStringNotIndex(t *testing.T) {
	// A bare unbracketed root literal is always a KEY, even if numeric-looking.
	pairs := []KeyValuePair{{Key: "123", Value: "x"}}
	result, err := Interpret(pairs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(*ordered.Map)
	if got := mustJSON(t, m); got != `{"123":"x"}` {
		t.Fatalf("got %s", got)
	}
}
