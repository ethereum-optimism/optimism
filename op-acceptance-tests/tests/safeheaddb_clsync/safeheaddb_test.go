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

func TestPreserveDatabaseOnCLResync(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with op-reth:
	//
	// assertions.go:387:             ERROR[03-31|09:40:50.281]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/l2_el_p2p_util.go:61
	// assertions.go:387:             	            				/optimism/op-devstack/dsl/l2_el.go:281
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/safeheaddb_clsync/safeheaddb_test.go:42
	// assertions.go:387:             	Error:      	Received unexpected error:
	// assertions.go:387:             	            	context deadline exceeded
	// assertions.go:387:             	Test:       	TestPreserveDatabaseOnCLResync
	// assertions.go:387:             	Messages:   	The peer was not connected
	// assertions.go:387:
	sysgo.SkipOnOpReth(t, "not supported (peering issue)")
	sys := presets.NewSingleChainMultiNode(t,
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(p devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.SequencerSyncMode = sync.CLSync
			cfg.VerifierSyncMode = sync.CLSync
			cfg.SafeDBPath = p.TempDir()
		})),
	)

	startSafeBlock := sys.L2CLB.SafeL2BlockRef().Number
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(types.LocalSafe, 1, 30))

	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 30)
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

	sys.L2CLB.InSync(sys.L2CL, types.LocalSafe, 30)
	sys.L2CLB.Advanced(types.LocalSafe, 1, 30) // At least one safe head db update after resync

	// Safe head db should not have been reset
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(startSafeBlock))
}
