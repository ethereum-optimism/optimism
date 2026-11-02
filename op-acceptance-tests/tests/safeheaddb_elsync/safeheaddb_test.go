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

func newSingleChainMultiNodeELSync(t devtest.T) *presets.SingleChainMultiNode {
	// The default preset sync check auto-detects ELSync mode on node B and
	// uses a 4x CrossSafe budget (see singlechain_from_runtime.go), which is
	// what this test needs.
	return presets.NewSingleChainMultiNode(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.VerifierSyncMode = sync.ELSync
			cfg.SafeDBPath = p.TempDir()
		})),
	)
}

func TestTruncateDatabaseOnELResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSingleChainMultiNodeELSync(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	// Stop the verifier node and wipe its EL database.
	// With the EL reset to genesis, when the CL restarts it will use EL sync to resync the chain rather than
	// deriving it from L1.
	sys.L2ELB.Stop()
	sys.L2ELB.Wipe()
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(safety.LocalSafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	// Wait for the restarted EL to be ready to accept RPC/P2P connections
	// before attempting to peer. Without this, PeerWith may waste its 30s
	// timeout on a node whose P2P stack isn't fully initialized yet.
	sys.L2ELB.WaitForOnline()
	sys.L2ELB.PeerWith(sys.L2EL)

	// EL Sync after a full database wipe requires more time than the initial sync:
	// the EL must re-download all blocks via P2P before the CL can begin derivation,
	// and node A keeps advancing LocalSafe in the meantime.
	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 90)
	sys.L2CLB.Advanced(safety.LocalSafe, 1, 90) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)
}

func TestNotTruncateDatabaseOnRestartWithExistingDatabase(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSingleChainMultiNodeELSync(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))
	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)

	// Baseline from the db's own floor, not the live safe head, so the check only fails on real truncation.
	preRestartSafeBlock := sys.L2CLB.SafeHeadDBFloor()
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))

	// Restart the verifier op-node, but not the EL so the existing chain data is not deleted.
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(safety.LocalSafe, 3, 30)

	sys.L2CLB.Start()

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 60)
	sys.L2CLB.Advanced(safety.LocalSafe, 1, 60) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))
}
