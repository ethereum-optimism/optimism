package match

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
)

type testObject struct {
	id stack.ComponentID
}

func (t *testObject) ID() stack.ComponentID {
	return t.id
}

var _ stack.Identifiable = (*testObject)(nil)

func newTestObject(key string) *testObject {
	return &testObject{id: stack.NewComponentIDKeyOnly(stack.KindL2ELNode, key)}
}

func TestUtils(t *testing.T) {
	a := newTestObject("a")
	b := newTestObject("b")
	c := newTestObject("c")
	d := newTestObject("d")

	t.Run("first", func(t *testing.T) {
		m := First[*testObject]()
		require.Equal(t, m.String(), "ByIndex(0)")
		require.Equal(t, []*testObject{a}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, a, c, d}))
		require.Equal(t, []*testObject{b}, m.Match([]*testObject{b, b, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("last", func(t *testing.T) {
		m := Last[*testObject]()
		require.Equal(t, m.String(), "Last")
		require.Equal(t, []*testObject{d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{b, a, c}))
	})
	t.Run("only", func(t *testing.T) {
		m := Only[*testObject]()
		t.Log(m.String())
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
	})
	t.Run("and", func(t *testing.T) {
		m := And(First[*testObject](), Second[*testObject]())
		require.Equal(t, m.String(), "And(ByIndex(0), ByIndex(1))")
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b, c, d}))
		// narrowed down to single element with First
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, a}))
		m2 := And(Second[*testObject](), First[*testObject]())
		// Narrowed down to b, then select b as first
		require.Equal(t, []*testObject{b}, m2.Match([]*testObject{a, b}))
	})
	t.Run("or", func(t *testing.T) {
		m := Or(First[*testObject](), Second[*testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject{a, b}, m.Match([]*testObject{a, b, c, d}))
	})
	t.Run("not", func(t *testing.T) {
		m := Not(Or(First[*testObject](), Second[*testObject]()))
		require.Equal(t, m.String(), "Not(Or(ByIndex(0), ByIndex(1)))")
		require.Equal(t, []*testObject{c, d}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{}, m.Match([]*testObject{}))
		m2 := Not(Last[*testObject]())
		t.Log(m.String())
		require.Equal(t, []*testObject{a, b, c}, m2.Match([]*testObject{a, b, c, d}))
	})
	t.Run("by-index", func(t *testing.T) {
		m := ByIndex[*testObject](2)
		require.Equal(t, m.String(), "ByIndex(2)")
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c, d}))
		require.Equal(t, []*testObject{c}, m.Match([]*testObject{a, b, c}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a, b}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{a}))
		require.Equal(t, []*testObject(nil), m.Match([]*testObject{}))
		m2 := ByIndex[*testObject](-1)
		require.Equal(t, []*testObject(nil), m2.Match([]*testObject{a, b}))
	})
}
