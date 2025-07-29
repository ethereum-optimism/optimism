package reorgs

import (
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestReorgInitExecMsg(gt *testing.T) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSimpleInterop(t)
	l := sys.Log

	ia := sys.TestSequencer.Escape().ControlAPI(sys.L2ChainA.ChainID())

	// three EOAs for triggering the init and exec interop txs, as well as a simple transfer tx
	alice := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)
	bob := sys.FunderB.NewFundedEOA(eth.OneHundredthEther)
	cathrine := sys.FunderA.NewFundedEOA(eth.OneHundredthEther)

	sys.L1Network.WaitForBlock()
	sys.L2ChainA.WaitForBlock()

	// stop batchers on chain A and on chain B
	sys.L2BatcherA.Stop()
	sys.L2BatcherB.Stop()

	// deploy event logger on chain A
	var eventLoggerAddress common.Address
	{
		tx := txplan.NewPlannedTx(txplan.Combine(
			alice.Plan(),
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
	// prepare and include initiating message on chain A
	{
		initTx = txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](alice.Plan())
		initTx.Content.Set(initTrigger)
		var err error
		initReceipt, err = initTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)

		l.Info("initiating message included", "chain", sys.L2ChainA.ChainID(), "block_number", initReceipt.BlockNumber, "block_hash", initReceipt.BlockHash, "now", time.Now().Unix())
	}

	// stop sequencer on chain A so that we later force a reorg/removal of the init msg
	sys.L2CLA.StopSequencer()

	// at least one block between the init tx on chain A and the exec tx on chain B
	sys.L2ChainB.WaitForBlock()

	var execTx *txintent.IntentTx[*txintent.ExecTrigger, *txintent.InteropOutput]
	var execReceipt *types.Receipt
	// prepare and include executing message on chain B
	{
		execTx = txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](bob.Plan())
		execTx.Content.DependOn(&initTx.Result)
		// single event in tx so index is 0. ExecuteIndexed returns a lambda to transform InteropOutput to a new ExecTrigger
		execTx.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &initTx.Result, 0))
		var err error
		execReceipt, err = execTx.PlannedTx.Included.Eval(ctx)
		require.NoError(t, err)
		require.Equal(t, 1, len(execReceipt.Logs))

		l.Info("executing message included", "chain", sys.L2ChainB.ChainID(), "block_number", execReceipt.BlockNumber, "block_hash", execReceipt.BlockHash, "now", time.Now().Unix())
	}

	// record divergence block numbers and original refs for future validation checks
	var divergenceBlockNumber_A, divergenceBlockNumber_B uint64
	var originalRef_A, originalRef_B eth.L2BlockRef

	// sequence a conflicting block with a simple transfer tx, based on the parent of the parent of the unsafe head
	{
		var err error
		divergenceBlockNumber_B = execReceipt.BlockNumber.Uint64()
		originalRef_B, err = sys.L2ELB.Escape().L2EthClient().L2BlockRefByHash(ctx, execReceipt.BlockHash)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		headToReorgA := initReceipt.BlockHash
		headToReorgARef, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByHash(ctx, headToReorgA)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		divergenceBlockNumber_A = headToReorgARef.Number
		originalRef_A = headToReorgARef

		parentOfHeadToReorgA := headToReorgARef.ParentID()
		parentsL1Origin, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByHash(ctx, parentOfHeadToReorgA.Hash)
		require.NoError(t, err, "Expected to be able to call L2BlockRefByHash API, but got error")

		nextL1Origin := parentsL1Origin.L1Origin.Number + 1
		l1Origin, err := sys.L1Network.Escape().L1ELNode(match.FirstL1EL).EthClient().InfoByNumber(ctx, nextL1Origin)
		require.NoError(t, err, "Expected to get block number %v from L1 execution client", nextL1Origin)
		l1OriginHash := l1Origin.Hash()

		l.Info("Sequencing a conflicting block", "chain", sys.L2ChainA.ChainID(), "newL1Origin", eth.ToBlockID(l1Origin), "headToReorgA", headToReorgARef, "parent", parentOfHeadToReorgA, "parent_l1_origin", parentsL1Origin.L1Origin)

		err = ia.New(ctx, seqtypes.BuildOpts{
			Parent:   parentOfHeadToReorgA.Hash,
			L1Origin: &l1OriginHash,
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
	}

	// sequence a second block with op-test-sequencer
	{
		unsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		l.Info("Current unsafe ref", "unsafeHead", unsafe)
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   unsafe.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
	}

	// continue sequencing with op-node
	sys.L2CLA.StartSequencer()

	// start batchers on chain A and on chain B
	sys.L2BatcherA.Start()
	sys.L2BatcherB.Start()

	// wait and confirm reorgs on chain A and B
	dsl.CheckAll(t,
		sys.L2ELA.ReorgTriggeredFn(eth.L2BlockRef{
			Number:     divergenceBlockNumber_A,
			Hash:       originalRef_A.Hash,
			ParentHash: originalRef_A.ParentID().Hash,
		}, 30),
		sys.L2ELB.ReorgTriggeredFn(eth.L2BlockRef{
			Number:     divergenceBlockNumber_B,
			Hash:       originalRef_B.Hash,
			ParentHash: originalRef_B.ParentID().Hash,
		}, 30),
	)

	// executing tx should eventually be no longer confirmed on chain B
	require.Eventually(t, func() bool {
		receipt, err := sys.L2ELB.Escape().EthClient().TransactionReceipt(ctx, execReceipt.TxHash)
		if err == nil || err.Error() != "not found" { // want to get "not found" error
			return false
		}
		if receipt != nil { // want to get nil receipt
			return false
		}
		return true
	}, 60*time.Second, 3*time.Second, "Expected for the executing tx to be removed from chain B")

	err := wait.For(ctx, 5*time.Second, func() (bool, error) {
		safeL2Head_supervisor_A := sys.Supervisor.SafeBlockID(sys.L2ChainA.ChainID()).Hash
		safeL2Head_supervisor_B := sys.Supervisor.SafeBlockID(sys.L2ChainB.ChainID()).Hash
		safeL2Head_sequencer_A := sys.L2CLA.SafeL2BlockRef()
		safeL2Head_sequencer_B := sys.L2CLB.SafeL2BlockRef()

		if safeL2Head_sequencer_A.Number < divergenceBlockNumber_A {
			l.Info("Safe ref number is still behind divergence block A number", "divergence", divergenceBlockNumber_A, "safe", safeL2Head_sequencer_A.Number)
			return false, nil
		}

		if safeL2Head_sequencer_B.Number < divergenceBlockNumber_B {
			l.Info("Safe ref number is still behind divergence block B number", "divergence", divergenceBlockNumber_B, "safe", safeL2Head_sequencer_B.Number)
			return false, nil
		}

		if safeL2Head_sequencer_A.Hash.Cmp(safeL2Head_supervisor_A) != 0 {
			l.Info("Safe ref still not the same on supervisor and sequencer A", "supervisor", safeL2Head_supervisor_A, "sequencer", safeL2Head_sequencer_A.Hash)
			return false, nil
		}

		if safeL2Head_sequencer_B.Hash.Cmp(safeL2Head_supervisor_B) != 0 {
			l.Info("Safe ref still not the same on supervisor and sequencer B", "supervisor", safeL2Head_supervisor_B, "sequencer", safeL2Head_sequencer_B.Hash)
			return false, nil
		}

		l.Info("Safe ref the same across supervisor and sequencers",
			"supervisor_A", safeL2Head_supervisor_A,
			"sequencer_A", safeL2Head_sequencer_A.Hash,
			"supervisor_B", safeL2Head_supervisor_B,
			"sequencer_B", safeL2Head_sequencer_B.Hash)

		return true, nil
	})
	require.NoError(t, err, "Expected to get same safe ref on both supervisor and sequencer eventually")
}
