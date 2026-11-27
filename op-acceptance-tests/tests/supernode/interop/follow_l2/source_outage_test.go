package follow_l2

import (
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestLightSequencerHoldsSafeWhileFollowSourceIsDown checks that, with one chain's follow source
// down, no safe label moves on either chain, while both light sequencers keep building blocks.
// Once the source is back, every safe label moves on, and each light sequencer agrees with its
// source on the safe chain.
//
// The light sequencers sequence from the start, so the restarted source does not sequence too.
func TestLightSequencerHoldsSafeWhileFollowSourceIsDown(gt *testing.T) {
	t := devtest.ParallelT(gt)
	if sysgo.ResolveSupernodeKind(t) != sysgo.SupernodeLokahi {
		t.Skip("op-supernode derives every chain in one process, so one chain's follow source cannot stop alone")
	}
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0)
	sys.L2ACL.ManagePeer(sys.L2ASupernodeCL)
	sys.L2BCL.ManagePeer(sys.L2BSupernodeCL)
	nodes := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL, sys.L2ASupernodeCL, sys.L2BSupernodeCL}
	running := []*dsl.L2CLNode{sys.L2ACL, sys.L2BCL, sys.L2BSupernodeCL}

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 3, 120))
	}
	dsl.CheckAll(t, checks...)

	sys.L2ASupernodeCL.Stop()
	held := make(map[*dsl.L2CLNode]uint64, len(running))
	for _, node := range running {
		node.WaitForStall(safety.CrossSafe)
		held[node] = node.SyncStatus().SafeL2.Number
	}
	t.Logger().Info("Follow source of chain A stopped", "held", held)
	checks = []dsl.CheckFunc{
		sys.L2ACL.AdvancedFn(safety.LocalUnsafe, 10, 60),
		sys.L2BCL.AdvancedFn(safety.LocalUnsafe, 10, 60),
	}
	for _, node := range running {
		checks = append(checks, node.NotAdvancedFn(safety.CrossSafe, 15))
	}
	dsl.CheckAll(t, checks...)
	for node, safeHead := range held {
		t.Require().Equalf(safeHead, node.SyncStatus().SafeL2.Number, "%s must hold its safe label", node)
	}

	sys.L2ASupernodeCL.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 180))
	}
	dsl.CheckAll(t, checks...)
	for _, chain := range []struct {
		sequencer, source     *dsl.L2CLNode
		sequencerEL, sourceEL *dsl.L2ELNode
	}{
		{sys.L2ACL, sys.L2ASupernodeCL, sys.L2ELA, sys.L2ASupernodeEL},
		{sys.L2BCL, sys.L2BSupernodeCL, sys.L2ELB, sys.L2BSupernodeEL},
	} {
		common := slices.Min([]uint64{chain.sequencer.SyncStatus().SafeL2.Number, chain.source.SyncStatus().SafeL2.Number})
		t.Require().Equalf(chain.sourceEL.BlockRefByNumber(common).ID(), chain.sequencerEL.BlockRefByNumber(common).ID(),
			"%s and its follow source must agree at their common safe height %d", chain.sequencer, common)
	}
}
