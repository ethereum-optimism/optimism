package genesis

import (
	"bytes"
	"math/big"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
)

// TestToBlockMatchesOpGeth checks that ToBlock computes the genesis block op-geth
// computes from the genesis that op-chain-ops builds, for a staggered schedule and for
// every timestamp fork as the latest at genesis. It is a pre-cutover check and goes away
// with op-geth.
func TestToBlockMatchesOpGeth(t *testing.T) {
	schedules := map[string]forkstest.Schedule{"staggered": forkstest.Staggered()}
	for _, fork := range forkstest.TimestampELForks {
		schedules["through "+string(fork)] = forkstest.AtGenesisThrough(fork)
	}
	allocs := map[string]func(types.GenesisAlloc){
		"message passer with storage": func(types.GenesisAlloc) {},
		"message passer without storage": func(a types.GenesisAlloc) {
			passer := a[predeploys.L2ToL1MessagePasserAddr]
			passer.Storage = nil
			a[predeploys.L2ToL1MessagePasserAddr] = passer
		},
		"no message passer": func(a types.GenesisAlloc) {
			delete(a, predeploys.L2ToL1MessagePasserAddr)
		},
	}
	for name, s := range schedules {
		for allocName, editAlloc := range allocs {
			t.Run(name+"/"+allocName, func(t *testing.T) {
				g, gg := fixture(t, s)
				editAlloc(g.Alloc)
				editAlloc(gg.Alloc)
				want := gg.ToBlock()
				got := g.ToBlock()
				require.Equal(t, want.Header(), got.Header())
				require.Equal(t, want.Hash(), got.Hash())
				require.Equal(t, want.Withdrawals(), got.Withdrawals())
			})
		}
	}
}

// TestToBlockWithdrawalsRoot pins the withdrawals root of the fixture genesis: from
// Isthmus it is the L2ToL1MessagePasser storage root, before it the empty withdrawals
// root.
func TestToBlockWithdrawalsRoot(t *testing.T) {
	g, _ := fixture(t, forkstest.AtGenesisThrough(forks.Isthmus))
	block := g.ToBlock()
	require.Equal(t,
		common.HexToHash("0xc14a972ff37a31d58badf880928348d5ce3fa02d50a3d935686d937d6b68b071"),
		*block.Header().WithdrawalsHash)
	require.Equal(t, directStorageRoot(t, fixtureAlloc()[predeploys.L2ToL1MessagePasserAddr].Storage),
		*block.Header().WithdrawalsHash)
	require.NotNil(t, block.Withdrawals())
	require.Empty(t, block.Withdrawals())

	g, _ = fixture(t, forkstest.AtGenesisThrough(forks.Holocene))
	require.Equal(t, types.EmptyWithdrawalsHash, *g.ToBlock().Header().WithdrawalsHash)
}

// TestStorageRoot checks storageRoot against a storage trie built directly from the slots:
// keccak256(slot) keys, RLP-encoded left-trimmed values, zero values absent.
func TestStorageRoot(t *testing.T) {
	addr := predeploys.L2ToL1MessagePasserAddr
	storage := map[common.Hash]common.Hash{
		{0x01}:                          {0x02},
		common.BigToHash(big.NewInt(7)): common.BigToHash(big.NewInt(0x1234)),
		{0x05}:                          {}, // zero values are not stored
	}
	require.Equal(t, directStorageRoot(t, storage), storageRoot(types.GenesisAlloc{addr: {Storage: storage}}, addr))
	require.Equal(t, types.EmptyRootHash, storageRoot(types.GenesisAlloc{addr: {}}, addr))
	require.Equal(t, common.Hash{}, storageRoot(types.GenesisAlloc{}, addr))
}

func directStorageRoot(t *testing.T, storage map[common.Hash]common.Hash) common.Hash {
	type kv struct{ k, v []byte }
	var leaves []kv
	for slot, value := range storage {
		if value == (common.Hash{}) {
			continue
		}
		v, err := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
		require.NoError(t, err)
		leaves = append(leaves, kv{crypto.Keccak256(slot[:]), v})
	}
	slices.SortFunc(leaves, func(a, b kv) int { return bytes.Compare(a.k, b.k) })
	st := trie.NewStackTrie(nil)
	for _, l := range leaves {
		require.NoError(t, st.Update(l.k, l.v))
	}
	return st.Hash()
}

// TestToBlockHash pins the fixture's genesis block hash, as op-geth computes it, so that
// the whole header stays pinned once the op-geth differential test is gone.
func TestToBlockHash(t *testing.T) {
	for _, tc := range []struct {
		name     string
		schedule forkstest.Schedule
		hash     common.Hash
	}{
		{name: "through Karst", schedule: forkstest.AtGenesisThrough(forks.Karst), hash: common.HexToHash("0xc34387921589ff94c75e83aee14e9de72e2b94f44ded3a9f20187a55b7cd5c97")},
		{name: "staggered", schedule: forkstest.Staggered(), hash: common.HexToHash("0xdf49d2e36cfe214fcadfba5131d735bf96c32c9d0cdeade41a0ef376d9f2552b")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := fixture(t, tc.schedule)
			require.Equal(t, tc.hash, g.ToBlock().Hash())
		})
	}
}
