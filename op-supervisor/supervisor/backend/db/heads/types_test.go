package heads

import (
	"encoding/json"
	"fmt"
	"math/rand" // nosemgrep
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestHeads(t *testing.T) {
	rng := rand.New(rand.NewSource(1234))
	randHeadPtr := func() HeadPointer {
		var h common.Hash
		rng.Read(h[:])
		return HeadPointer{
			LastSealedBlockHash: h,
			LastSealedBlockNum:  rng.Uint64(),
			LogsSince:           rng.Uint32(),
		}
	}
	t.Run("RoundTripViaJson", func(t *testing.T) {
		heads := NewHeads()
		heads.Put(types.ChainIDFromUInt64(3), ChainHeads{
			Unsafe:         randHeadPtr(),
			CrossUnsafe:    randHeadPtr(),
			LocalSafe:      randHeadPtr(),
			CrossSafe:      randHeadPtr(),
			LocalFinalized: randHeadPtr(),
			CrossFinalized: randHeadPtr(),
		})
		heads.Put(types.ChainIDFromUInt64(9), ChainHeads{
			Unsafe:         randHeadPtr(),
			CrossUnsafe:    randHeadPtr(),
			LocalSafe:      randHeadPtr(),
			CrossSafe:      randHeadPtr(),
			LocalFinalized: randHeadPtr(),
			CrossFinalized: randHeadPtr(),
		})
		heads.Put(types.ChainIDFromUInt64(4892497242424), ChainHeads{
			Unsafe:         randHeadPtr(),
			CrossUnsafe:    randHeadPtr(),
			LocalSafe:      randHeadPtr(),
			CrossSafe:      randHeadPtr(),
			LocalFinalized: randHeadPtr(),
			CrossFinalized: randHeadPtr(),
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
			Unsafe: randHeadPtr(),
		}
		chainAModifiedHeads1 := ChainHeads{
			Unsafe: randHeadPtr(),
		}
		chainAModifiedHeads2 := ChainHeads{
			Unsafe: randHeadPtr(),
		}
		chainBModifiedHeads := ChainHeads{
			Unsafe: randHeadPtr(),
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
