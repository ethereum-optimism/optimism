package chain

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// TestUnsafeSync tests that the verifier nodes are able to sync the unsafe head of the chain.
// It does this by waiting for the sequencer to produce 10 unsafe blocks and requiring the verifier nodes to
// be no more than 10 blocks behind the unsafe head.
func TestUnsafeSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)
	l2chainID := sys.L2Chain.Escape().ChainID()
	const NUM_UNSAFE_BLOCKS = 10

	// In order for this test to be valid, we need at least 2 L2 EL nodes.
	t.Require().Greater(len(sys.L2Chain.L2ELNodes()), 1, "expected at least 2 L2 EL nodes")

	// Find the index of the sequencer node.
	indexOfSequencer := 0
	for i, el := range sys.L2Chain.L2ELNodes() {
		if strings.Contains(el.ID().String(), "sequencer") {
			indexOfSequencer = i
			break
		}
	}

	// Wait for the sequencer to produce at least NUM_UNSAFE_BLOCKS unsafe blocks.
	initialUnsafeHeadNumber := sys.L2Chain.L2ELNodes()[0].ChainSyncStatus(sys.L2Chain.Escape().ChainID(), types.LocalUnsafe).Number
	finalUnsafeHeadNumber := initialUnsafeHeadNumber + NUM_UNSAFE_BLOCKS
	ticker := time.NewTicker(time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second)
	for range ticker.C {
		sequencerUnsafeHead := sys.L2Chain.L2ELNodes()[indexOfSequencer].ChainSyncStatus(sys.L2Chain.Escape().ChainID(), types.LocalUnsafe)
		t.Logf("Sequencer unsafe head (number %d) is %d/%d blocks ahead of initial unsafe head (number %d)",
			sequencerUnsafeHead.Number, sequencerUnsafeHead.Number-initialUnsafeHeadNumber, NUM_UNSAFE_BLOCKS, initialUnsafeHeadNumber)
		if sequencerUnsafeHead.Number >= finalUnsafeHeadNumber {
			break
		}
	}

	// Check verifier nodes progess to the final unsafe head number.
	allVerifierNodesInSync := func() bool {
		for i, el := range sys.L2Chain.L2ELNodes() {
			if i == indexOfSequencer {
				continue
			}
			verifierUnsafeHead := el.ChainSyncStatus(l2chainID, types.LocalUnsafe)
			if verifierUnsafeHead.Number < initialUnsafeHeadNumber+NUM_UNSAFE_BLOCKS {
				return false
			}
		}
		return true
	}
	require.Eventually(t, allVerifierNodesInSync, 30*time.Second, time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime)*time.Second)
	t.Log("All verifier nodes progressed to the final unsafe head number", finalUnsafeHeadNumber)

	// Final sanity check on the block hashes being consistent.
	finalUnsafeBlockHash := sys.L2Chain.L2ELNodes()[indexOfSequencer].BlockRefByNumber(finalUnsafeHeadNumber).Hash
	for i, el := range sys.L2Chain.L2ELNodes() {
		if i == indexOfSequencer {
			continue
		}
		verifierUnsafeBlockHash := el.BlockRefByNumber(finalUnsafeHeadNumber).Hash
		if verifierUnsafeBlockHash != finalUnsafeBlockHash {
			t.Require().Fail("verifier (%s) unsafe block hash (%s) does not match final unsafe block hash (%s)", el.ID(), verifierUnsafeBlockHash, finalUnsafeBlockHash)
		}
	}
	t.Log("All verifier nodes have the final unsafe block hash", finalUnsafeBlockHash)
}
