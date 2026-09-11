package follow_l2

import (
	"context"
	"fmt"
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
)

func TestFollowL2_Safe_Finalized_CurrentL1(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:33:11.255]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:18
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_Safe_Finalized_CurrentL1
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL node
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainTwoVerifiersFollowL2(t)
	logger := t.Logger()

	// Takes about 2 minutes for L1 finalization
	attempts := 70
	target := uint64(3)

	// L2CL is the sequencer with CL follow source, derivation disabled
	// L2CLB is the verifier without follow source, derivation enabled
	// L2CLC is the verifier with CL follow source, derivation disabled
	// All verifiers must eventually advance unsafe, safe, finalized
	checkMatchedAll := func(lvl safety.Level) {
		dsl.CheckAll(t,
			sys.L2CL.ReachedFn(lvl, target, attempts),
			sys.L2CLB.ReachedFn(lvl, target, attempts),
			sys.L2CLC.ReachedFn(lvl, target, attempts),
		)
		dsl.CheckAll(t,
			sys.L2CLB.InSyncFn(sys.L2CL, lvl, attempts),
			sys.L2CLB.InSyncFn(sys.L2CLC, lvl, attempts),
		)
	}

	checkMatchedAll(safety.LocalUnsafe)
	logger.Info("Unsafe head advanced due to CLP2P", "target", target)

	checkMatchedAll(safety.LocalSafe)
	logger.Info("Safe head followed source", "target", target)

	checkMatchedAll(safety.Finalized)
	logger.Info("Finalized head followed source", "target", target)

	attempts = 10
	dsl.CheckAll(t,
		sys.L2CLC.CurrentL1MatchedFn(sys.L2CLB, attempts),
		sys.L2CL.CurrentL1MatchedFn(sys.L2CLB, attempts),
	)
	logger.Info("CurrentL1 followed source", "currentL1", sys.L2CL.SyncStatus().CurrentL1, "currentL1C", sys.L2CLC.SyncStatus().CurrentL1)
}

func TestFollowL2_ReorgRecovery(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:31:11.567]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:60
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_ReorgRecovery
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL node
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainTwoVerifiersFollowL2(t, presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
		cfg.Stopped = true
	}))
	require := t.Require()
	logger := t.Logger()
	ctx := t.Ctx()

	// L2CLB is the verifier without follow source, derivation enabled

	// Keep L2 sequencing stopped while establishing a bounded L1 fork. Build
	// the reorg target and its child through the test sequencer, then let the
	// regular L1 producer add one block. That last block supplies the normal
	// L1-head signal that makes the target eligible after two confirmations.
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())
	sys.L2CL.StopSequencer()
	// Pass the L1 genesis
	sys.L1Network.WaitForBlock()

	// Stop auto advancing L1
	sys.L1CL.Stop()
	require.NoError(ts.Next(ctx))
	l1BlockBeforeReorg := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	require.NoError(ts.Next(ctx))
	secondNewL1Block := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	sys.L1CL.Start()
	sys.L1EL.WaitForBlockNumber(secondNewL1Block.Number + 1)
	sys.L1CL.Stop()
	l1Head := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Sequence against the bounded fork, then freeze L2 after it adopts the
	// first new L1 origin. Starting the batcher only now guarantees that its
	// first pending transaction contains this range.
	sys.L2CL.StartSequencer()
	sys.L2EL.WaitL1OriginHash(eth.Unsafe, l1BlockBeforeReorg.ID(), 30)
	sys.L2CL.StopSequencer()
	sys.L2Batcher.Start()
	batchInbox := sys.L2Chain.Escape().RollupConfig().BatchInboxAddress
	// L1 remains stopped, so this transaction cannot be included until the
	// explicit test-sequencer block below.
	require.Eventually(func() bool {
		lookupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		var content struct {
			Pending map[string]map[string]struct {
				To *common.Address `json:"to"`
			} `json:"pending"`
		}
		if err := sys.L1EL.EthClient().RPC().CallContext(lookupCtx, &content, "txpool_content"); err != nil {
			logger.Info("Waiting for pending batch transaction", "error", err)
			return false
		}
		for _, accountTxs := range content.Pending {
			for _, tx := range accountTxs {
				if tx.To != nil && *tx.To == batchInbox {
					return true
				}
			}
		}
		return false
	}, 60*time.Second, 200*time.Millisecond)

	// Include that transaction with one explicit build job, then leave L1
	// frozen while the verifier derives the target safe block.
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1Head.Hash}))
	require.NoError(ts.Next(ctx))
	inclusionBlock := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	require.Equal(l1Head.Number+1, inclusionBlock.Number)
	require.Equal(l1Head.Hash, inclusionBlock.ParentHash)
	sys.L2ELB.WaitL1OriginHash(eth.Safe, l1BlockBeforeReorg.ID(), 60)

	l2BlockBeforeReorg := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.Equal(l1BlockBeforeReorg.ID(), l2BlockBeforeReorg.L1Origin)
	logger.Info("Target L2 Block to reorg", "l2", l2BlockBeforeReorg, "l1_origin", l2BlockBeforeReorg.L1Origin)

	// Make sure verifier safe head is also advanced from reorgL2Block or matched
	sys.L2ELB.Reached(eth.Safe, l2BlockBeforeReorg.Number, 3)

	// Reorg L1 block which safe block L1 Origin points to
	require.Less(sys.L1EL.BlockRefByLabel(eth.Safe).Number, l1BlockBeforeReorg.Number, "reorg target must be above the L1 safe head")
	logger.Info("Triggering L1 reorg", "l1", l1BlockBeforeReorg)
	require.NoError(ts.New(ctx, seqtypes.BuildOpts{Parent: l1BlockBeforeReorg.ParentHash}))
	require.NoError(ts.Next(ctx))

	// Start advancing L1 and L2
	sys.L1CL.Start()
	sys.L2CL.StartSequencer()

	// Make sure L1 reorged
	sys.L1EL.WaitForBlockNumber(l1BlockBeforeReorg.Number)
	l1BlockAfterReorg := sys.L1EL.BlockRefByNumber(l1BlockBeforeReorg.Number)
	logger.Info("Triggered L1 reorg", "l1", l1BlockAfterReorg)
	require.NotEqual(l1BlockAfterReorg.Hash, l1BlockBeforeReorg.Hash)

	// Need to poll until the L2CL detects L1 Reorg and trigger L2 Reorg
	// What happens:
	//  L2CL detects L1 Reorg and reset the pipeline. op-node example logs: "reset: detected L1 reorg"
	//  L2ELB detects L2 reorg and replaces the original block. The replacement
	//  block at this height may also come from a different parent chain, so only
	//  assert that the original block is replaced before checking convergence.
	var l2BlockAfterReorg eth.L2BlockRef
	require.Eventually(func() bool {
		l2BlockAfterReorg = sys.L2ELB.BlockRefByNumber(l2BlockBeforeReorg.Number)
		if l2BlockAfterReorg.Hash == l2BlockBeforeReorg.Hash {
			logger.Info("Waiting for L2 reorg", "before", l2BlockBeforeReorg, "current", l2BlockAfterReorg)
			return false
		}
		return true
	}, 60*time.Second, 2*time.Second)
	logger.Info("Triggered L2 reorg", "l2", l2BlockAfterReorg)

	attempts := 30
	dsl.CheckAll(t,
		sys.L2CL.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
		sys.L2CL.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
	)
}

func TestFollowL2_WithoutCLP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	// Example error with kona-node:
	//
	// assertions.go:387:             ERROR[03-31|11:27:57.797]
	// assertions.go:387:             	Error Trace:	/optimism/op-devstack/sysgo/singlechain_variants.go:143
	// assertions.go:387:             	            				/optimism/op-devstack/sysgo/singlechain_variants.go:53
	// assertions.go:387:             	            				/optimism/op-devstack/presets/singlechain_twoverifiers.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/setup_test.go:24
	// assertions.go:387:             	            				/optimism/op-acceptance-tests/tests/sync/follow_l2/sync_test.go:136
	// assertions.go:387:             	Error:      	Should be true
	// assertions.go:387:             	Test:       	TestFollowL2_WithoutCLP2P
	// assertions.go:387:             	Messages:   	single-chain test sequencer requires an op-node CL nod
	sysgo.SkipOnKonaNode(t, "not supported")
	sys := newSingleChainTwoVerifiersFollowL2(t)
	require := t.Require()
	logger := t.Logger()

	attempts := 20
	target := uint64(3)

	// L2CLB is the verifier without follow source, derivation enabled
	sys.L2CLB.Advanced(safety.LocalUnsafe, target, attempts)

	// The test's primary target is the L2CLC, with follow source and derivation disabled
	// There is often a gap between safe and unsafe before disconnect, but the
	// follow-source verifier may also catch up before we observe it. The actual
	// property this test cares about is the post-disconnect behavior below.
	status := sys.L2CLC.SyncStatus()
	logger.Info("Initial follow-source sync status", "safe", status.LocalSafeL2, "unsafe", status.UnsafeL2)

	logger.Info("Disconnect CLP2P")
	// L2CLC is the verifier with follow source, derivation disabled
	// Disconnect CLP2P of verifier which follow source is enabled
	sys.L2CLC.DisconnectPeer(sys.L2CLB)
	sys.L2CLB.DisconnectPeer(sys.L2CLC)
	sys.L2CLC.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLC)

	// Advance few safe blocks
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)
	sys.L2CLC.ReachedRef(safety.LocalSafe, sys.L2CLB.HeadBlockRef(safety.LocalSafe).ID(), attempts)

	// Wait for L2CLC's local-safe to catch up to its now-frozen unsafe head.
	// The initial gap can exceed a fixed 40s budget on loaded CI runners
	// (#20718), so bound the wait by stall detection on local-safe progress.
	require.NoError(sys.L2CLC.MatchedWithProgressFn(
		unsafeAsLocalSafe{cl: sys.L2CLC},
		safety.LocalSafe, safety.LocalSafe,
		5*time.Minute, 30*time.Second,
	)())
	// The only data source for L2CLC is the follow source.
	// L2CLC unsafe head will only be advancing with safe head together
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Advance few safe blocks
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Check once again that the unsafe head is moving together with safe head
	status = sys.L2CLC.SyncStatus()
	require.Equal(status.LocalSafeL2, status.UnsafeL2)
	sys.L2CLC.Advanced(safety.LocalSafe, target, attempts)

	// Recover CLP2P
	logger.Info("Recover CLP2P")
	sys.L2CLC.ConnectPeer(sys.L2CLB)
	sys.L2CLB.ConnectPeer(sys.L2CLC)
	sys.L2CLC.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLC)

	// Sequencer unsafe payload will arrive to the verifier, triggering EL sync and filling in the unsafe gap
	dsl.CheckAll(t,
		// In sync with sequencer, with derivation disabled
		sys.L2CLC.InSyncFn(sys.L2CL, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CL, safety.LocalUnsafe, attempts),
		// In sync with other verifier, with derivation enabled
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalSafe, attempts),
		sys.L2CLC.InSyncFn(sys.L2CLB, safety.LocalUnsafe, attempts),
	)

	t.Cleanup(func() {
		sys.L2CLC.ConnectPeer(sys.L2CLB)
		sys.L2CLB.ConnectPeer(sys.L2CLC)
		sys.L2CLC.ConnectPeer(sys.L2CL)
		sys.L2CL.ConnectPeer(sys.L2CLC)
	})
}

// unsafeAsLocalSafe lets MatchedWithProgressFn compare cl.LocalSafe to
// cl.UnsafeL2: it takes two providers at one level, so we synthesize a
// reference whose LocalSafe reports cl's unsafe head.
type unsafeAsLocalSafe struct{ cl *dsl.L2CLNode }

func (u unsafeAsLocalSafe) ChainSyncStatus(chainID eth.ChainID, lvl safety.Level) eth.BlockID {
	if lvl != safety.LocalSafe {
		panic(fmt.Sprintf("unsafeAsLocalSafe queried at %s, expected LocalSafe", lvl))
	}
	return u.cl.ChainSyncStatus(chainID, safety.LocalUnsafe)
}

func (u unsafeAsLocalSafe) String() string {
	return u.cl.String() + "/unsafe-as-local-safe"
}
