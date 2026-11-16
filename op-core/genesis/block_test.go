package genesis

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	for name, s := range schedules {
		for _, variant := range messagePasserVariants {
			t.Run(name+"/"+variant.name, func(t *testing.T) {
				g, gg := fixture(t, s)
				variant.editAlloc(g.Alloc)
				variant.editAlloc(gg.Alloc)
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
	require.NotNil(t, block.Withdrawals())
	require.Empty(t, block.Withdrawals())

	g, _ = fixture(t, forkstest.AtGenesisThrough(forks.Holocene))
	require.Equal(t, types.EmptyWithdrawalsHash, *g.ToBlock().Header().WithdrawalsHash)
}

// TestStorageRoot pins storageRoot to the roots go-ethereum's state database computes for the
// same storage.
func TestStorageRoot(t *testing.T) {
	addr := predeploys.L2ToL1MessagePasserAddr
	storage := map[common.Hash]common.Hash{
		{0x01}:                          {0x02},
		common.BigToHash(big.NewInt(7)): common.BigToHash(big.NewInt(0x1234)),
		{0x05}:                          {}, // zero values are not stored
	}
	require.Equal(t,
		common.HexToHash("0x494d8ce4a991aa3310a547052a1b56e0ff645cec6095e068842fadcbfa209948"),
		storageRoot(types.GenesisAlloc{addr: {Storage: storage}}, addr))
	many := make(map[common.Hash]common.Hash)
	for i := range int64(300) {
		many[common.BigToHash(big.NewInt(i))] = common.BigToHash(big.NewInt(i*i + 1))
	}
	require.Equal(t,
		common.HexToHash("0x41b76207fef93e9417eb978dbe52a985974570bb80fb092cd4365257ec7f4069"),
		storageRoot(types.GenesisAlloc{addr: {Storage: many}}, addr))
	require.Equal(t, types.EmptyRootHash, storageRoot(types.GenesisAlloc{addr: {}}, addr))
	require.Equal(t, common.Hash{}, storageRoot(types.GenesisAlloc{}, addr))
}

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

// TestToBlockHash pins the fixture's genesis block hash, as op-geth computes it, for each
// fork schedule and message passer variant, so that the whole header stays pinned.
func TestToBlockHash(t *testing.T) {
	var (
		regolith = [3]string{"0xdf49d2e36cfe214fcadfba5131d735bf96c32c9d0cdeade41a0ef376d9f2552b", "0xbc5cd0145d18c13d3862becb4dad58bd67a78541af82a870d9137c957f02deed", "0x8f055549a49446224eaebf8171de2356244a9f711b54e935b60d620d9e8f2ef6"}
		canyon   = [3]string{"0xf5958adaa7fd29176e3d570556bc07aed9866e6c926d60bd13f357caf41786a6", "0x1ac74e3350b16cd021faa0fedc10f949e415c9dd14e134809ef40c1393af2e63", "0xa3c1fce102847a02cdbe56b8b24401a91458fb83552cd6a5aa64e3bc7451be33"}
		ecotone  = [3]string{"0xfaff5be8c9bb92871a97c1d2f2f2f18dab0ef66914f5f582fb9d0e41ee2f6f40", "0xcfc53f7d3fe68fbb144d9ca0bd09e9501a2a01dff2b92437125a0425d64a8c70", "0xbf6c0445c38d50a88b105b0dae826cc68750135812e1a8db3fdafe762fea7ce9"}
		isthmus  = [3]string{"0xc34387921589ff94c75e83aee14e9de72e2b94f44ded3a9f20187a55b7cd5c97", "0x09cea63168c49973e437f28491302ea30a2f24d4bf20f77d3a3c7924529c07ab", "0xf7b148c6fe3baa92ea48179acb3fb3d901ee5c79403da3a810cd0ad5590e5a2b"}
	)
	for _, tc := range []struct {
		name     string
		schedule forkstest.Schedule
		// hashes are in the order of messagePasserVariants.
		hashes [3]string
	}{
		{name: "staggered", schedule: forkstest.Staggered(), hashes: regolith},
		{name: "through Regolith", schedule: forkstest.AtGenesisThrough(forks.Regolith), hashes: regolith},
		{name: "through Canyon", schedule: forkstest.AtGenesisThrough(forks.Canyon), hashes: canyon},
		{name: "through Ecotone", schedule: forkstest.AtGenesisThrough(forks.Ecotone), hashes: ecotone},
		{name: "through Fjord", schedule: forkstest.AtGenesisThrough(forks.Fjord), hashes: ecotone},
		{name: "through Granite", schedule: forkstest.AtGenesisThrough(forks.Granite), hashes: ecotone},
		{name: "through Holocene", schedule: forkstest.AtGenesisThrough(forks.Holocene), hashes: ecotone},
		{name: "through Isthmus", schedule: forkstest.AtGenesisThrough(forks.Isthmus), hashes: isthmus},
		{name: "through Jovian", schedule: forkstest.AtGenesisThrough(forks.Jovian), hashes: isthmus},
		{name: "through Karst", schedule: forkstest.AtGenesisThrough(forks.Karst), hashes: isthmus},
		{name: "through Lagoon", schedule: forkstest.AtGenesisThrough(forks.Lagoon), hashes: isthmus},
	} {
		for i, variant := range messagePasserVariants {
			t.Run(tc.name+"/"+variant.name, func(t *testing.T) {
				g, _ := fixture(t, tc.schedule)
				variant.editAlloc(g.Alloc)
				require.Equal(t, common.HexToHash(tc.hashes[i]), g.ToBlock().Hash())
			})
		}
	}
}
