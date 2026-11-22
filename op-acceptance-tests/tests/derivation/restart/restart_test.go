package restart

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

const (
	seqWindow = 8
	// Each verifier derives only from L1 blocks this deep, so its safe head trails the gossiped
	// unsafe head by more than a sequencing window of L1 origins. The margin over the window keeps
	// a safe head that a reset moved up shallower than this for about a dozen L1 blocks, so the
	// check below sees it.
	l1Confs = 24
)

// A restart resets the CL's forkchoice. When derivation trails the unsafe head by more than a
// sequencing window, the reset must not label safe the unsafe blocks that derivation has not
// reached: a block is safe only once it is derived from L1 data, and a verifier that waits for
// l1Confs confirmations cannot have derived a block whose L1 origin is shallower than that.
func TestRestartKeepsUnderivedBlocksUnsafe(gt *testing.T) {
	t := devtest.ParallelT(gt)
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{ELKey: "sequencer-el", CLKey: "sequencer-cl", CLKind: sysgo.MixedL2CLOpNode, IsSequencer: true},
			{ELKey: "go-el", CLKey: "go-cl", CLKind: sysgo.MixedL2CLOpNode, VerifierL1Confs: l1Confs},
			{ELKey: "kona-el", CLKey: "kona-cl", CLKind: sysgo.MixedL2CLKona, VerifierL1Confs: l1Confs},
		},
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithSequencingWindow(seqWindow)},
	})
	sys := presets.NewMixedSingleChainFrontends(t, runtime)

	for _, verifier := range sys.Nodes[1:] {
		// A finalized head beyond genesis makes the restart an ordinary reset rather than the
		// completion of EL sync.
		awaitStatus(t, verifier.CL, "a finalized head, and an unsafe head more than a sequencing window ahead of the safe head", func(s *eth.SyncStatus) bool {
			return s.FinalizedL2.Number > 0 && s.UnsafeL2.L1Origin.Number > s.SafeL2.L1Origin.Number+seqWindow
		})

		// Discovery is off, so peer the restarted CL with the sequencer again. kona-node completes
		// EL sync from gossip before its first reset.
		verifier.CL.ManagePeer(sys.Nodes[0].CL)
		verifier.CL.Stop()
		// The safe head the reset starts from.
		previousSafe := verifier.EL.BlockRefByLabel(eth.Safe)
		verifier.CL.Start()

		awaitStatus(t, verifier.CL, "derivation past the safe head from before the restart", func(s *eth.SyncStatus) bool {
			l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)
			t.Require().LessOrEqual(s.SafeL2.L1Origin.Number+l1Confs, l1Head.Number,
				"%s labelled L2 block %s safe, but its L1 origin %d has fewer than %d confirmations on L1 head %d, so derivation cannot have reached it",
				verifier.CL, s.SafeL2, s.SafeL2.L1Origin.Number, l1Confs, l1Head.Number)
			return s.SafeL2.Number > previousSafe.Number
		})
	}
}

// awaitStatus polls the CL's sync status until accept returns true. RPC errors, as while the CL
// restarts, are logged and retried.
func awaitStatus(t devtest.T, node *dsl.L2CLNode, expectation string, accept func(*eth.SyncStatus) bool) *eth.SyncStatus {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Ctx(), 3*time.Minute)
	defer cancel()
	var last *eth.SyncStatus
	err := wait.For(ctx, 100*time.Millisecond, func() (bool, error) {
		s, err := node.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			t.Logger().Warn("Waiting for rollup sync status", "node", node, "expectation", expectation, "err", err)
			return false, nil
		}
		last = s
		t.Logger().Info("Waiting for sync status", "node", node, "expectation", expectation,
			"head_l1", s.HeadL1, "current_l1", s.CurrentL1, "unsafe_l2", s.UnsafeL2, "safe_l2", s.SafeL2, "finalized_l2", s.FinalizedL2)
		return accept(s), nil
	})
	require.NoError(t, err, "%s did not reach %s; last status: %+v", node, expectation, last)
	return last
}
