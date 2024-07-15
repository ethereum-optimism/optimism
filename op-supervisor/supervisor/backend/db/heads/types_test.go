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
			Unsafe:         ChainHead{Index: 10, ID: 100},
			CrossUnsafe:    ChainHead{Index: 9, ID: 99},
			LocalSafe:      ChainHead{Index: 8, ID: 98},
			CrossSafe:      ChainHead{Index: 7, ID: 97},
			LocalFinalized: ChainHead{Index: 6, ID: 96},
			CrossFinalized: ChainHead{Index: 5, ID: 95},
		})
		heads.Put(types.ChainIDFromUInt64(9), ChainHeads{
			Unsafe:         ChainHead{Index: 90, ID: 9},
			CrossUnsafe:    ChainHead{Index: 80, ID: 8},
			LocalSafe:      ChainHead{Index: 70, ID: 7},
			CrossSafe:      ChainHead{Index: 60, ID: 6},
			LocalFinalized: ChainHead{Index: 50, ID: 5},
			CrossFinalized: ChainHead{Index: 40, ID: 4},
		})
		heads.Put(types.ChainIDFromUInt64(4892497242424), ChainHeads{
			Unsafe:         ChainHead{Index: 1000, ID: 11},
			CrossUnsafe:    ChainHead{Index: 900, ID: 22},
			LocalSafe:      ChainHead{Index: 800, ID: 33},
			CrossSafe:      ChainHead{Index: 700, ID: 44},
			LocalFinalized: ChainHead{Index: 600, ID: 55},
			CrossFinalized: ChainHead{Index: 400, ID: 66},
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
			Unsafe: ChainHead{Index: 1, ID: 10},
		}
		chainAModifiedHeads1 := ChainHeads{
			Unsafe: ChainHead{Index: 2, ID: 20},
		}
		chainAModifiedHeads2 := ChainHeads{
			Unsafe: ChainHead{Index: 4, ID: 40},
		}
		chainBModifiedHeads := ChainHeads{
			Unsafe: ChainHead{Index: 2, ID: 50},
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
