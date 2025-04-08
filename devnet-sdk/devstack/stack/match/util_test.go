package match

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/stack"
)

type testID string

type testObject struct {
	id testID
}

func (t *testObject) ID() testID {
	return t.id
}

var _ stack.Identifiable[testID] = (*testObject)(nil)

func TestUtils(t *testing.T) {
	a := &testObject{id: "a"}
	b := &testObject{id: "b"}
	c := &testObject{id: "c"}
	d := &testObject{id: "d"}
	all := []*testObject{a, b, c, d}

	t.Run("first", func(t *testing.T) {
		m := First[testID, *testObject]()
		t.Log(m.String())
		require.Equal(t, []*testObject{a}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, a, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, b, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("last", func(t *testing.T) {
		m := Last[testID, *testObject]()
		t.Log(m.String())
		require.Equal(t, []*testObject{d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{b, a, c}))
	})
	t.Run("random", func(t *testing.T) {
		rng := rand.New(rand.NewSource(123))
		i0 := rng.Intn(4)
		i1 := rng.Intn(4)
		rng.Seed(123) // reset
		m := Random[testID, *testObject](rng)
		t.Log(m.String())
		expected0 := all[i0]
		expected1 := all[i1]
		require.Equal(t, []*testObject{expected0}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{expected1}, m.Match([]*testObject{a, b, c, d}))
	})
	t.Run("only", func(t *testing.T) {
		m := Only[testID, *testObject]()
		t.Log(m.String())
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("combine", func(t *testing.T) {
		m := Combine(First[testID, *testObject](), Second[testID, *testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		// narrowed down to single element with First
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, a}))
		m2 := Combine(Second[testID, *testObject](), First[testID, *testObject]())
		// Narrowed down to b, then select b as first
		require.Equal(t, []*testObject{b}, m2.Match([]*testObject{a, b}))
	})
	t.Run("or", func(t *testing.T) {
		m := Or(First[testID, *testObject](), Second[testID, *testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject{a, b}, m.Match([]*testObject{a, b, c, d}))
	})
	t.Run("not", func(t *testing.T) {
		m := Not(Or(First[testID, *testObject](), Second[testID, *testObject]()))
		t.Log(m.String())
		require.Equal(t, []*testObject{c, d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{}, m.Match([]*testObject{}))
		m2 := Not(Last[testID, *testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject{a, b, c}, m2.Match([]*testObject{a, b, c, d}))
	})
	t.Run("by-index", func(t *testing.T) {
		m := ByIndex[testID, *testObject](2)
		t.Log(m.String())
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
		m2 := ByIndex[testID, *testObject](-1)
		require.Equal(t, []*testObject(nil), m2.Match([]*testObject{a, b}))
	})
}
