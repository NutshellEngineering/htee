package itemsyntax

import "testing"

func TestParseItem(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantKey string
		wantVal string
		wantSep Separator
		wantErr bool
	}{
		{name: "header", in: "X-Foo:bar", wantKey: "X-Foo", wantVal: "bar", wantSep: SepHeader},
		{name: "empty header", in: "X-Foo;", wantKey: "X-Foo", wantVal: "", wantSep: SepHeaderEmpty},
		{name: "query param", in: "search==term", wantKey: "search", wantVal: "term", wantSep: SepQueryParam},
		{name: "json string field", in: "name=bob", wantKey: "name", wantVal: "bob", wantSep: SepDataString},
		{name: "raw json field", in: "age:=30", wantKey: "age", wantVal: "30", wantSep: SepDataRawJSON},
		{name: "file upload", in: "file@./report.pdf", wantKey: "file", wantVal: "./report.pdf", wantSep: SepFileUpload},
		{name: "data embed file", in: "name=@notes.txt", wantKey: "name", wantVal: "notes.txt", wantSep: SepDataEmbedFile},
		{name: "raw json embed file", in: "obj:=@data.json", wantKey: "obj", wantVal: "data.json", wantSep: SepDataEmbedRawJSONFile},
		{name: "header embed file", in: "X-Sig:@sig.txt", wantKey: "X-Sig", wantVal: "sig.txt", wantSep: SepHeaderEmbed},
		{name: "query embed file", in: "q==@query.txt", wantKey: "q", wantVal: "query.txt", wantSep: SepQueryEmbedFile},
		{
			name: "escaped colon in key stays literal, = wins",
			// field-name-with\:colon=value  ->  key: "field-name-with:colon", sep '=', value: "value"
			in: `field-name-with\:colon=value`, wantKey: "field-name-with:colon", wantVal: "value", wantSep: SepDataString,
		},
		{
			name: "longest separator wins at same position",
			in:   "age:=30", wantKey: "age", wantVal: "30", wantSep: SepDataRawJSON,
		},
		{name: "no separator is an error", in: "justastring", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseItem(tc.in, AllItemSeparators)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Key != tc.wantKey || got.Value != tc.wantVal || got.Sep != tc.wantSep {
				t.Fatalf("ParseItem(%q) = %+v, want key=%q value=%q sep=%q", tc.in, got, tc.wantKey, tc.wantVal, tc.wantSep)
			}
		})
	}
}

func TestTokenizeEscaping(t *testing.T) {
	// Mirrors the KeyValueArgType.tokenize doctest:
	// tokenize(r'foo\=bar\\baz') == ['foo', Escaped('='), 'bar\\baz']
	toks := tokenize(`foo\=bar\\baz`, specialChars([]Separator{SepDataString}))
	if len(toks) != 3 {
		t.Fatalf("expected 3 tokens, got %d: %+v", len(toks), toks)
	}
	if toks[0].text != "foo" || toks[0].escaped {
		t.Fatalf("token0 = %+v", toks[0])
	}
	if toks[1].text != "=" || !toks[1].escaped {
		t.Fatalf("token1 = %+v", toks[1])
	}
	if toks[2].text != `bar\\baz` || toks[2].escaped {
		t.Fatalf("token2 = %+v", toks[2])
	}
}
