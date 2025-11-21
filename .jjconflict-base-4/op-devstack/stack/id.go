package stack

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
)

// Kind represents a kind of component, this is used to make each ID unique, even when encoded as text.
type Kind string

var _ slog.LogValuer = (*Kind)(nil)

func (k Kind) LogValue() slog.Value {
	return slog.StringValue(string(k))
}

func (k Kind) String() string {
	return string(k)
}

func (k Kind) MarshalText() ([]byte, error) {
	return []byte(k), nil
}

func (k *Kind) UnmarshalText(data []byte) error {
	*k = Kind(data)
	return nil
}

const maxIDLength = 100

var errInvalidID = errors.New("invalid ID")

// ComponentID is comparable, can be copied, contains a chain-ID,
// and has type-safe text encoding/decoding to prevent accidental mixups.
type ComponentID struct {
	Key  string
	Kind Kind
}

var _ slog.LogValuer = (*ComponentID)(nil)

func (id ComponentID) String() string {
	return fmt.Sprintf("%s-%s", id.Kind, id.Key)
}

func (id ComponentID) marshalText() ([]byte, error) {
	k := string(id.Key)
	if len(k) > maxIDLength {
		return nil, errInvalidID
	}
	k = fmt.Sprintf("%s-%s", id.Kind, k)
	return []byte(k), nil
}

func (id *ComponentID) unmarshalText(data []byte) error {
	kindData, mainData, ok := bytes.Cut(data, []byte("-"))
	if !ok {
		return fmt.Errorf("expected kind-prefix, but id has none: %q", data)
	}
	if len(mainData) > maxIDLength {
		return errInvalidID
	}
	id.Kind = Kind(kindData)
	id.Key = string(mainData)
	return nil
}

func (id ComponentID) LogValue() slog.Value {
	return slog.StringValue(id.String())
}

// copyAndSort helps copy and sort a slice of objects with the given less function
func copyAndSort[V ~[]E, E any](vs V, lessFn func(a, b E) bool) V {
	out := slices.Clone(vs)
	sort.Slice(out, func(i, j int) bool {
		a := out[i]
		b := out[j]
		return lessFn(a, b)
	})
	return out
}

// isLess is a helper function to compare two idWithChain objects.
// It does not use generics, since idWithChain is a concrete type with struct fields and no accessor methods in the types that wrap this type.
func isLess(a, b ComponentID) bool {
	if a.Key > b.Key {
		return false
	}
	return true
}
