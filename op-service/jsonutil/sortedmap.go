package jsonutil

import (
	"encoding/json"
	"fmt"
	"sort"
)

// LazySortedJsonMap provides sorted encoding order for JSON maps.
// The sorting is lazy: in-memory it's just a map, until it sorts just-in-time when the map is encoded to JSON.
// Warning: the just-in-time sorting requires a full allocation of the map structure and keys slice during encoding.
// Sorting order is not enforced when decoding from JSON.
type LazySortedJsonMap[K comparable, V any] map[K]V

func (m LazySortedJsonMap[K, V]) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(m))
	values := make(map[string]V)
	for k, v := range m {
		s := fmt.Sprintf("%q", any(k)) // format as quoted string
		keys = append(keys, s)
		values[s] = v
	}
	sort.Strings(keys)
	var out []byte
	out = append(out, '{')
	for i, k := range keys {
		out = append(out, k...) // quotes are already included
		out = append(out, ':')
		v, err := json.Marshal(values[k])
		if err != nil {
			return nil, fmt.Errorf("failed to encode value of %s: %w", k, err)
		}
		out = append(out, v...)
		if i != len(keys)-1 {
			out = append(out, ',')
		}
	}
	out = append(out, '}')
	return out, nil
}

func (m *LazySortedJsonMap[K, V]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*map[K]V)(m))
}
