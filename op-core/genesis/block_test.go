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

// TestToBlockWithdrawalsRoot pins the withdrawals root of the fixture genesis: from
// Isthmus it is the L2ToL1MessagePasser storage root, before it the empty withdrawals
// root.
func TestToBlockWithdrawalsRoot(t *testing.T) {
	g := fixture(forkstest.AtGenesisThrough(forks.Isthmus))
	block := g.ToBlock()
	require.Equal(t,
		common.HexToHash("0xc14a972ff37a31d58badf880928348d5ce3fa02d50a3d935686d937d6b68b071"),
		*block.Header().WithdrawalsHash)
	require.NotNil(t, block.Withdrawals())
	require.Empty(t, block.Withdrawals())

	g = fixture(forkstest.AtGenesisThrough(forks.Holocene))
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
	require.Equal(t, types.EmptyRootHash, storageRoot(types.GenesisAlloc{}, addr))
}
