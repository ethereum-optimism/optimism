package reorgs

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestReorgInitExecMsg(gt *testing.T) {
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
		sys.FaucetA.Fund(alice.Address(), eth.ThousandEther)

		// bob is on chain B
		pk, err = crypto.GenerateKey()
		require.NoError(t, err)
		bob = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELB)
		sys.FaucetB.Fund(bob.Address(), eth.ThousandEther)

		// cathrine is on chain A
		pk, err = crypto.GenerateKey()
		require.NoError(t, err)
		cathrine = dsl.NewEOA(dsl.NewKey(t, pk), sys.L2ELA)
		sys.FaucetA.Fund(cathrine.Address(), eth.ThousandEther)

		l.Info("alice", "address", alice.Address())
		l.Info("bob", "address", bob.Address())
		l.Info("cathrine", "address", cathrine.Address())
	}

	sys.L1Network.WaitForBlock()
	sys.L2ChainA.WaitForBlock()

	// stop batchers on chain A and on chain B
	{
		err := retry.Do0(ctx, 3, retry.Exponential(), func() error {
			err := sys.L2BatcherA.Escape().ActivityAPI().StopBatcher(ctx)
			if err != nil && strings.Contains(err.Error(), "batcher is not running") {
				return nil
			}
			return err
		})
		require.NoError(t, err, "Expected to be able to call StopBatcher API on chain A, but got error")

		err = retry.Do0(ctx, 3, retry.Exponential(), func() error {
			err := sys.L2BatcherB.Escape().ActivityAPI().StopBatcher(ctx)
			if err != nil && strings.Contains(err.Error(), "batcher is not running") {
				return nil
			}
			return err
		})
		require.NoError(t, err, "Expected to be able to call StopBatcher API on chain B, but got error")
	}

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
	{
		unsafeHead, err := sys.L2CLA.Escape().RollupAPI().StopSequencer(ctx)
		require.NoError(t, err, "expected to be able to call StopSequencer API, but got error")

		// wait for the sequencer to become inactive
		var active bool
		err = wait.For(ctx, 1*time.Second, func() (bool, error) {
			active, err = sys.L2CLA.Escape().RollupAPI().SequencerActive(ctx)
			return !active, err
		})
		require.NoError(t, err, "expected to be able to call SequencerActive API, and wait for inactive state for sequencer, but got error")

		l.Info("rollup node sequencer status", "chain", sys.L2ChainA.ChainID(), "active", active, "unsafeHead", unsafeHead)
	}

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
			to := cathrine.PlanTransfer(alice.Address(), eth.OneEther)
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
		currentUnsafeRef := sys.L2ChainA.UnsafeHeadRef()
		l.Info("Current unsafe ref", "unsafeHead", currentUnsafeRef)
		err := ia.New(ctx, seqtypes.BuildOpts{
			Parent:   currentUnsafeRef.Hash,
			L1Origin: nil,
		})
		require.NoError(t, err, "Expected to be able to create a new block job for sequencing on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)

		err = ia.Next(ctx)
		require.NoError(t, err, "Expected to be able to call Next() after New() on op-test-sequencer, but got error")
		time.Sleep(2 * time.Second)
	}

	// continue sequencing with op-node
	{
		newUnsafeHeadRef := sys.L2ChainA.UnsafeHeadRef()
		l.Info("Continue sequencing with consensus node (op-node)", "unsafeHead", newUnsafeHeadRef)

		err := sys.L2CLA.Escape().RollupAPI().StartSequencer(ctx, newUnsafeHeadRef.Hash)
		require.NoError(t, err, "Expected to be able to start sequencer on rollup node")

		// wait for the sequencer to become active
		var active bool
		err = wait.For(ctx, 1*time.Second, func() (bool, error) {
			active, err = sys.L2CLA.Escape().RollupAPI().SequencerActive(ctx)
			return active, err
		})
		require.NoError(t, err, "Expected to be able to call SequencerActive API, and wait for an active state for sequencer, but got error")

		l.Info("Rollup node sequencer", "active", active)
	}

	// start batchers on chain A and on chain B
	{
		err := retry.Do0(ctx, 3, retry.Exponential(), func() error {
			return sys.L2BatcherA.Escape().ActivityAPI().StartBatcher(ctx)
		})
		require.NoError(t, err, "Expected to be able to call StartBatcher API on chain A, but got error")

		err = retry.Do0(ctx, 3, retry.Exponential(), func() error {
			return sys.L2BatcherB.Escape().ActivityAPI().StartBatcher(ctx)
		})
		require.NoError(t, err, "Expected to be able to call StartBatcher API on chain B, but got error")
	}

	// confirm reorg on chain A
	{
		reorgedRef_A, err := sys.L2ELA.Escape().EthClient().BlockRefByNumber(ctx, divergenceBlockNumber_A)
		require.NoError(t, err, "Expected to be able to call BlockRefByNumber API, but got error")

		l.Info("Reorged chain A on divergence block number (prior the reorg)", "chain", sys.L2ChainA.ChainID(), "number", divergenceBlockNumber_A, "head", originalRef_A.Hash, "parent", originalRef_A.ParentID().Hash)
		l.Info("Reorged chain A on divergence block number (after the reorg)", "chain", sys.L2ChainA.ChainID(), "number", divergenceBlockNumber_A, "head", reorgedRef_A.Hash, "parent", reorgedRef_A.ParentID().Hash)
		require.NotEqual(t, originalRef_A.Hash, reorgedRef_A.Hash, "Expected to get different heads on divergence block A number, but got the same hash, so no reorg happened")
		require.Equal(t, originalRef_A.ParentID().Hash, reorgedRef_A.ParentHash, "Expected to get same parent hashes on divergence block A number, but got different hashes")
	}

	// wait for reorg on chain B
	require.Eventually(t, func() bool {
		reorgedRef_B, err := sys.L2ELB.Escape().EthClient().BlockRefByNumber(ctx, divergenceBlockNumber_B)
		if err != nil {
			if strings.Contains(err.Error(), "not found") { // reorg is happening wait a bit longer
				l.Info("Supervisor still hasn't reorged chain B", "error", err)
				return false
			}
			require.NoError(t, err, "Expected to be able to call BlockRefByNumber API, but got error")
		}

		if originalRef_B.Hash.Cmp(reorgedRef_B.Hash) == 0 { // want not equal
			l.Info("Supervisor still hasn't reorged chain B", "ref", originalRef_B)
			return false
		}

		l.Info("Reorged chain B on divergence block number (prior the reorg)", "chain", sys.L2ChainB.ChainID(), "number", divergenceBlockNumber_B, "head", originalRef_B.Hash, "parent", originalRef_B.ParentID().Hash)
		l.Info("Reorged chain B on divergence block number (after the reorg)", "chain", sys.L2ChainB.ChainID(), "number", divergenceBlockNumber_B, "head", reorgedRef_B.Hash, "parent", reorgedRef_B.ParentID().Hash)
		return true
	}, 180*time.Second, 10*time.Second, "No reorg happened on chain B. Should have been triggered by the supervisor.")

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

	sys.L2ChainA.PrintChain()
	sys.L2ChainB.PrintChain()
	spew.Dump(sys.Supervisor.FetchSyncStatus())
}
