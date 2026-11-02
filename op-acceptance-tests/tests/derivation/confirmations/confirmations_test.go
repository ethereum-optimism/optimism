package confirmations

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// capturedTest sends both in-process Go logs and the Kona subprocess logs to the
// same capture, so a brief reset cannot escape between sync-status samples.
type capturedTest struct {
	devtest.T
	logger log.Logger
}

func (t capturedTest) Logger() log.Logger { return t.logger }

func TestVerifierConfirmationsProtectDerivationFromShallowL1Reorg(gt *testing.T) {
	sysgo.SkipOnKonaNode(devtest.SerialT(gt), "pins op-node and kona-node explicitly; covered by the op-node variant")
	for _, confirmations := range []uint64{0, 4} {
		gt.Run(fmt.Sprintf("confirmations_%d", confirmations), func(gt *testing.T) {
			testShallowL1Reorg(gt, confirmations)
		})
	}
}

func testShallowL1Reorg(gt *testing.T, confirmations uint64) {
	logger, logs := testlog.CaptureLogger(gt, log.LevelInfo)
	t := capturedTest{T: devtest.SerialT(gt), logger: logger}
	runtime := sysgo.NewMixedSingleChainRuntime(t, sysgo.MixedSingleChainPresetConfig{
		NodeSpecs: []sysgo.MixedSingleChainNodeSpec{
			{ELKey: "sequencer-el", CLKey: "sequencer-cl", ELKind: sysgo.MixedL2ELOpGeth, CLKind: sysgo.MixedL2CLOpNode, IsSequencer: true},
			{ELKey: "go-el", CLKey: "go-cl", ELKind: sysgo.MixedL2ELOpGeth, CLKind: sysgo.MixedL2CLOpNode, VerifierL1Confs: confirmations},
			{ELKey: "kona-el", CLKey: "kona-cl", ELKind: sysgo.MixedL2ELOpGeth, CLKind: sysgo.MixedL2CLKona, VerifierL1Confs: confirmations},
		},
		WithTestSequencer: true,
		// Keep all manually produced empty blocks within one sequencing window.
		DeployerOptions: []sysgo.DeployerOption{sysgo.WithSequencingWindow(12)},
	})
	sys := presets.NewMixedSingleChainFrontends(t, runtime)
	verifiers := sys.Nodes[1:]
	for _, verifier := range verifiers {
		awaitStatus(t, verifier.CL, "a non-genesis safe head before the reorg", func(s *eth.SyncStatus) bool {
			return s.SafeL2.Number >= 12
		})
	}

	sys.L2Batcher.Stop()
	sys.L1CL.Stop()
	// Drain the final batch input past the delayed verifiers. The replaced tip
	// then carries no data needed to derive their existing safe heads.
	for range confirmations + 2 {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
	}
	oldTip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	confirmed := sys.L1EL.BlockRefByNumber(oldTip.Number - confirmations)
	before := make([]eth.L2BlockRef, len(verifiers))
	for i, verifier := range verifiers {
		s := awaitStatus(t, verifier.CL, "derivation at the configured confirmation distance", func(s *eth.SyncStatus) bool {
			return s.HeadL1.Hash == oldTip.Hash && s.CurrentL1.ID() == confirmed.ID()
		})
		before[i] = s.SafeL2
	}
	logs.Clear()

	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), oldTip.ParentHash)
	replacement := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	t.Require().Equal(oldTip.Number, replacement.Number, "the L1 replacement must be at the old tip's height")
	t.Require().NotEqual(oldTip.Hash, replacement.Hash, "the test must replace the old L1 block")
	t.Require().Equal(oldTip.ParentHash, replacement.ParentHash, "only the L1 tip should be replaced")
	for _, verifier := range verifiers {
		awaitStatus(t, verifier.CL, "observation of the replacement L1 head", func(s *eth.SyncStatus) bool {
			return s.HeadL1.Hash == replacement.Hash
		})
	}

	// A zero-confirmation verifier detects replacement of its consumed block
	// when reading the replacement's child. Delayed verifiers instead consume
	// the canonical replacement once it has the requested confirmations.
	for range max(confirmations, 1) {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
		tip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		confirmed = sys.L1EL.BlockRefByNumber(tip.Number - confirmations)
		for _, verifier := range verifiers {
			awaitStatus(t, verifier.CL, "derivation of the canonical L1 block at the confirmation boundary", func(s *eth.SyncStatus) bool {
				return s.HeadL1.Hash == tip.Hash && s.CurrentL1.ID() == confirmed.ID()
			})
		}
	}
	if confirmations > 0 {
		t.Require().Equal(replacement.ID(), confirmed.ID(), "the replacement must eventually become eligible for derivation")
	}
	for i, verifier := range verifiers {
		awaitStatus(t, verifier.CL, "the same safe chain after the shallow L1 reorg", func(s *eth.SyncStatus) bool {
			return s.SafeL2 == before[i]
		})
	}

	if confirmations == 0 {
		ctx, cancel := context.WithTimeout(t.Ctx(), 120*time.Second)
		defer cancel()
		err := wait.For(ctx, 100*time.Millisecond, func() (bool, error) {
			goResets, konaResets := resetCounts(logs)
			t.Logger().Info("Waiting for both clients to detect the consumed L1 tip replacement", "go_resets", goResets, "kona_resets", konaResets)
			return goResets > 0 && konaResets > 0, nil
		})
		t.Require().NoError(err, "both zero-confirmation verifiers must report their derivation reset")
	}
	goResets, konaResets := resetCounts(logs)
	t.Logger().Info("Compared derivation resets after shallow L1 reorg", "confirmations", confirmations, "go_resets", goResets, "kona_resets", konaResets)
	if confirmations == 0 {
		t.Require().Positive(goResets, "Go must reset after consuming the replaced tip at zero confirmations")
		t.Require().Positive(konaResets, "Kona must reset after consuming the replaced tip at zero confirmations")
	} else {
		t.Require().Zero(goResets, "Go must avoid a derivation reset for a reorg within its confirmation distance")
		t.Require().Zero(konaResets, "Kona must avoid a derivation reset for a reorg within its confirmation distance")
	}
}

func resetCounts(logs *testlog.CapturingHandler) (goResets, konaResets int) {
	goResets = len(logs.FindLogs(
		testlog.NewMessageFilter("Deriver system is resetting"),
		testlog.NewAttributesFilter("component", "l2cl-go-cl"),
	))
	konaResets = len(logs.FindLogs(
		testlog.NewMessageContainsFilter("Derivation pipeline is being reset"),
		testlog.NewAttributesFilter("component", "kona-node"),
	))
	return
}

// awaitStatus keeps the scenario-specific predicate here while using the shared
// bounded wait primitive. RPC failures are logged and retried without calling
// FailNow from an asynchronous Eventually callback.
func awaitStatus(t devtest.T, node *dsl.L2CLNode, expectation string, accept func(*eth.SyncStatus) bool) *eth.SyncStatus {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Ctx(), 120*time.Second)
	defer cancel()
	var last *eth.SyncStatus
	err := wait.For(ctx, 100*time.Millisecond, func() (bool, error) {
		s, err := node.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			t.Logger().Warn("Waiting for rollup sync status", "node", node, "expectation", expectation, "err", err)
			return false, nil
		}
		last = s
		t.Logger().Info("Waiting for derivation state", "node", node, "expectation", expectation, "head_l1", s.HeadL1, "current_l1", s.CurrentL1, "safe_l2", s.SafeL2)
		return accept(s), nil
	})
	require.NoError(t, err, "%s did not reach %s; last status: %+v", node, expectation, last)
	return last
}
