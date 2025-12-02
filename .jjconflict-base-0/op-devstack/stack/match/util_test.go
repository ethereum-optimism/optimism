package match

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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

	t.Run("first", func(t *testing.T) {
		m := First[testID, *testObject]()
		require.Equal(t, m.String(), "ByIndex(0)")
		require.Equal(t, []*testObject{a}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, a, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, b, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("last", func(t *testing.T) {
		m := Last[testID, *testObject]()
		require.Equal(t, m.String(), "Last")
		require.Equal(t, []*testObject{d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{b, a, c}))
	})
	t.Run("only", func(t *testing.T) {
		m := Only[testID, *testObject]()
		t.Log(m.String())
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("and", func(t *testing.T) {
		m := And(First[testID, *testObject](), Second[testID, *testObject]())
		require.Equal(t, m.String(), "And(ByIndex(0), ByIndex(1))")
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		// narrowed down to single element with First
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, a}))
		m2 := And(Second[testID, *testObject](), First[testID, *testObject]())
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
		require.Equal(t, m.String(), "Not(Or(ByIndex(0), ByIndex(1)))")
		require.Equal(t, []*testObject{c, d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{}, m.Match([]*testObject{}))
		m2 := Not(Last[testID, *testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject{a, b, c}, m2.Match([]*testObject{a, b, c, d}))
	})
	t.Run("by-index", func(t *testing.T) {
		m := ByIndex[testID, *testObject](2)
		require.Equal(t, m.String(), "ByIndex(2)")
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
		m2 := ByIndex[testID, *testObject](-1)
		require.Equal(t, []*testObject(nil), m2.Match([]*testObject{a, b}))
	})
}
