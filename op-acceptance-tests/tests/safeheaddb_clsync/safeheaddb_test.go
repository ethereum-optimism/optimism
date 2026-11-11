package safeheaddb_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

func TestPreserveDatabaseOnCLResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleChainMultiNode(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.SequencerSyncMode = sync.CLSync
			cfg.VerifierSyncMode = sync.CLSync
			cfg.SafeDBPath = p.TempDir()
		})),
	)

	startSafeBlock := sys.L2CLB.SafeL2BlockRef().Number
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	// Stop the verifier node and wipe its EL database.
	// With the EL reset to genesis and the verifier in CL sync mode, the restarted CL resets to genesis and
	// derives the chain again from L1, re-recording safe heads that are already in its database.
	sys.L2ELB.Stop()
	sys.L2ELB.Wipe()
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(safety.LocalSafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.PeerWith(sys.L2EL)

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)
	sys.L2CLB.Advanced(safety.LocalSafe, 1, 30) // At least one safe head db update after resync

	// Safe head db should not have been reset
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(startSafeBlock))
}
