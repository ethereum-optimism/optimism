package prune

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/op-rs/op-geth/proofs/utils"
	"github.com/stretchr/testify/require"
)

// Steps:
// 1) Create some tx and validate proof for a block (pre-prune):
// 2) Wait for that specific block to be pruned:
//   - Ensure the chain advances enough so the pruner *can* move `earliest` past `targetBlock`
//     (i.e. latest >= targetBlock + proofWindow).
//   - Poll debug_proofsSyncStatus until earliest > targetBlock (meaning targetBlock is now pruned).
//
// 3) Call validate checks for getProof and check everything is consistent for the new earliest block.
func TestPruneProofStorageWithGetProofConsistency(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := utils.NewMixedOpProofPreset(t)

	// Defined in the devnet yaml
	var proofWindow = uint64(200)

	// An expected time within the prune should be detected.
	var pruneDetectTimeout = 5 * time.Minute

	opRethELNode := sys.RethWithProofL2ELNode()
	ethClient := opRethELNode.Escape().EthClient()

	// -----------------------------
	// (1) Create tx + validate proof pre-prune
	// -----------------------------
	const numAccounts = 2
	const initialFunding = 10

	accounts := sys.FunderL2.NewFundedEOAs(numAccounts, eth.Ether(initialFunding))
	recipient := sys.FunderL2.NewFundedEOA(eth.Ether(1))
	recipientAddr := recipient.Address()
	transferAmount := eth.Ether(1)

	t.Log("Sending transactions from both accounts (to create state changes)")
	tx0 := accounts[0].Transfer(recipientAddr, transferAmount)
	tx1 := accounts[1].Transfer(recipientAddr, transferAmount)

	receipt0, err := tx0.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt0.Status)

	receipt1, err := tx1.Included.Eval(ctx)
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, receipt1.Status)

	// Choose a deterministic target block: the later of the two inclusion blocks.
	targetBlock := receipt0.BlockNumber.Uint64()
	if receipt1.BlockNumber.Uint64() > targetBlock {
		targetBlock = receipt1.BlockNumber.Uint64()
	}
	t.Logf("Target block for proof validation (pre-prune): %d", targetBlock)

	// Make sure validator has the block too (keeps the test stable).
	sys.L2ELValidatorNode().WaitForBlockNumber(targetBlock)

	// Pre-prune proof verification at targetBlock.
	// This verifies the proof against the block's state root (efficient correctness check).
	t.Logf("Pre-prune: verifying getProof proofs at block %d", targetBlock)
	utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, targetBlock)
	utils.FetchAndVerifyProofs(t, sys, accounts[1].Address(), []common.Hash{}, targetBlock)
	t.Log("Pre-prune: proofs verified successfully")

	// -----------------------------
	// (2) Wait until targetBlock is pruned (earliest > targetBlock)
	// -----------------------------
	initialStatus := getProofSyncStatus(t, ethClient)
	t.Logf("Initial proofs sync status: earliest=%d latest=%d", initialStatus.Earliest, initialStatus.Latest)

	// Ensure we advance far enough that pruning *can* move earliest past targetBlock.
	// If latest < targetBlock + proofWindow, earliest cannot advance beyond targetBlock yet.
	requiredLatest := targetBlock + proofWindow
	if initialStatus.Latest < requiredLatest {
		t.Logf("Waiting for chain to advance to at least block %d so pruning can pass targetBlock", requiredLatest)
		opRethELNode.WaitForBlockNumber(requiredLatest)
	}

	t.Logf("Waiting for pruner to advance earliest past targetBlock=%d ...", targetBlock)
	waitUntil := time.Now().Add(pruneDetectTimeout)

	var prunedStatus proofSyncStatus
	for {
		if time.Now().After(waitUntil) {
			t.Errorf("Timed out waiting for prune: earliest did not advance past targetBlock=%d within %s", targetBlock, pruneDetectTimeout)
		}

		prunedStatus = getProofSyncStatus(t, ethClient)
		t.Logf("Polling proofs sync status: earliest=%d latest=%d (target=%d)", prunedStatus.Earliest, prunedStatus.Latest, targetBlock)

		// This is the key condition: the specific block we validated earlier is now out of window.
		if prunedStatus.Earliest > targetBlock {
			break
		}

		time.Sleep(5 * time.Second)
	}

	currentProofWindow := prunedStatus.Latest - prunedStatus.Earliest
	require.GreaterOrEqual(t, currentProofWindow, proofWindow, "pruner should maintain at least the configured proof window")
	t.Logf("Detected prune past targetBlock. Now earliest=%d latest=%d window=%d", prunedStatus.Earliest, prunedStatus.Latest, currentProofWindow)

	// -----------------------------
	// (3) Post-prune consistency checks for getProof
	// -----------------------------
	t.Logf("Post-prune: expecting getProof verification to succeed at new earliest block=%d", prunedStatus.Earliest)
	utils.FetchAndVerifyProofs(t, sys, accounts[0].Address(), []common.Hash{}, prunedStatus.Earliest)
	utils.FetchAndVerifyProofs(t, sys, accounts[1].Address(), []common.Hash{}, prunedStatus.Earliest)
	t.Log("Post-prune: getProof consistency checks passed")
}

type proofSyncStatus struct {
	Earliest uint64 `json:"earliest"`
	Latest   uint64 `json:"latest"`
}

func getProofSyncStatus(t devtest.T, client apis.EthClient) proofSyncStatus {
	var result proofSyncStatus
	err := client.RPC().CallContext(t.Ctx(), &result, "debug_proofsSyncStatus")
	if err != nil {
		t.Errorf("debug_proofsSyncStatus call failed: %v", err)
	}
	return result
}
