// Package ordered provides an insertion-order-preserving map, needed because
// Go's encoding/json alphabetizes map[string]any keys on Marshal, whereas
// httpie (backed by Python dicts) preserves the declaration order of
// key=value items in the request body.
package ordered

import (
	"bytes"
	"encoding/json"
)

// Map is an insertion-order-preserving string-keyed map of JSON values.
type Map struct {
	keys   []string
	values map[string]any
}

// NewMap returns an empty ordered Map.
func NewMap() *Map {
	return &Map{values: map[string]any{}}
}

// Set inserts or updates key. Existing keys keep their original position;
// new keys are appended in insertion order.
func (m *Map) Set(key string, value any) {
	if _, exists := m.values[key]; !exists {
		m.keys = append(m.keys, key)
	}
	m.values[key] = value
}

// Get returns the value for key and whether it was present.
func (m *Map) Get(key string) (any, bool) {
	v, ok := m.values[key]
	return v, ok
}

// Keys returns the map's keys in insertion order.
func (m *Map) Keys() []string {
	return m.keys
}

// Len returns the number of entries.
func (m *Map) Len() int {
	return len(m.keys)
}

// Sorted returns a copy of the Map with keys sorted alphabetically. Used by
// the json.sort_keys / --sorted output-formatting option; does not affect
// how the request body itself is built (which always preserves declaration
// order).
func (m *Map) Sorted() *Map {
	out := NewMap()
	keys := append([]string(nil), m.keys...)
	sortStrings(keys)
	for _, k := range keys {
		out.Set(k, m.values[k])
	}
	return out
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

// MarshalJSON emits the map's entries in insertion order.
func (m *Map) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')
		valBytes, err := json.Marshal(m.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
