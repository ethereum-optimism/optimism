package stack

import (
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"

	"github.com/ethereum-optimism/optimism/op-service/eth"
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

// Defined types based on idWithChain should implement this interface so they may be used as logging attributes.
type IDWithChain interface {
	slog.LogValuer
	ChainID() eth.ChainID
	Kind() Kind
	Key() string
}

// idWithChain is comparable, can be copied, contains a chain-ID,
// and has type-safe text encoding/decoding to prevent accidental mixups.
type idWithChain struct {
	key     string
	chainID eth.ChainID
}

func (id idWithChain) string(kind Kind) string {
	return fmt.Sprintf("%s-%s-%s", kind, id.key, id.chainID)
}

func (id idWithChain) marshalText(kind Kind) ([]byte, error) {
	k := string(id.key)
	if len(k) > maxIDLength {
		return nil, errInvalidID
	}
	k = fmt.Sprintf("%s-%s-%s", kind, k, id.chainID)
	return []byte(k), nil
}

func (id *idWithChain) unmarshalText(kind Kind, data []byte) error {
	kindData, mainData, ok := bytes.Cut(data, []byte("-"))
	if !ok {
		return fmt.Errorf("expected kind-prefix, but id has none: %q", data)
	}
	if x := string(kindData); x != string(kind) {
		return fmt.Errorf("id %q has unexpected kind %q, expected %q", string(data), x, kind)
	}
	before, after, ok := bytes.Cut(mainData, []byte("-"))
	if !ok {
		return fmt.Errorf("expected chain separator, but found none: %q", string(data))
	}
	var chainID eth.ChainID
	if err := chainID.UnmarshalText(after); err != nil {
		return fmt.Errorf("failed to unmarshal chain part: %w", err)
	}
	if len(before) > maxIDLength {
		return errInvalidID
	}
	id.key = string(before)
	id.chainID = chainID
	return nil
}

// Defined types based on idOnlyChainID should implement this interface so they may be used as logging attributes.
type IDOnlyChainID interface {
	slog.LogValuer
	ChainID() eth.ChainID
	Kind() Kind
}

// idChainID is comparable, can be copied, contains only a chain-ID,
// and has type-safe text encoding/decoding to prevent accidental mixups.
type idOnlyChainID eth.ChainID

func (id idOnlyChainID) string(kind Kind) string {
	return fmt.Sprintf("%s-%s", kind, eth.ChainID(id))
}

func (id idOnlyChainID) marshalText(kind Kind) ([]byte, error) {
	k := fmt.Sprintf("%s-%s", kind, eth.ChainID(id))
	return []byte(k), nil
}

func (id *idOnlyChainID) unmarshalText(kind Kind, data []byte) error {
	kindData, mainData, ok := bytes.Cut(data, []byte("-"))
	if !ok {
		return fmt.Errorf("expected kind-prefix, but id has none: %q", data)
	}
	if x := string(kindData); x != string(kind) {
		return fmt.Errorf("id %q has unexpected kind %q, expected %q", string(data), x, kind)
	}
	var chainID eth.ChainID
	if err := chainID.UnmarshalText(mainData); err != nil {
		return fmt.Errorf("failed to unmarshal chain part: %w", err)
	}
	*id = idOnlyChainID(chainID)
	return nil
}

// Defined types based on genericID should implement this interface so they may be used as logging attributes.
type GenericID interface {
	slog.LogValuer
	Kind() Kind
}

// genericID is comparable, can be copied,
// and has type-safe text encoding/decoding to prevent accidental mixups.
type genericID string

func (id genericID) string(kind Kind) string {
	return fmt.Sprintf("%s-%s", kind, string(id))
}

func (id genericID) marshalText(kind Kind) ([]byte, error) {
	if len(id) > maxIDLength {
		return nil, errInvalidID
	}
	return []byte(fmt.Sprintf("%s-%s", kind, string(id))), nil
}

func (id *genericID) unmarshalText(kind Kind, data []byte) error {
	kindData, mainData, ok := bytes.Cut(data, []byte("-"))
	if !ok {
		return fmt.Errorf("expected kind-prefix, but id has none: %q", data)
	}
	if x := string(kindData); x != string(kind) {
		return fmt.Errorf("id %q has unexpected kind %q, expected %q", string(data), x, kind)
	}
	if len(mainData) > maxIDLength {
		return errInvalidID
	}
	*id = genericID(mainData)
	return nil
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

// lessIDWithChain is a helper function to compare two idWithChain objects.
// It does not use generics, since idWithChain is a concrete type with struct fields and no accessor methods in the types that wrap this type.
func lessIDWithChain(a, b idWithChain) bool {
	if a.key > b.key {
		return false
	}
	if a.key == b.key {
		return a.chainID.Cmp(b.chainID) < 0
	}
	return true
}

// lessIDOnlyChainID is a helper function to compare two idOnlyChainID objects.
func lessIDOnlyChainID(a, b idOnlyChainID) bool {
	return eth.ChainID(a).Cmp(eth.ChainID(b)) < 0
}

func lessElemOrdered[I cmp.Ordered, E Identifiable[I]](a, b E) bool {
	return a.ID() < b.ID()
}

// copyAndSortCmp is a helper function to copy and sort a slice of elements that are already natively comparable.
func copyAndSortCmp[V ~[]E, E cmp.Ordered](vs V) V {
	out := slices.Clone(vs)
	slices.Sort(out)
	return out
}
