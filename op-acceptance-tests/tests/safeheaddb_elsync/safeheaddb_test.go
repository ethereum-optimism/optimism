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
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL)

	// Stop the verifier node. Since the sysgo EL uses in-memory storage this also wipes its database.
	// With the EL reset to genesis, when the CL restarts it will use EL sync to resync the chain rather than
	// deriving it from L1.
	sys.L2ELB.Stop()
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
		sys.L2CL.AdvancedFn(safety.LocalSafe, 1, 30),
		sys.L2CLB.AdvancedFn(safety.LocalSafe, 1, 30))
	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 30)

	preRestartSafeBlock := sys.L2CLB.SafeL2BlockRef().Number
	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))

	// Restart the verifier op-node, but not the EL so the existing chain data is not deleted.
	sys.L2CLB.Stop()

	sys.L2CL.Advanced(safety.LocalSafe, 3, 30)

	sys.L2CLB.Start()

	sys.L2CLB.InSync(sys.L2CL, safety.LocalSafe, 60)
	sys.L2CLB.Advanced(safety.LocalSafe, 1, 60) // At least one safe head db update after resync

	sys.L2CLB.VerifySafeHeadDatabaseMatches(sys.L2CL, dsl.WithMinRequiredL2Block(preRestartSafeBlock))
}
