package el_restart

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestVerifierResumesWithoutResetAfterELRestart restarts a verifier's EL over its persistent
// datadir while the verifier's rollup node keeps running. The EL must come back with its chain,
// and the rollup node must reconnect and keep following the chain without a reset.
func TestVerifierResumesWithoutResetAfterELRestart(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnOpGeth(t, "the sysgo op-geth keeps its chain in memory, so a restart loses it")
	sys := presets.NewSingleChainMultiNode(t)
	require := t.Require()

	sys.L2CLB.Advanced(safety.LocalSafe, 1, 30)

	resetWatch := sys.L2CLB.WatchForReset()
	preRestartHead := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	sys.L2ELB.Stop()

	// Keep the EL down until the sequencer has produced new safe blocks, so the rollup node
	// must retry its engine calls and has new blocks to process once the EL is back. The EL
	// stays down until every check returns, so the verifier must stall for the whole window.
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.NotAdvancedFn(safety.LocalUnsafe, 3),
		sys.L2CLB.NotAdvancedFn(safety.LocalSafe, 3))

	sys.L2ELB.Start()
	require.Equal(preRestartHead, sys.L2ELB.BlockRefByNumber(preRestartHead.Number),
		"restarted EL must still have its chain from before the restart")

	seqUnsafe := sys.L2CL.HeadBlockRef(safety.LocalUnsafe)
	seqSafe := sys.L2CL.HeadBlockRef(safety.LocalSafe)
	dsl.CheckAll(t,
		sys.L2ELB.ReachedFn(eth.Unsafe, seqUnsafe.Number+1, 30),
		sys.L2ELB.ReachedFn(eth.Safe, seqSafe.Number+1, 30))
	resetWatch.VerifyNoReset()
}
