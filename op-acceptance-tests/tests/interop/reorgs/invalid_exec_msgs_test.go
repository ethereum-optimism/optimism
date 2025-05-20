package reorgs

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// TestReorgInvalidExecMsgs tests that the supervisor reorgs the chain when an invalid exec msg is included
// Each subtest runs a test with  a different invalid message, by modifying the message in the txModifierFn
func TestReorgInvalidExecMsgs(gt *testing.T) {
	gt.Run("invalid log index", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *suptypes.Message) {
			msg.Identifier.LogIndex = 1024
		})
	})

	gt.Run("invalid block number", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *suptypes.Message) {
			msg.Identifier.BlockNumber = msg.Identifier.BlockNumber - 1
		})
	})

	gt.Run("invalid chain id", func(gt *testing.T) {
		testReorgInvalidExecMsg(gt, func(msg *suptypes.Message) {
			msg.Identifier.ChainID = eth.ChainIDFromUInt64(1024)
		})
	})
}

func testReorgInvalidExecMsg(gt *testing.T, txModifierFn func(msg *suptypes.Message)) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSimpleInterop(t)
	l := sys.Log

	ia := sys.Sequencer.Escape().IndividualAPI(sys.L2ChainA.ChainID())

	// three EOAs for triggering the init and exec interop txs, as well as a simple transfer tx
	var alice, bob, cathrine *dsl.EOA
	{
		// alice is on chain A
		pk, err := crypto.GenerateKey()
		require.NoError(t, err)
		alice = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELA)
		sys.FaucetA.Fund(alice.Address(), eth.OneEther)

		// bob is on chain B
		pk, err = crypto.GenerateKey()
		require.NoError(t, err)
		bob = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELB)
		sys.FaucetB.Fund(bob.Address(), eth.OneEther)

		// cathrine is on chain A
		pk, err = crypto.GenerateKey()
		require.NoError(t, err)
		cathrine = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELA)
		sys.FaucetA.Fund(cathrine.Address(), eth.OneEther)

		l.Info("alice", "address", alice.Address())
		l.Info("bob", "address", bob.Address())
		l.Info("cathrine", "address", cathrine.Address())
	}

	sys.L1Network.WaitForBlock()
	sys.L2ChainA.WaitForBlock()

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
	sys.L2ChainB.CatchUpTo(sys.L2ChainA)

	var initTx *txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]
	var initReceipt *types.Receipt
	// prepare and include initiating message on chain B
	{
		initTx = txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](bob.Plan())
		initTx.Content.Set(initTrigger)
		var err error
		initReceipt, err = initTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)

		l.Info("initiating message included in chain B", "chain", sys.L2ChainB.ChainID(), "block_number", initReceipt.BlockNumber, "block_hash", initReceipt.BlockHash, "now", time.Now().Unix())
	}

	// at least one block between the init tx on chain B and the exec tx on chain A
	sys.L2ChainA.WaitForBlock()

	// stop sequencer on chain A so that we later force include an invalid exec msg
	latestUnsafe_A := sys.L2CLA.StopSequencer()

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
				Executor: constants.CrossL2Inbox,
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
		l.Info("Building chain A with op-test-sequencer, and include invalid exec msg", "chain", sys.L2ChainA.ChainID(), "unsafeHead", latestUnsafe_A)

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
		currentUnsafeRef := sys.L2ChainA.UnsafeHeadRef()

		l.Info("Unsafe head after invalid exec msg has been included in chain A", "chain", sys.L2ChainA.ChainID(), "unsafeHead", currentUnsafeRef, "parent", currentUnsafeRef.ParentID())

		divergenceBlockNumber_A = currentUnsafeRef.Number
		originalHash_A = currentUnsafeRef.Hash
		originalParentHash_A = currentUnsafeRef.ParentHash
		l.Info("Continue building chain A with another block with op-test-sequencer", "chain", sys.L2ChainA.ChainID(), "unsafeHead", currentUnsafeRef, "parent", currentUnsafeRef.ParentID())
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   currentUnsafeRef.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)

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
		time.Sleep(2 * time.Second)
	}

	// continue sequencing with op-node
	sys.L2CLA.StartSequencer()

	// start batcher on chain A
	sys.L2BatcherA.Start()

	// wait for reorg on chain A
	dsl.CheckAll(t,
		sys.L2ELA.ReorgTriggered(eth.L2BlockRef{
			Number:     divergenceBlockNumber_A,
			Hash:       originalHash_A,
			ParentHash: originalParentHash_A,
		}, 30),
	)

	err := wait.For(ctx, 5*time.Second, func() (bool, error) {
		safeL2Head_supervisor_A := sys.Supervisor.SafeBlockID(sys.L2ChainA.ChainID()).Hash
		safeL2Head_sequencer_A := sys.L2CLA.SafeL2BlockRef()

		if safeL2Head_sequencer_A.Number < divergenceBlockNumber_A {
			l.Info("Safe ref number is still behind divergence block A number", "divergence", divergenceBlockNumber_A, "safe", safeL2Head_sequencer_A.Number)
			return false, nil
		}

		if safeL2Head_sequencer_A.Hash.Cmp(safeL2Head_supervisor_A) != 0 {
			l.Info("Safe ref still not the same on supervisor and sequencer A", "supervisor", safeL2Head_supervisor_A, "sequencer", safeL2Head_sequencer_A.Hash)
			return false, nil
		}

		l.Info("Safe ref the same across supervisor and sequencers",
			"supervisor_A", safeL2Head_supervisor_A,
			"sequencer_A", safeL2Head_sequencer_A.Hash)

		return true, nil
	})
	require.NoError(t, err, "Expected to get same safe ref on both supervisor and sequencer eventually")

	sys.L2ChainA.PrintChain()
	sys.L2ChainB.PrintChain()
	spew.Dump(sys.Supervisor.FetchSyncStatus())
}
