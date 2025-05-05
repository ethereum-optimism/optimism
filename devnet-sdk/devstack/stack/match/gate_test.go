package match

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/devtest"
)

type gateTesting struct {
	log func(format string, args ...interface{})
	hit bool
}

func (g *gateTesting) Errorf(format string, args ...interface{}) {
	g.log(format, args...)
}

func (g *gateTesting) FailNow() {
	g.hit = true
	panic("gate hit")
}

type fakeTesting struct {
	devtest.T // embedded but nil, to inherit interface
	g         *gateTesting
}

func (f *fakeTesting) Gate() *require.Assertions {
	return require.New(f.g)
}

func TestAssume(t *testing.T) {
	a := &testObject{id: "a"}
	b := &testObject{id: "b"}
	fT := &fakeTesting{T: nil, g: &gateTesting{log: t.Logf}}

	m := Assume(fT, First[testID, *testObject]())
	require.Equal(t, m.String(), "Assume(ByIndex(0))")
	require.Equal(t, []*testObject{a}, m.Match([]*testObject{a}))
	require.Equal(t, []*testObject{a}, m.Match([]*testObject{a, b}))
	require.False(t, fT.g.hit, "no skipping if we got a match")
	require.PanicsWithValue(t, "gate hit", func() {
		m.Match([]*testObject{})
	})
	require.True(t, fT.g.hit, "skip if we have no match")
}
