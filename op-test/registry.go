package op_test

import (
	"sync"

	"github.com/stretchr/testify/require"
)

type registry struct {
	data sync.Map
}

func (imp *testImpl) SetGlobal(name string, value any) {
	prev, loaded := imp.registry.data.Swap(name, value)
	if loaded {
		require.Equal(imp.T, prev, value, "already known global conflicts with new global")
	}
}

func (imp *testImpl) GetGlobal(name string) any {
	v, ok := imp.registry.data.Load(name)
	require.True(imp.T, ok, "requested global value must be known")
	return v
}

// Global retrieves a test value with GetGlobal and ensures typing.
func Global[E any](t Testing, name string) E {
	x := t.GetGlobal(name)
	out, ok := x.(E)
	require.True(t, ok, "global value must be of type %T but is %T", out, x)
	return out
}
