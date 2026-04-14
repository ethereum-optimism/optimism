package safeheaddb_elsync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func newSingleChainMultiNodeELSync(t devtest.T) *presets.SingleChainMultiNode {
	// Use WithoutCheck because the default preset sync check uses 30 attempts
	// (60s) for CrossSafe matching, which is insufficient for EL Sync mode.
	// EL Sync must complete the initial sync phase before derivation can start,
	// so CrossSafe takes longer to advance than in CL Sync mode.
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.VerifierSyncMode = sync.ELSync
			cfg.SafeDBPath = p.TempDir()
		})),
	)
	// Run the initial sync check with 60 attempts (120s) to accommodate EL Sync.
	dsl.CheckAll(t,
		sys.L2CLB.MatchedFn(sys.L2CL, types.CrossSafe, 60),
		sys.L2CLB.MatchedFn(sys.L2CL, types.LocalUnsafe, 60),
	)
	return sys
}

func TestTruncateDatabaseOnELResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with op-reth:
	//
	// assertions.go:387:             ERROR[03-31|09:41:28.788]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/l2_el_p2p_util.go:61
	// assertions.go:387:             	            				/optimism/op-devstack/dsl/l2_el.go:281
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/safeheaddb_elsync/safeheaddb_test.go:44
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	dial tcp 127.0.0.1:55994: i/o timeout
	// assertions.go:387:             	Test:       	TestTruncateDatabaseOnELResync
	// assertions.go:387:             	Messages:   	The peer was not connected
	// assertions.go:387:
	sysgo.SkipOnOpReth(t, "not supported (peering issue)")
	sys := newSingleChainMultiNodeELSync(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	// Stop the verifier node. Since the sysgo EL uses in-memory storage this also wipes its database.
	// With the EL reset to genesis, when the CL restarts it will use EL sync to resync the chain rather than
	// deriving it from L1.
	sys.L2ELB.Stop()
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(types.LocalSafe, 3, 30)

	sys.L2ELB.Start()
	sys.L2CLB.Start()
	sys.L2ELB.PeerWith(sys.L2EL)

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 60)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)
}

func TestNotTruncateDatabaseOnRestartWithExistingDatabase(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|10:35:59.154]
	// assertions.go:387:             	Error Trace:	/Users/josh/repos/optimism/op-devstack/dsl/safedb.go:22
	// assertions.go:387:             	            				/Users/josh/repos/optimism/op-devstack/dsl/l2_cl.go:432
	// assertions.go:387:             	            				/Users/josh/repos/optimism/op-acceptance-tests/tests/safeheaddb_elsync/safeheaddb_test.go:74
	// assertions.go:387:             	Error:      	Expected value not to be nil.
	// assertions.go:387:             	Test:       	TestNotTruncateDatabaseOnRestartWithExistingDatabase
	// assertions.go:387:             	Messages:   	no safe head data available at L1 block 4
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainMultiNodeELSync(t)

	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))
	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 30)

	preRestartSafeBlock := sys.L2CLB.SafeL2BlockRef().Number
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))

	// Restart the verifier op-node, but not the EL so the existing chain data is not deleted.
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(types.LocalSafe, 3, 30)

	sys.L2CLB.Start()

	sys.L2CLB.Matched(sys.L2CL, types.LocalSafe, 60)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 60) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))
}
