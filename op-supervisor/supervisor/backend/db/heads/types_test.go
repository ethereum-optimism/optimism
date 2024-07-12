package heads

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

func TestHeads(t *testing.T) {
	t.Run("RoundTripViaJson", func(t *testing.T) {
		heads := NewHeads()
		heads.Put(types.ChainIDFromUInt64(3), ChainHeads{
			Unsafe:         10,
			CrossUnsafe:    9,
			LocalSafe:      8,
			CrossSafe:      7,
			LocalFinalized: 6,
			CrossFinalized: 5,
		})
		heads.Put(types.ChainIDFromUInt64(9), ChainHeads{
			Unsafe:         90,
			CrossUnsafe:    80,
			LocalSafe:      70,
			CrossSafe:      60,
			LocalFinalized: 50,
			CrossFinalized: 40,
		})
		heads.Put(types.ChainIDFromUInt64(4892497242424), ChainHeads{
			Unsafe:         1000,
			CrossUnsafe:    900,
			LocalSafe:      800,
			CrossSafe:      700,
			LocalFinalized: 600,
			CrossFinalized: 400,
		})

		j, err := json.Marshal(heads)
		require.NoError(t, err)

		fmt.Println(string(j))
		var result Heads
		err = json.Unmarshal(j, &result)
		require.NoError(t, err)
		require.Equal(t, heads.Chains, result.Chains)
	})

	t.Run("Copy", func(t *testing.T) {
		chainA := types.ChainIDFromUInt64(3)
		chainB := types.ChainIDFromUInt64(4)
		chainAOrigHeads := ChainHeads{
			Unsafe: 1,
		}
		chainAModifiedHeads1 := ChainHeads{
			Unsafe: 2,
		}
		chainAModifiedHeads2 := ChainHeads{
			Unsafe: 4,
		}
		chainBModifiedHeads := ChainHeads{
			Unsafe: 2,
		}

		heads := NewHeads()
		heads.Put(chainA, chainAOrigHeads)

		otherHeads := heads.Copy()
		otherHeads.Put(chainA, chainAModifiedHeads1)
		otherHeads.Put(chainB, chainBModifiedHeads)

		require.Equal(t, heads.Get(chainA), chainAOrigHeads)
		require.Equal(t, heads.Get(chainB), ChainHeads{})

		heads.Put(chainA, chainAModifiedHeads2)
		require.Equal(t, heads.Get(chainA), chainAModifiedHeads2)

		require.Equal(t, otherHeads.Get(chainA), chainAModifiedHeads1)
		require.Equal(t, otherHeads.Get(chainB), chainBModifiedHeads)
	})
}
