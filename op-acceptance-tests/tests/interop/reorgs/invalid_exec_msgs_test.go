package reorgs

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txintent/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// TestReorgInvalidExecMsgs tests that the supernode reorgs the chain when an invalid exec msg is included
// Each subtest runs a test with  a different invalid message, by modifying the message in the txModifierFn
func TestReorgInvalidExecMsgs(gt *testing.T) {
	gt.Run("invalid log index", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *messages.Message) {
			msg.Identifier.LogIndex = 1024
		})
	})

	gt.Run("invalid block number", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *messages.Message) {
			msg.Identifier.BlockNumber = msg.Identifier.BlockNumber - 1
		})
	})

	gt.Run("invalid chain id", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *messages.Message) {
			msg.Identifier.ChainID = eth.ChainIDFromUInt64(1024)
		})
	})
}

func TestReorgInvalidExecMsgOpRethProofsHistoryTinyWindow(gt *testing.T) {
	testReorgInvalidExecMsgWithOptionsAndHooks(gt,
		[]presets.Option{
			presets.WithOpRethOption(sysgo.OpRethWithProofsHistoryWindow(1)),
		},
		func(msg *messages.Message) {
			msg.Identifier.LogIndex = 1024
		},
		func(t devtest.T, sys *presets.TwoL2SupernodeInterop, original eth.L2BlockRef) {
			sys.Log.Info("waiting for tiny proofs-history window to flush before CL-driven reorg",
				"number", original.Number,
				"hash", original.Hash)
			select {
			case <-t.Ctx().Done():
				t.Require().NoError(t.Ctx().Err())
			case <-time.After(12 * time.Second):
			}
		},
		func(t devtest.T, sys *presets.TwoL2SupernodeInterop, original eth.L2BlockRef) {
			ctx := t.Ctx()
			sys.Log.Info("waiting for op-reth to exit after proofs-history reorg unwind",
				"number", original.Number,
				"hash", original.Hash)
			require.Eventually(t, func() bool {
				_, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByLabel(ctx, eth.Unsafe)
				if err == nil {
					return false
				}
				sys.Log.Info("observed op-reth RPC failure after proofs-history unwind", "err", err)
				return true
			}, time.Minute, 200*time.Millisecond, "expected op-reth to exit during proofs-history reorg unwind")
			sys.Log.Info("restarting op-reth after proofs-history unwind exit",
				"number", original.Number,
				"hash", original.Hash)
			sys.L2ELA.Stop()
			if err := sys.L2ELA.StartWithTimeout(15 * time.Second); err != nil {
				require.Failf(t,
					"repro observed",
					"after exiting during proofs-history reorg unwind, op-reth did not become RPC-ready within 15s of restart: original=%s err=%v",
					original, err)
			}
			failAfterRestartedTinyProofWindowStall(t, sys, original, 30*time.Second)
		},
		false)
}

func failAfterRestartedTinyProofWindowStall(
	t devtest.T,
	sys *presets.TwoL2SupernodeInterop,
	original eth.L2BlockRef,
	duration time.Duration,
) {
	ctx := t.Ctx()
	l := sys.Log
	el := sys.L2ELA.Escape().L2EthClient()
	supernode := sys.SuperNodeClient()
	chainID := sys.L2A.ChainID()

	baseline, baselineErr := supernode.SyncStatus(ctx)
	start := time.Now()
	var lastEL any
	var lastSupernode any
	var lastLog time.Time

	for time.Since(start) < duration {
		unsafe, elErr := el.L2BlockRefByLabel(ctx, eth.Unsafe)
		status, statusErr := supernode.SyncStatus(ctx)
		chainStatus, chainStatusOK := status.Chains[chainID]

		if elErr != nil {
			lastEL = elErr
		} else {
			lastEL = unsafe
		}
		if statusErr != nil {
			lastSupernode = statusErr
		} else if !chainStatusOK {
			lastSupernode = fmt.Errorf("supernode_syncStatus missing chain %s", chainID)
		} else {
			lastSupernode = chainStatus
		}

		if elErr == nil && unsafe.Number > original.Number {
			require.Failf(t,
				"unexpected recovery",
				"chain A op-reth RPC recovered and EL unsafe advanced past reorg point: original=%s el_unsafe=%s supernode_status=%v",
				original, unsafe, supernodeChainStatusOrErr(status, chainID, statusErr))
		}
		if statusErr == nil && chainStatusOK &&
			(chainStatus.LocalSafeL2.Number > original.Number || chainStatus.SafeL2.Number > original.Number) {
			require.Failf(t,
				"unexpected supernode recovery",
				"supernode chain A advanced past reorg point: original=%s local_safe=%s cross_safe=%s unsafe=%s el_unsafe=%v",
				original, chainStatus.LocalSafeL2, chainStatus.SafeL2, chainStatus.UnsafeL2, blockRefOrErr(unsafe, elErr))
		}

		if lastLog.IsZero() || time.Since(lastLog) >= 5*time.Second {
			lastLog = time.Now()
			l.Info("observing tiny proofs-history repro non-recovery after op-reth restart",
				"elapsed", time.Since(start),
				"duration", duration,
				"original", original,
				"baseline_supernode_status", supernodeChainStatusOrErr(baseline, chainID, baselineErr),
				"el_unsafe", blockRefOrErr(unsafe, elErr),
				"supernode_status", supernodeChainStatusOrErr(status, chainID, statusErr))
		}

		select {
		case <-ctx.Done():
			require.NoError(t, ctx.Err())
		case <-time.After(time.Second):
		}
	}

	require.Failf(t,
		"repro observed",
		"after restarting op-reth from the proofs-history reorg-unwind exit, chain A EL did not advance and supernode did not advance chain A safe past the reorg point for %s: original=%s baseline_supernode_status=%v last_el_unsafe=%v last_supernode_status=%v",
		duration, original, supernodeChainStatusOrErr(baseline, chainID, baselineErr), lastEL, lastSupernode)
}

func blockRefOrErr(ref eth.L2BlockRef, err error) any {
	if err != nil {
		return err
	}
	return ref
}

func supernodeChainStatusOrErr(status eth.SuperNodeSyncStatusResponse, chainID eth.ChainID, err error) any {
	if err != nil {
		return err
	}
	chainStatus, ok := status.Chains[chainID]
	if !ok {
		return fmt.Errorf("supernode_syncStatus missing chain %s", chainID)
	}
	return chainStatus
}

func testReorgInvalidExecMsg(gt *testing.T, txModifierFn func(msg *messages.Message)) {
	testReorgInvalidExecMsgWithOptionsAndHooks(gt, nil, txModifierFn, nil, nil, true)
}

func testReorgInvalidExecMsgWithOptionsAndHooks(
	gt *testing.T,
	opts []presets.Option,
	txModifierFn func(msg *messages.Message),
	beforeSequencerRestart func(devtest.T, *presets.TwoL2SupernodeInterop, eth.L2BlockRef),
	afterReorgStarted func(devtest.T, *presets.TwoL2SupernodeInterop, eth.L2BlockRef),
	expectRecovery bool,
) {
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()

	sys := presets.NewTwoL2SupernodeInterop(t, 0, opts...)
	l := sys.Log

	ia := sys.TestSequencer.Escape().ControlAPI(sys.L2A.ChainID())

	// three EOAs for triggering the init and exec interop txs, as well as a simple transfer tx
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	cathrine := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)

	sys.L1Network.WaitForBlock()
	sys.L2A.WaitForBlock()

	// stop batcher on chain A
	sys.L2BatcherA.Stop()

	// deploy event logger on chain B
	var eventLoggerAddress common.Address
	{
		tx := txplan.NewPlannedTx(txplan.Combine(
			bob.Plan(),
			txplan.WithData(common.FromHex(bindings.EventloggerBin)),
		))
		res, err := tx.Included.Eval(ctx)
		require.NoError(t, err)

		eventLoggerAddress = res.ContractAddress
		l.Info("deployed EventLogger", "chainID", tx.ChainID.Value(), "address", eventLoggerAddress)
	}

	sys.L1Network.WaitForBlock()

	var initTrigger *txintent.InitTrigger
	// prepare init trigger (i.e. what logs to emit on chain A)
	{
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		nTopics := 3
		lenData := 10
		initTrigger = interop.RandomInitTrigger(rng, eventLoggerAddress, nTopics, lenData)

		l.Info("created init trigger", "address", eventLoggerAddress, "topics", nTopics, "lenData", lenData)
	}

	// wait for chain B to catch up to chain A if necessary
	sys.L2B.CatchUpTo(sys.L2A)

	var initTx *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	var initReceipt *types.Receipt
	// prepare and include initiating message on chain B
	{
		initTx = txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](bob.Plan())
		initTx.Content.Set(initTrigger)
		var err error
		initReceipt, err = initTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)

		l.Info("initiating message included in chain B", "chain", sys.L2B.ChainID(), "block_number", initReceipt.BlockNumber, "block_hash", initReceipt.BlockHash, "now", time.Now().Unix())
	}

	// at least one block between the init tx on chain B and the exec tx on chain A
	sys.L2A.WaitForBlock()

	// stop sequencer on chain A so that we later force include an invalid exec msg
	latestUnsafe_A := sys.L2ACL.StopSequencer()

	var execTx *txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput]
	var execSignedTx *types.Transaction
	var execTxEncoded []byte
	// prepare and include invalid executing message on chain B via the op-test-sequencer (no other way to force-include an invalid message)
	{
		execTx = txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](alice.Plan())
		execTx.Content.DependOn(&initTx.Result)
		// single event in tx so index is 0.
		index := 0
		// lambda to transform InteropOutput to a new broken ExecTrigger
		execTx.Content.Fn(func(ctx context.Context) (*txintent.ExecTrigger, error) {
			events := initTx.Result.Value()
			if x := len(events.Entries); x <= index {
				return nil, fmt.Errorf("invalid index: %d, only have %d events", index, x)
			}
			msg := events.Entries[index]
			// modify the message in order to make it invalid
			txModifierFn(&msg)
			return &txintent.ExecTrigger{
				Executor: predeploys.CrossL2InboxAddr,
				Msg:      msg,
			}, nil
		})

		var err error
		execSignedTx, err = execTx.PlannedTx.Signed.Eval(ctx)
		require.NoError(t, err)

		l.Info("executing message signed", "to", execSignedTx.To(), "nonce", execSignedTx.Nonce(), "data", len(execSignedTx.Data()))

		execTxEncoded, err = execSignedTx.MarshalBinary()
		require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")
	}

	// sequence a new block with an invalid executing msg on chain A
	{
		l.Info("Building chain A with op-test-sequencer, and include invalid exec msg", "chain", sys.L2A.ChainID(), "unsafeHead", latestUnsafe_A)

		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   latestUnsafe_A,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

		// include invalid executing msg in opened block
		err = ia.IncludeTx(ctx, execTxEncoded)
		require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
	}

	// record divergence block numbers and original refs for future validation checks
	var divergenceBlockNumber_A uint64
	var originalHash_A common.Hash
	var originalParentHash_A common.Hash
	// sequence a second block with op-test-sequencer
	{
		currentUnsafeRef := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

		l.Info("Unsafe head after invalid exec msg has been included in chain A", "chain", sys.L2A.ChainID(), "unsafeHead", currentUnsafeRef, "parent", currentUnsafeRef.ParentID())

		divergenceBlockNumber_A = currentUnsafeRef.Number
		originalHash_A = currentUnsafeRef.Hash
		originalParentHash_A = currentUnsafeRef.ParentHash
		l.Info("Continue building chain A with another block with op-test-sequencer", "chain", sys.L2A.ChainID(), "unsafeHead", currentUnsafeRef, "parent", currentUnsafeRef.ParentID())
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   currentUnsafeRef.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

		// include simple transfer tx in opened block
		{
			to := cathrine.PlanTransfer(alice.Address(), eth.OneGWei)
			opt := txplan.Combine(to)
			ptx := txplan.NewPlannedTx(opt)
			signed_tx, err := ptx.Signed.Eval(ctx)
			require.NoError(t, err, "Expected to be able to evaluate a planned transaction on op-test-sequencer, but got error")
			txdata, err := signed_tx.MarshalBinary()
			require.NoError(t, err, "Expected to be able to marshal a signed transaction on op-test-sequencer, but got error")

			err = ia.IncludeTx(ctx, txdata)
			require.NoError(t, err, "Expected to be able to include a signed transaction on op-test-sequencer, but got error")
		}

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		// Wait for the op-node to observe the block the test-sequencer just committed
		// before handing sequencing back to it via StartSequencer.
		expectedUnsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		waitCtx, waitCancel := context.WithTimeout(ctx, 30*time.Second)
		err = wait.For(waitCtx, 100*time.Millisecond, func() (bool, error) {
			return sys.L2ACL.SyncStatus().UnsafeL2.Hash == expectedUnsafe.Hash, nil
		})
		waitCancel()
		require.NoError(t, err, "op-node never observed test-sequencer's committed unsafe head %s", expectedUnsafe.Hash)
	}

	originalRef_A := eth.L2BlockRef{
		Number:     divergenceBlockNumber_A,
		Hash:       originalHash_A,
		ParentHash: originalParentHash_A,
	}
	if beforeSequencerRestart != nil {
		beforeSequencerRestart(t, sys, originalRef_A)
	}

	// continue sequencing with the supernode
	sys.L2ACL.StartSequencer()

	// start batcher on chain A
	sys.L2BatcherA.Start()

	if afterReorgStarted != nil {
		afterReorgStarted(t, sys, originalRef_A)
	}

	if !expectRecovery {
		return
	}

	// wait for reorg on chain A
	sys.L2ELA.ReorgExact(originalRef_A, 30)

	// Wait for the supernode to validate the replacement block's timestamp.
	divergenceTimestamp_A := sys.L2A.TimestampForBlockNum(divergenceBlockNumber_A)
	sys.Supernode.AwaitValidatedTimestamp(divergenceTimestamp_A)

	l.Info("supernode validated replacement block",
		"divergence", divergenceBlockNumber_A,
		"timestamp", divergenceTimestamp_A)
}
