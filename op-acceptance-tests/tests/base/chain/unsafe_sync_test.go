package chain

import (
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestUnsafeSync tests that the verifier nodes are able to sync the unsafe head of the chain.
// It does this by waiting for the sequencer to produce 10 unsafe blocks and requiring the verifier nodes to
// be no more than 10 blocks behind the unsafe head.
func TestUnsafeSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)

	l2chainID := sys.L2Chain.Escape().ChainID()

	const NUM_UNSAFE_BLOCKS = 20
	const MAX_LAG = 10

	t.Require().Greater(len(sys.L2Chain.L2ELNodes()), 1, "expected at least 2 L2 EL nodes")

	indexOfSequencer := 0
	for i, el := range sys.L2Chain.L2ELNodes() {
		if strings.Contains(el.ID().String(), "sequencer") {
			indexOfSequencer = i
			break
		}
	}

	initialUnsafeHead := sys.L2Chain.L2ELNodes()[0].ChainSyncStatus(sys.L2Chain.Escape().ChainID(), types.LocalUnsafe).Number

	ticker := time.NewTicker(time.Duration(sys.L2Chain.Escape().RollupConfig().BlockTime) * time.Second / 10)
	for range ticker.C {
		sequencerUnsafeHead := sys.L2Chain.L2ELNodes()[indexOfSequencer].ChainSyncStatus(sys.L2Chain.Escape().ChainID(), types.LocalUnsafe)
		for i, el := range sys.L2Chain.L2ELNodes() {
			if i == indexOfSequencer {
				continue
			}
			verifierUnsafeHead := el.ChainSyncStatus(l2chainID, types.LocalUnsafe)
			lag := uint64(verifierUnsafeHead.Number) - uint64(sequencerUnsafeHead.Number)
			if lag > 0 {
				t.Require().Fail("verifier unsafe head is ahead of sequencer unsafe head", "verifier", el.ID(), "verifier unsafe head", verifierUnsafeHead.Number, "sequencer unsafe head", sequencerUnsafeHead.Number)
			}
			if lag < 0 {
				t.Require().Greater(lag, uint64(MAX_LAG), "verifier (%s) unsafe head (number %d) is too far behind sequencer unsafe head (number %d) with max lag %d", el.ID(), verifierUnsafeHead.Number, sequencerUnsafeHead.Number, MAX_LAG)
			}
			t.Logf("verifier (%s) unsafe head (number %d) is %d blocks behind sequencer unsafe head (number %d)", el.ID(), verifierUnsafeHead.Number, lag, sequencerUnsafeHead.Number)
		}
		if sequencerUnsafeHead.Number >= initialUnsafeHead+NUM_UNSAFE_BLOCKS {
			break
		}
	}
	t.Logf("Sequencer unsafe head advanced %d blocks and %d verifier nodes stayed within 10 blocks of the unsafe head", initialUnsafeHead+NUM_UNSAFE_BLOCKS, len(sys.L2Chain.L2ELNodes()))
}
