package core

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/rust/op-reth/tests/proofs/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestL2MultipleTransactionsInDifferentBlocks tests transactions from different accounts
// on L2 across multiple blocks. This verifies account state changes across multiple L2 blocks.
// Check if the proof retrieved from geth and reth match for each account at each block height,
// and verify the proofs against the respective block state roots.
func TestL2MultipleTransactionsInDifferentBlocks(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()
	sys := utils.NewMixedOpProofPreset(t)

	const numAccounts = 2
	const initialFunding = 10
	accounts := sys.FunderL2.NewFundedEOAs(numAccounts, eth.Ether(initialFunding))

	recipient := sys.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()

	// Block 1: Send transaction from first account
	currentBlock := sys.L2ELSequencerNode().WaitForBlock()
	t.Logf("Current L2 block number: %d", currentBlock.Number)

	transferAmount := eth.Ether(1)
	tx1 := accounts[0].Transfer(recipientAddr, transferAmount)
	t.Logf("Sent transaction from account 0: %s", accounts[0].Address().Hex())
	receipt1, err := tx1.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt1.Status)
	t.Logf("Transaction 1 included in block: %d", bigs.Uint64Strict(receipt1.BlockNumber))

	sys.L2ELValidatorNode().WaitForBlockNumber(bigs.Uint64Strict(receipt1.BlockNumber))
	utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, bigs.Uint64Strict(receipt1.BlockNumber))
	sys.L2ELSequencerNode().WaitForBlockNumber(currentBlock.Number + 1)

	// Block 2: Send transaction from second account
	currentBlock = sys.L2ELSequencerNode().WaitForBlock()
	t.Logf("Current L2 block number: %d", currentBlock.Number)

	tx2 := accounts[1].Transfer(recipientAddr, transferAmount)
	t.Logf("Sent transaction from account 1: %s", accounts[1].Address().Hex())
	receipt2, err := tx2.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt2.Status)
	t.Logf("Transaction 2 included in block: %d", bigs.Uint64Strict(receipt2.BlockNumber))

	sys.L2ELValidatorNode().WaitForBlockNumber(bigs.Uint64Strict(receipt2.BlockNumber))
	utils.FetchAndVerifyProofs(t, sys, accounts[1].Address(), []common.Hash{}, bigs.Uint64Strict(receipt2.BlockNumber))

	// Also verify we can get proofs for account 0 at block 2 (different block height)
	utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, bigs.Uint64Strict(receipt2.BlockNumber))
}

// TestL2MultipleTransactionsInSingleBlock tests 2 different accounts sending transactions
// that get included in the same L2 block.
// It verifies that the account proofs for both accounts can be retrieved and verified
// against the same block's state root, and that the proofs from geth and reth match.
func TestL2MultipleTransactionsInSingleBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()
	sys := utils.NewMixedOpProofPreset(t)

	const numAccounts = 2
	const initialFunding = 10
	accounts := sys.FunderL2.NewFundedEOAs(numAccounts, eth.Ether(initialFunding))

	recipient := sys.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()

	transferAmount := eth.Ether(1)

	t.Log("Sending transactions from both accounts")
	tx0 := accounts[0].Transfer(recipientAddr, transferAmount)
	t.Logf("Sent transaction from account 0: %s", accounts[0].Address().Hex())

	tx1 := accounts[1].Transfer(recipientAddr, transferAmount)
	t.Logf("Sent transaction from account 1: %s", accounts[1].Address().Hex())

	// Wait for both transactions to be included
	receipt0, err := tx0.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt0.Status)
	t.Logf("Transaction 0 included in block %d", bigs.Uint64Strict(receipt0.BlockNumber))

	receipt1, err := tx1.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt1.Status)
	t.Logf("Transaction 1 included in block %d", bigs.Uint64Strict(receipt1.BlockNumber))

	sys.L2ELValidatorNode().WaitForBlockNumber(bigs.Uint64Strict(receipt1.BlockNumber))
	// Txns can land in the same or different blocks depending on timing.
	if bigs.Uint64Strict(receipt0.BlockNumber) == bigs.Uint64Strict(receipt1.BlockNumber) {
		t.Logf("Both transactions included in the same L2 block: %d", bigs.Uint64Strict(receipt0.BlockNumber))

		// Verify both proofs against the same block state root
		utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, bigs.Uint64Strict(receipt0.BlockNumber))
		utils.FetchAndVerifyProofs(t, sys, accounts[1].Address(), []common.Hash{}, bigs.Uint64Strict(receipt0.BlockNumber))

	} else {
		t.Logf("Transactions in different blocks: %d and %d",
			bigs.Uint64Strict(receipt0.BlockNumber), bigs.Uint64Strict(receipt1.BlockNumber))

		// Different blocks: verify each proof's merkle root matches its respective block's state root
		utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, bigs.Uint64Strict(receipt0.BlockNumber))
		utils.FetchAndVerifyProofs(t, sys, accounts[1].Address(), []common.Hash{}, bigs.Uint64Strict(receipt1.BlockNumber))
	}

	t.Logf("Proof for account 0 and 1 verified successfully")
}
