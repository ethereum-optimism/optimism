package core

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// Helper to convert block number to hex string for eth_getProof
func toBlockTag(blockNumber *big.Int) string {
	return fmt.Sprintf("0x%x", blockNumber)
}

// TestL2MultipleTransactionsInDifferentBlocks tests transactions from different accounts
// on L2 across multiple blocks. This verifies account state changes across multiple L2 blocks.
// Check if the proof retrieved from geth and reth match for each account at each block height,
// and verify the proofs against the respective block state roots.
func TestL2MultipleTransactionsInDifferentBlocks(gt *testing.T) {
	dt := devtest.SerialT(gt)
	preset := presets.NewSingleChainMultiNode(dt)

	const numAccounts = 2
	const initialFunding = 10
	accounts := preset.FunderL2.NewFundedEOAs(numAccounts, eth.Ether(initialFunding))

	recipient := preset.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()

	// Block 1: Send transaction from first account
	currentBlock := preset.L2EL.WaitForBlock()
	gt.Logf("Current L2 block number: %d", currentBlock.Number)

	transferAmount := eth.Ether(1)
	tx1 := accounts[0].Transfer(recipientAddr, transferAmount)
	gt.Logf("Sent transaction from account 0: %s", accounts[0].Address().Hex())
	receipt1, err := tx1.Included.Eval(dt.Ctx())
	require.NoError(gt, err)
	require.Equal(gt, types.ReceiptStatusSuccessful, receipt1.Status)
	gt.Logf("Transaction 1 included in block: %d", receipt1.BlockNumber.Uint64())

	// Query account proof for account 0 at block 1 from geth and reth
	l2GethClient := preset.L2EL.Escape().L2EthClient()
	blockTag := toBlockTag(receipt1.BlockNumber)
	proof0Geth, err := l2GethClient.GetProof(dt.Ctx(), accounts[0].Address(), []common.Hash{}, blockTag)
	require.NoError(gt, err)

	l2RethClient := preset.L2ELB.Escape().L2EthClient()
	proof0Reth, err := l2RethClient.GetProof(dt.Ctx(), accounts[0].Address(), []common.Hash{}, blockTag)
	require.NoError(gt, err)
	require.Equal(gt, proof0Geth, proof0Reth, "Geth and Reth proofs for account 0 at block %s should match", blockTag)

	// Get the block to verify proof against its state root
	block1, err := l2RethClient.InfoByNumber(dt.Ctx(), receipt1.BlockNumber.Uint64())
	require.NoError(gt, err)

	// Verify reth proof merkle root matches block state root
	err = proof0Reth.Verify(block1.Root())
	require.NoError(gt, err, "Proof for account 0 should verify against block %d state root", block1.NumberU64())
	gt.Logf("Proof for account 0 verified successfully against block %d state root", block1.NumberU64())

	preset.L2EL.WaitForBlockNumber(currentBlock.Number + 1)

	// Block 2: Send transaction from second account
	currentBlock = preset.L2EL.WaitForBlock()
	gt.Logf("Current L2 block number: %d", currentBlock.Number)

	tx2 := accounts[1].Transfer(recipientAddr, transferAmount)
	gt.Logf("Sent transaction from account 1: %s", accounts[1].Address().Hex())
	receipt2, err := tx2.Included.Eval(dt.Ctx())
	require.NoError(gt, err)
	require.Equal(gt, types.ReceiptStatusSuccessful, receipt2.Status)
	gt.Logf("Transaction 2 included in block: %d", receipt2.BlockNumber.Uint64())

	// Query account proof for account 1 at block 2
	blockTag2 := toBlockTag(receipt2.BlockNumber)
	proof1Geth, err := l2GethClient.GetProof(dt.Ctx(), accounts[1].Address(), []common.Hash{}, blockTag2)
	require.NoError(gt, err)
	proof1Reth, err := l2RethClient.GetProof(dt.Ctx(), accounts[1].Address(), []common.Hash{}, blockTag2)
	require.NoError(gt, err)
	require.Equal(gt, proof1Geth, proof1Reth, "Geth and Reth proofs for account 1 at block %s should match", blockTag2)

	// Get block 2 to verify proof against its state root
	block2, err := l2RethClient.InfoByNumber(dt.Ctx(), receipt2.BlockNumber.Uint64())
	require.NoError(gt, err)

	// Verify proof1 merkle root matches block2 state root
	err = proof1Reth.Verify(block2.Root())
	require.NoError(gt, err, "Proof for account 1 should verify against block %d state root", block2.NumberU64())
	gt.Logf("Proof for account 1 verified successfully against block %d state root", block2.NumberU64())

	// Also verify we can get proofs for account 0 at block 2 (different block height)
	proof0AtBlock2, err := l2RethClient.GetProof(dt.Ctx(), accounts[0].Address(), []common.Hash{}, blockTag2)
	require.NoError(gt, err)

	// Verify proof0AtBlock2 merkle root matches block2 state root
	err = proof0AtBlock2.Verify(block2.Root())
	require.NoError(gt, err, "Proof for account 0 at block %d should verify against block %d state root", block2.NumberU64(), block2.NumberU64())
	gt.Logf("Proof for account 0 at block %d verified successfully", block2.NumberU64())
}

// TestL2MultipleTransactionsInSingleBlock tests 2 different accounts sending transactions
// that get included in the same L2 block.
// It verifies that the account proofs for both accounts can be retrieved and verified
// against the same block's state root, and that the proofs from geth and reth match.
func TestL2MultipleTransactionsInSingleBlock(gt *testing.T) {
	dt := devtest.SerialT(gt)
	preset := presets.NewSingleChainMultiNode(dt)

	const numAccounts = 2
	const initialFunding = 10
	accounts := preset.FunderL2.NewFundedEOAs(numAccounts, eth.Ether(initialFunding))

	recipient := preset.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()

	transferAmount := eth.Ether(1)

	gt.Log("Sending transactions from both accounts")
	tx0 := accounts[0].Transfer(recipientAddr, transferAmount)
	gt.Logf("Sent transaction from account 0: %s", accounts[0].Address().Hex())

	tx1 := accounts[1].Transfer(recipientAddr, transferAmount)
	gt.Logf("Sent transaction from account 1: %s", accounts[1].Address().Hex())

	// Wait for both transactions to be included
	receipt0, err := tx0.Included.Eval(dt.Ctx())
	require.NoError(gt, err)
	require.Equal(gt, types.ReceiptStatusSuccessful, receipt0.Status)
	gt.Logf("Transaction 0 included in block %d", receipt0.BlockNumber.Uint64())

	receipt1, err := tx1.Included.Eval(dt.Ctx())
	require.NoError(gt, err)
	require.Equal(gt, types.ReceiptStatusSuccessful, receipt1.Status)
	gt.Logf("Transaction 1 included in block %d", receipt1.BlockNumber.Uint64())

	// Query account proofs for both accounts at the block(s) they were included
	l2GethClient := preset.L2EL.Escape().L2EthClient()
	l2RethClient := preset.L2ELB.Escape().L2EthClient()

	blockTag0 := toBlockTag(receipt0.BlockNumber)
	proof0Geth, err := l2GethClient.GetProof(dt.Ctx(), accounts[0].Address(), []common.Hash{}, blockTag0)
	require.NoError(gt, err)
	proof0Reth, err := l2RethClient.GetProof(dt.Ctx(), accounts[0].Address(), []common.Hash{}, blockTag0)
	require.NoError(gt, err)

	blockTag1 := toBlockTag(receipt1.BlockNumber)
	proof1Geth, err := l2GethClient.GetProof(dt.Ctx(), accounts[1].Address(), []common.Hash{}, blockTag1)
	require.NoError(gt, err)
	proof1Reth, err := l2RethClient.GetProof(dt.Ctx(), accounts[1].Address(), []common.Hash{}, blockTag1)
	require.NoError(gt, err)

	// Ensure geth and reth proofs match for both accounts
	require.Equal(gt, proof0Geth, proof0Reth, "Geth and Reth proofs for account 0 should match")
	require.Equal(gt, proof1Geth, proof1Reth, "Geth and Reth proofs for account 1 should match")

	// Txns can land in the same or different blocks depending on timing.
	if receipt0.BlockNumber.Uint64() == receipt1.BlockNumber.Uint64() {
		gt.Logf("Both transactions included in the same L2 block: %d", receipt0.BlockNumber.Uint64())

		// Same block: verify both proofs' merkle roots match that block's state root
		block, err := l2RethClient.InfoByNumber(dt.Ctx(), receipt0.BlockNumber.Uint64())
		require.NoError(gt, err)

		// Verify both proofs against the same block state root
		err = proof0Reth.Verify(block.Root())
		require.NoError(gt, err, "Proof for account 0 should verify against block %d state root", block.NumberU64())
		gt.Logf("Proof for account 0 verified successfully")

		err = proof1Reth.Verify(block.Root())
		require.NoError(gt, err, "Proof for account 1 should verify against block %d state root", block.NumberU64())
		gt.Logf("Proof for account 1 verified successfully")
	} else {
		gt.Logf("Transactions in different blocks: %d and %d",
			receipt0.BlockNumber.Uint64(), receipt1.BlockNumber.Uint64())

		// Different blocks: verify each proof's merkle root matches its respective block's state root
		block0, err := l2RethClient.InfoByNumber(dt.Ctx(), receipt0.BlockNumber.Uint64())
		require.NoError(gt, err)

		err = proof0Reth.Verify(block0.Root())
		require.NoError(gt, err, "Proof for account 0 should verify against block %d state root", block0.NumberU64())
		gt.Logf("Proof for account 0 verified successfully against block %d", block0.NumberU64())

		block1, err := l2RethClient.InfoByNumber(dt.Ctx(), receipt1.BlockNumber.Uint64())
		require.NoError(gt, err)

		err = proof1Reth.Verify(block1.Root())
		require.NoError(gt, err, "Proof for account 1 should verify against block %d state root", block1.NumberU64())
		gt.Logf("Proof for account 1 verified successfully against block %d", block1.NumberU64())
	}
}
