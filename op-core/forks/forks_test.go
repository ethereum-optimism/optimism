package forks

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNext(t *testing.T) {
	tests := []struct {
		name     string
		fork     Name
		expected Name
	}{
		{"first fork", Bedrock, Regolith},
		{"middle fork", Ecotone, Fjord},
		{"second-to-last", Karst, Interop},
		{"last fork returns None", Interop, None},
		{"unknown fork returns None", Name("unknown"), None},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, Next(tc.fork))
		})
	}
}

func TestPrev(t *testing.T) {
	tests := []struct {
		name     string
		fork     Name
		expected Name
	}{
		{"first fork returns None", Bedrock, None},
		{"second fork", Regolith, Bedrock},
		{"middle fork", Fjord, Ecotone},
		{"last fork", Interop, Karst},
		{"unknown fork returns None", Name("unknown"), None},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, Prev(tc.fork))
		})
	}
}

func TestNextPrevInverse(t *testing.T) {
	// For every mainline fork that is not the first, Prev(Next(prev)) == prev,
	// and for every fork that is not the last, Next(Prev(next)) == next.
	// This guards against the two maps drifting out of sync.
	for i, f := range All {
		if i < len(All)-1 {
			require.Equal(t, f, Prev(Next(f)), "Prev(Next(%s)) must equal %s", f, f)
		}
		if i > 0 {
			require.Equal(t, f, Next(Prev(f)), "Next(Prev(%s)) must equal %s", f, f)
		}
	}
}

func TestFrom(t *testing.T) {
	t.Run("from first returns all", func(t *testing.T) {
		require.Equal(t, All, From(Bedrock))
	})
	t.Run("from middle returns tail", func(t *testing.T) {
		got := From(Ecotone)
		require.Equal(t, Ecotone, got[0])
		require.Equal(t, All[len(All)-1], got[len(got)-1])
		require.Len(t, got, len(All)-4) // Bedrock, Regolith, Canyon, Delta excluded
	})
	t.Run("from last returns single", func(t *testing.T) {
		require.Equal(t, []Name{Interop}, From(Interop))
	})
	t.Run("unknown fork panics", func(t *testing.T) {
		require.Panics(t, func() { From(Name("unknown")) })
	})
}
