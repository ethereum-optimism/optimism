package genesis

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// messagePasserVariants edit the fixture alloc. From Isthmus the withdrawals root is the
// L2ToL1MessagePasser storage root.
var messagePasserVariants = []struct {
	name      string
	editAlloc func(types.GenesisAlloc)
}{
	{name: "message passer with storage", editAlloc: func(types.GenesisAlloc) {}},
	{name: "message passer without storage", editAlloc: func(a types.GenesisAlloc) {
		passer := a[predeploys.L2ToL1MessagePasserAddr]
		passer.Storage = nil
		a[predeploys.L2ToL1MessagePasserAddr] = passer
	}},
	{name: "no message passer", editAlloc: func(a types.GenesisAlloc) {
		delete(a, predeploys.L2ToL1MessagePasserAddr)
	}},
}

// TestToBlockHashGoldenVectors pins the fixture's genesis block hash, as op-geth computes
// it, for each fork schedule and message passer variant.
func TestToBlockHashGoldenVectors(t *testing.T) {
	// Each set of hashes is named after the last fork that changes the fixture's genesis block;
	// the forks after it, up to the next name, give the same block. The hashes are in the order
	// of messagePasserVariants.
	var (
		bedrock = [3]string{"0xdf49d2e36cfe214fcadfba5131d735bf96c32c9d0cdeade41a0ef376d9f2552b", "0xbc5cd0145d18c13d3862becb4dad58bd67a78541af82a870d9137c957f02deed", "0x8f055549a49446224eaebf8171de2356244a9f711b54e935b60d620d9e8f2ef6"}
		canyon  = [3]string{"0xf5958adaa7fd29176e3d570556bc07aed9866e6c926d60bd13f357caf41786a6", "0x1ac74e3350b16cd021faa0fedc10f949e415c9dd14e134809ef40c1393af2e63", "0xa3c1fce102847a02cdbe56b8b24401a91458fb83552cd6a5aa64e3bc7451be33"}
		ecotone = [3]string{"0xfaff5be8c9bb92871a97c1d2f2f2f18dab0ef66914f5f582fb9d0e41ee2f6f40", "0xcfc53f7d3fe68fbb144d9ca0bd09e9501a2a01dff2b92437125a0425d64a8c70", "0xbf6c0445c38d50a88b105b0dae826cc68750135812e1a8db3fdafe762fea7ce9"}
		isthmus = [3]string{"0xc34387921589ff94c75e83aee14e9de72e2b94f44ded3a9f20187a55b7cd5c97", "0x09cea63168c49973e437f28491302ea30a2f24d4bf20f77d3a3c7924529c07ab", "0xf7b148c6fe3baa92ea48179acb3fb3d901ee5c79403da3a810cd0ad5590e5a2b"}
	)
	// The genesis block hashes with the forks up to and including the key active at genesis.
	throughFork := map[forks.Name][3]string{
		forks.Regolith: bedrock,
		forks.Canyon:   canyon,
		forks.Ecotone:  ecotone,
		forks.Fjord:    ecotone,
		forks.Granite:  ecotone,
		forks.Holocene: ecotone,
		forks.Isthmus:  isthmus,
		forks.Jovian:   isthmus,
		forks.Karst:    isthmus,
		forks.Lagoon:   isthmus,
	}
	check := func(t *testing.T, name string, schedule forkstest.Schedule, hashes [3]string) {
		for i, variant := range messagePasserVariants {
			t.Run(name+"/"+variant.name, func(t *testing.T) {
				g := fixture(schedule)
				variant.editAlloc(g.Alloc)
				require.Equal(t, common.HexToHash(hashes[i]), g.ToBlock().Hash())
			})
		}
	}
	for _, fork := range forkstest.TimestampELForks {
		hashes, ok := throughFork[fork]
		require.True(t, ok, "no golden genesis block hashes for fork %s", fork)
		check(t, "through "+string(fork), forkstest.AtGenesisThrough(fork), hashes)
	}
	// A fork that activates after genesis leaves the genesis block unchanged.
	check(t, "staggered", forkstest.Staggered(), bedrock)
}
