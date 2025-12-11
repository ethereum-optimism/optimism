package prune

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/op-rs/op-geth/proofs/utils"
	"github.com/stretchr/testify/require"
)

func TestPruneProofStorage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)

	var proofWindow = uint64(200)            // Defined in the devnet yaml
	var pruneDetectTimeout = time.Minute * 5 // An expected time within the prune should be detected.
	opRethELNode, _ := utils.IdentifyELNodes(sys.L2EL, sys.L2ELB)

	syncStatus := getProofSyncStatus(t, opRethELNode.Escape().EthClient())
	t.Log("Initial sync status:", syncStatus)
	distance := syncStatus.Latest - syncStatus.Earliest

	if distance < proofWindow {
		// Wait till we reach proof window
		t.Logf("Waiting for block %d", syncStatus.Earliest+proofWindow)
		opRethELNode.WaitForBlockNumber(syncStatus.Earliest + proofWindow)
	}
	// Now we need to wait for pruner to execute pruning, which can be done anytime within 1 minute (pruner prune interval = 1 min)
	startTime := time.Now()
	var newSyncStatus proofSyncStatus
	for {
		// Get sync status each Second
		if time.Since(startTime) > pruneDetectTimeout {
			t.Error("Pruner did not prune proof storage within the interval")
			return
		}
		newSyncStatus = getProofSyncStatus(t, opRethELNode.Escape().EthClient())
		if syncStatus.Earliest != newSyncStatus.Earliest {
			break
		}
		t.Log("Waiting on earliest state to be changed: ", syncStatus.Earliest)
		time.Sleep(time.Second * 5)
	}
	// Check how many has been pruned -  we should have current proof window intake
	currentProofWindow := newSyncStatus.Latest - newSyncStatus.Earliest
	t.Log("Sync status:", syncStatus)
	require.GreaterOrEqual(t, currentProofWindow, proofWindow, "Pruner has changed the proof window")
	t.Logf("Successfully pruned proof storage. sync status: %v", syncStatus)
}

type proofSyncStatus struct {
	Earliest uint64 `json:"earliest"`
	Latest   uint64 `json:"latest"`
}

func getProofSyncStatus(t devtest.T, client apis.EthClient) proofSyncStatus {
	var result proofSyncStatus
	err := client.RPC().CallContext(t.Ctx(), &result, "debug_proofsSyncStatus")
	if err != nil {
		t.Error(err)
	}
	return result
}
