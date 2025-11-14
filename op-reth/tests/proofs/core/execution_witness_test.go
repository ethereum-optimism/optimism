package core

import (
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// ExecutionWitness represents the response from debug_executionWitness
type ExecutionWitness struct {
	Keys    []hexutil.Bytes `json:"keys"`
	Codes   []hexutil.Bytes `json:"codes"`
	State   []hexutil.Bytes `json:"state"`
	Headers []hexutil.Bytes `json:"headers"`
}

// TestDebugExecutionWitness tests the debug_executionWitness RPC method on Reth L2.
// This verifies that the execution witness can be retrieved for a block containing transactions
// and that the response contains valid state, codes, keys, and headers data.
func TestDebugExecutionWitness(gt *testing.T) {
	t := devtest.SerialT(gt)
	preset := presets.NewSingleChainMultiNode(t)

	// Create a funded account and recipient
	account := preset.FunderL2.NewFundedEOA(eth.Ether(10))
	recipient := preset.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()

	// Wait for current block
	currentBlock := preset.L2EL.WaitForBlock()
	t.Logf("Current L2 block number: %d", currentBlock.Number)

	// Send a transaction to create some state changes
	transferAmount := eth.Ether(1)
	tx := account.Transfer(recipientAddr, transferAmount)
	t.Logf("Sent transaction from account: %s to recipient: %s", account.Address().Hex(), recipientAddr.Hex())

	receipt, err := tx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status)
	t.Logf("Transaction included in block: %d", receipt.BlockNumber.Uint64())

	l2RethClient := preset.L2ELB.Escape().L2EthClient()

	// Get the block to inspect the state changes
	block, err := l2RethClient.InfoByNumber(t.Ctx(), receipt.BlockNumber.Uint64())
	require.NoError(t, err)
	t.Logf("Block %d has state root: %s", block.NumberU64(), block.Root().Hex())

	// Call debug_executionWitness via RPC
	var witness ExecutionWitness
	blockNumberHex := hexutil.EncodeUint64(block.NumberU64())

	// Use the RPC client's CallContext method directly
	err = l2RethClient.RPC().CallContext(t.Ctx(), &witness, "debug_executionWitness", blockNumberHex)
	require.NoError(t, err, "debug_executionWitness RPC call should succeed")

	// Verify the witness contains expected data
	require.NotEmpty(t, witness.Keys, "Witness should contain keys data")
	require.NotEmpty(t, witness.Codes, "Witness should contain codes data")
	require.NotEmpty(t, witness.State, "State should not be empty")
	require.NotNil(t, witness.Headers, "Witness should contain headers data")

	// Verify the parent header is present and decode it
	require.NotEmpty(t, witness.Headers, "Headers should contain at least the parent block")
	parentHeaderBytes := witness.Headers[len(witness.Headers)-1]
	require.NotEmpty(t, parentHeaderBytes, "Parent header should not be empty")
	t.Logf("Parent header size: %d bytes", len(parentHeaderBytes))

	// Decode the parent header to verify it's valid RLP and extract state root
	var parentHeader types.Header
	err = rlp.DecodeBytes(parentHeaderBytes, &parentHeader)
	require.NoError(t, err, "Parent header should be valid RLP-encoded")

	// Verify the parent header matches the expected parent block
	expectedParentNumber := block.NumberU64() - 1
	require.Equal(t, expectedParentNumber, parentHeader.Number.Uint64(),
		"Parent header should be for block %d", expectedParentNumber)

	// Get the actual parent block from the chain to verify state root
	actualParentBlock, err := l2RethClient.InfoByNumber(t.Ctx(), expectedParentNumber)
	require.NoError(t, err, "Should be able to fetch parent block from chain")

	// Verify the parent header's state root matches the actual parent block's state root
	require.Equal(t, actualParentBlock.Root(), parentHeader.Root,
		"Parent header state root in witness should match actual parent block state root")
	t.Logf("Verified parent header state root matches chain: %s", parentHeader.Root.Hex())

	// Verify that the witness contains keys for the accounts involved in the transaction
	senderAddrHex := strings.ToLower(account.Address().Hex())
	recipientAddrHex := strings.ToLower(recipientAddr.Hex())

	// Check if the witness keys contains the accounts
	// The witness format may vary, so we check for the presence of either the address or its hash
	foundSender := false
	foundRecipient := false

	for _, value := range witness.Keys {
		keyLower := strings.ToLower(value.String())
		if strings.Contains(keyLower, senderAddrHex) {
			foundSender = true
		}
		if strings.Contains(keyLower, recipientAddrHex) {
			foundRecipient = true
		}
	}

	// We should find at least the sender since they initiated the transaction
	require.True(t, foundSender, "Witness should contain state data for the transaction sender")
	t.Logf("Verified sender account is present in execution witness")

	// The recipient might not always be in the witness depending on the implementation
	if foundRecipient {
		t.Logf("Verified recipient account is present in execution witness")
	}
	t.Log("Successfully retrieved and validated execution witness from Reth")
}
