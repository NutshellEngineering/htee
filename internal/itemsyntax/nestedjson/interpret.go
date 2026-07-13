package nestedjson

import (
	"encoding/json"
	"strings"

	"app.getnutshell/htee/internal/ordered"
)

// Array is a mutable, shared handle to a JSON array being built up across
// possibly-many `interpret` calls (e.g. repeated `tags[]=a tags[]=b` items),
// analogous to Python's in-place-mutable list.
type Array struct {
	Items []any
}

// MarshalJSON emits the array's items directly, as a plain JSON array.
func (a *Array) MarshalJSON() ([]byte, error) {
	if a.Items == nil {
		return []byte("[]"), nil
	}
	return json.Marshal(a.Items)
}

// KeyValuePair is one (bracket-path key, already-processed value) pair to
// fold into a nested JSON structure. Value is the final value to place at
// that path - for `:=`/`:=@` items it may itself be an arbitrary parsed JSON
// value (map/slice/number/bool/nil); for `=`/`=@` items it's always a string.
type KeyValuePair struct {
	Key   string
	Value any
}

// Interpret folds a sequence of bracket-path (key, value) pairs into one
// nested JSON structure - an *ordered.Map, or (if the pairs describe a
// top-level array, e.g. `[]=1 []=2`) an *Array. Mirrors
// httpie/cli/nested_json/interpret.py:interpret_nested_json plus
// wrap_with_dict/unwrap_top_level_list_if_needed, collapsed into a single
// call since Go doesn't need the dict-wrapping trick to carry the "was this
// a top-level array" fact - the concrete return type conveys it directly.
func Interpret(pairs []KeyValuePair) (any, error) {
	var context any
	for _, pair := range pairs {
		next, err := interpretOne(context, pair.Key, pair.Value)
		if err != nil {
			return nil, err
		}
		context = next
	}
	if context == nil {
		return ordered.NewMap(), nil
	}
	return context, nil
}

func objectFor(kind PathAction) any {
	if kind == ActionKey {
		return ordered.NewMap()
	}
	return &Array{}
}

func interpretOne(context any, key string, value any) (any, error) {
	paths, err := Parse(key)
	if err != nil {
		return nil, err
	}
	paths = append(paths, Path{Kind: ActionSet, Value: value})

	cursor := context

	typeCheck := func(index int, path Path, wantMap bool) error {
		_, isMap := cursor.(*ordered.Map)
		_, isArray := cursor.(*Array)
		if (wantMap && isMap) || (!wantMap && isArray) {
			return nil
		}
		var tok *Token
		if len(path.Tokens) > 0 {
			first, last := path.Tokens[0], path.Tokens[len(path.Tokens)-1]
			t := Token{Kind: -1, Start: first.Start, End: last.End}
			tok = &t
		}
		cursorType := jsonTypeName(cursor)
		requiredType := "object"
		if !wantMap {
			requiredType = "array"
		}
		var reconstructed strings.Builder
		for _, pp := range paths[:index] {
			reconstructed.WriteString(pp.Reconstruct())
		}
		message := "Cannot perform " + path.Kind.String() + " based access on " +
			quote(reconstructed.String()) + " which has a type of " + quote(cursorType) +
			" but this operation requires a type of " + quote(requiredType) + "."
		return newTypeError(key, tok, message)
	}

	for i := 0; i < len(paths)-1; i++ {
		path := paths[i]
		next := paths[i+1]

		if cursor == nil {
			cursor = objectFor(path.Kind)
			if context == nil {
				context = cursor
			}
		}

		switch path.Kind {
		case ActionKey:
			if err := typeCheck(i, path, true); err != nil {
				return nil, err
			}
			m := cursor.(*ordered.Map)
			if next.Kind == ActionSet {
				m.Set(path.KeyStr, next.Value)
				return context, nil
			}
			existing, ok := m.Get(path.KeyStr)
			if !ok || existing == nil {
				existing = objectFor(next.Kind)
				m.Set(path.KeyStr, existing)
			}
			cursor = existing

		case ActionIndex:
			if err := typeCheck(i, path, false); err != nil {
				return nil, err
			}
			if path.Index < 0 {
				var tok *Token
				if len(path.Tokens) > 1 {
					tok = &path.Tokens[1]
				}
				return nil, newValueError(key, tok, "Negative indexes are not supported.")
			}
			arr := cursor.(*Array)
			for path.Index >= len(arr.Items) {
				arr.Items = append(arr.Items, nil)
			}
			if next.Kind == ActionSet {
				arr.Items[path.Index] = next.Value
				return context, nil
			}
			if arr.Items[path.Index] == nil {
				arr.Items[path.Index] = objectFor(next.Kind)
			}
			cursor = arr.Items[path.Index]

		case ActionAppend:
			if err := typeCheck(i, path, false); err != nil {
				return nil, err
			}
			arr := cursor.(*Array)
			if next.Kind == ActionSet {
				arr.Items = append(arr.Items, next.Value)
				return context, nil
			}
			newObj := objectFor(next.Kind)
			arr.Items = append(arr.Items, newObj)
			cursor = newObj

		case ActionSet:
			// unreachable: the only ActionSet path is the terminal value
			// appended in Interpret above, which is always paths' last
			// element and excluded by this loop's i < len(paths)-1 bound.
		}
	}

	return context, nil
}

func jsonTypeName(v any) string {
	switch v.(type) {
	case *ordered.Map:
		return "object"
	case *Array:
		return "array"
	case string:
		return "string"
	case float64, int:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return "null"
	default:
		return "value"
	}
}

func quote(s string) string {
	return "'" + s + "'"
}
