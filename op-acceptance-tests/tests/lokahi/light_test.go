package lokahi

import (
	"math/rand"
	"slices"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestLightNodesSafeHeadsFollowLokahi checks chains whose sequencer and follower are light nodes,
// which take their safe labels from the chain's source: with a valid cross-chain message, every
// node makes the message's blocks safe and the nodes agree on the safe chain. With lokahi down, no
// safe label moves, while the sources keep deriving and the light nodes' unsafe heads keep moving.
func TestLightNodesSafeHeadsFollowLokahi(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiLight(t)
	nodes := slices.Concat(sys.ChainA.CLs(), sys.ChainB.CLs())

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	initMsg := alice.SendRandomInitMessage(rand.New(rand.NewSource(2468)), eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendExecMessage(initMsg)
	t.Logger().Info("Valid message sent", "init", initMsg.BlockID(), "exec", execMsg.BlockID())

	var checks []dsl.CheckFunc
	for _, node := range sys.ChainA.CLs() {
		checks = append(checks, node.ReachedRefFn(safety.CrossSafe, initMsg.BlockID(), 120))
	}
	for _, node := range sys.ChainB.CLs() {
		checks = append(checks, node.ReachedRefFn(safety.CrossSafe, execMsg.BlockID(), 120))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)

	sys.Lokahi.Stop()
	checks = nil
	for _, node := range nodes {
		node.WaitForStall(safety.CrossSafe)
		checks = append(checks, node.NotAdvancedFn(safety.CrossSafe, 20))
	}
	for _, chain := range []*presets.LokahiChainNodes{&sys.ChainA, &sys.ChainB} {
		checks = append(checks,
			chain.Source.AdvancedFn(safety.LocalSafe, 5, 60),
			chain.Sequencer.AdvancedFn(safety.LocalUnsafe, 5, 60),
			chain.Follower.AdvancedFn(safety.LocalUnsafe, 5, 60),
		)
	}
	dsl.CheckAll(t, checks...)
	held := logHeads(t, "lokahi stopped", nodes)

	sys.Lokahi.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, safeReachesWithinLocalSafe(t, node, held[node]+3, 3*time.Minute))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)
	logHeads(t, "lokahi restarted", nodes)
}

// TestLightNodesInvalidMessageReplacesBlock checks that a block with an invalid executing message,
// built by a light sequencer, is replaced on every node of its chain by the deposits-only block the
// source derives: the light nodes move onto it once the source's safe head passes it, and the light
// sequencer keeps building on it.
func TestLightNodesInvalidMessageReplacesBlock(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiLight(t)
	chainB := &sys.ChainB

	alice := sys.FunderA.NewFundedEOA(eth.OneEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneEther)
	eventLoggerA := alice.DeployEventLogger()
	sys.L2B.CatchUpTo(sys.L2A)
	sys.L2A.CatchUpTo(sys.L2B)
	initMsg := alice.SendRandomInitMessage(rand.New(rand.NewSource(1357)), eventLoggerA, 2, 10)
	sys.L2B.WaitForBlock()
	execMsg := bob.SendInvalidExecMessage(initMsg)
	denied := execMsg.BlockID()
	t.Logger().Info("Invalid executing message included", "block", denied, "tx", execMsg.TxHash())

	replacement := awaitReplaced(t, chainB.SourceEL, denied, 3*time.Minute)
	for _, el := range []*dsl.L2ELNode{chainB.SequencerEL, chainB.FollowerEL} {
		t.Require().Equal(replacement, awaitReplaced(t, el, denied, 3*time.Minute),
			"%s must replace the denied block with the source's replacement", el)
	}
	var checks []dsl.CheckFunc
	for _, node := range chainB.CLs() {
		checks = append(checks, node.ReachedFn(safety.CrossSafe, denied.Number+1, 120))
	}
	dsl.CheckAll(t, checks...)
	t.Logger().Info("Denied block replaced on every node", "denied", denied, "replacement", replacement)

	chainB.Sequencer.Advanced(safety.LocalUnsafe, 5, 30)
	chainB.Follower.MatchedUnsafe(chainB.Sequencer, 60)
	dsl.CheckAll(t, chainB.Source.AdvancedFn(safety.CrossSafe, 3, 120))
	for _, el := range chainB.ELs() {
		assertReplacement(t, el, denied, replacement, execMsg.TxHash())
	}
	assertSafeHeadsAgree(t, sys.Lokahi, chainB)
	logHeads(t, "denied block replaced", chainB.CLs())
}

// TestLightNodesSourceOutage checks that, with one chain's source down, no safe label moves on
// either chain, while that chain's light sequencer keeps building blocks and its light follower
// follows them. Once the source is back, every safe label moves on and the nodes agree.
func TestLightNodesSourceOutage(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2LokahiLight(t)
	nodes := slices.Concat(sys.ChainA.CLs(), sys.ChainB.CLs())
	running := []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainA.Follower,
		sys.ChainB.Sequencer, sys.ChainB.Source, sys.ChainB.Follower}
	sequencers := []*dsl.L2CLNode{sys.ChainA.Sequencer, sys.ChainB.Sequencer}

	var checks []dsl.CheckFunc
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 3, 120))
	}
	dsl.CheckAll(t, checks...)

	sys.ChainA.Source.Stop()
	for _, node := range running {
		node.WaitForStall(safety.CrossSafe)
	}
	held := logHeads(t, "source stopped", running)
	verified := verifiedHeads(t, sys.Lokahi, sequencers)
	checks = []dsl.CheckFunc{
		sys.ChainA.Sequencer.AdvancedFn(safety.LocalUnsafe, 10, 60),
		sys.ChainA.Follower.AdvancedFn(safety.LocalUnsafe, 10, 60),
	}
	for _, node := range running {
		checks = append(checks, node.NotAdvancedFn(safety.CrossSafe, 15))
	}
	dsl.CheckAll(t, checks...)
	sys.ChainA.Follower.MatchedUnsafe(sys.ChainA.Sequencer, 30)
	for node, head := range verifiedHeads(t, sys.Lokahi, sequencers) {
		t.Require().Equalf(verified[node], head, "lokahi must wait for chain %s's source", node.ChainID())
	}
	for node, safeHead := range held {
		t.Require().Equalf(safeHead, node.SyncStatus().SafeL2.Number, "%s must hold its safe label", node)
	}

	sys.ChainA.Source.Start()
	checks = nil
	for _, node := range nodes {
		checks = append(checks, node.AdvancedFn(safety.CrossSafe, 5, 180))
	}
	dsl.CheckAll(t, checks...)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainA)
	assertSafeHeadsAgree(t, sys.Lokahi, &sys.ChainB)
	logHeads(t, "source restarted", nodes)
}
