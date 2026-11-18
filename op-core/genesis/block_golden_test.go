package genesis

import (
	"encoding/json"
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
}

// TestToBlockHashGoldenVectors pins the fixture's genesis block hash for each fork active at
// genesis and each message passer variant. The hashes are those that op-geth computed, and
// op-reth computes the same for these genesis files.
//
// A new fork fails the test until it has an entry in throughFork. Take the hashes from op-reth,
// not from this code: the failure prints the genesis of the first variant, and
// `op-reth init --chain <file>` logs the hash of the genesis block it writes.
func TestToBlockHashGoldenVectors(t *testing.T) {
	// Each pair of hashes is named after the last fork that changes the fixture's genesis block;
	// the forks after it, up to the next name, give the same block. The hashes are in the order
	// of messagePasserVariants.
	var (
		bedrock = [2]common.Hash{common.HexToHash("0xbfb63475f2ba5ddd1e2981745f3cb9612d8134fb4280f224620e0f6739a99a20"), common.HexToHash("0x6521f0610747b787789a78eb78aad9e9fc62776b75a71adc4bedd22d787782bc")}
		canyon  = [2]common.Hash{common.HexToHash("0x7a5f390c5507f9c6446dbdbc4f6466cb2c82f12ee65142afdb4071a119e3480b"), common.HexToHash("0xf2933b70faa68d188171df99c39b76136cc47976a590edc4f8fe1375781eaf5b")}
		ecotone = [2]common.Hash{common.HexToHash("0xfaff5be8c9bb92871a97c1d2f2f2f18dab0ef66914f5f582fb9d0e41ee2f6f40"), common.HexToHash("0xcfc53f7d3fe68fbb144d9ca0bd09e9501a2a01dff2b92437125a0425d64a8c70")}
		isthmus = [2]common.Hash{common.HexToHash("0xc34387921589ff94c75e83aee14e9de72e2b94f44ded3a9f20187a55b7cd5c97"), common.HexToHash("0x09cea63168c49973e437f28491302ea30a2f24d4bf20f77d3a3c7924529c07ab")}
	)
	// The genesis block hashes with the forks up to and including the key active at genesis.
	throughFork := map[forks.Name][2]common.Hash{
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
	check := func(t *testing.T, name string, schedule forkstest.Schedule, hashes [2]common.Hash) {
		for i, variant := range messagePasserVariants {
			t.Run(name+"/"+variant.name, func(t *testing.T) {
				g := fixture(schedule)
				variant.editAlloc(g.Alloc)
				require.Equal(t, hashes[i], g.ToBlock().Hash())
			})
		}
	}
	for _, fork := range forkstest.TimestampELForks {
		schedule := forkstest.AtGenesisThrough(fork)
		hashes, ok := throughFork[fork]
		if !ok {
			data, err := json.Marshal(fixture(schedule))
			require.NoError(t, err)
			t.Fatalf("no golden genesis block hashes for fork %s; add op-reth's hashes for this genesis and its variants:\n%s", fork, data)
		}
		check(t, "through "+string(fork), schedule, hashes)
	}
	// A fork that activates after genesis leaves the genesis block unchanged.
	check(t, "staggered", forkstest.Staggered(), bedrock)
}
