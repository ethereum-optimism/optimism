package reorg

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// TestSupernodeRejectsExecutionBeforeInitiation proves that an EVM-valid
// executing message cannot become cross-safe before its declared initiation
// timestamp. The executing block is replaced with deposits only, the later
// initiating block stays canonical, and sequencing continues on the replacement.
func TestSupernodeRejectsExecutionBeforeInitiation(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	setup := presets.NewTwoL2SupernodeInterop(t, 0).ForSameTimestampTesting(t)

	blockTime := setup.L2A.Escape().RollupConfig().BlockTime
	initTimestamp := setup.NextTimestamp + blockTime
	pair := setup.Alice.PrepareSameTimestampInit(
		rand.New(rand.NewSource(902_901)),
		setup.EventLoggerA,
		setup.ExpectedBlockNumA+1,
		0,
		initTimestamp,
	)

	execTx := pair.SubmitExecTo(setup.Bob)
	txplan.WithStaticNonce(setup.Bob.PendingNonce())(execTx)
	parentB := setup.L2ELB.BlockRefByLabel(eth.Unsafe)
	execHash := setup.TestSequencer.SequencePlannedBlock(
		t, setup.L2B.ChainID(), parentB.Hash, execTx,
	)[0]
	execReceipt := setup.L2ELB.WaitForReceipt(execHash)
	require.Equal(types.ReceiptStatusSuccessful, execReceipt.Status,
		"ordering fault must be an interop conflict, not an EVM failure")
	invalidRef := setup.L2ELB.BlockRefByHash(execReceipt.BlockHash)
	require.Equal(setup.NextTimestamp, invalidRef.Time)
	require.Less(invalidRef.Time, initTimestamp)

	parentA := setup.L2ELA.BlockRefByLabel(eth.Unsafe)
	setup.TestSequencer.SequenceBlock(t, setup.L2A.ChainID(), parentA.Hash)
	preInitRef := setup.L2ELA.BlockRefByNumber(setup.ExpectedBlockNumA)
	require.Equal(setup.NextTimestamp, preInitRef.Time)

	initTx := pair.SubmitInit()
	txplan.WithStaticNonce(setup.Alice.PendingNonce())(initTx)
	initHash := setup.TestSequencer.SequencePlannedBlock(
		t, setup.L2A.ChainID(), preInitRef.Hash, initTx,
	)[0]
	initReceipt := setup.L2ELA.WaitForReceipt(initHash)
	require.Equal(types.ReceiptStatusSuccessful, initReceipt.Status)
	initRef := setup.L2ELA.BlockRefByHash(initReceipt.BlockHash)
	require.Equal(initTimestamp, initRef.Time)

	setup.Supernode.ResumeInterop()
	setup.Supernode.AwaitValidatedTimestamp(invalidRef.Time)

	replacement := setup.L2ELB.BlockRefByNumber(invalidRef.Number)
	require.NotEqual(invalidRef.Hash, replacement.Hash,
		"execution-before-initiation block was not replaced")
	setup.L2ELB.AssertTxNotInBlock(invalidRef.Number, execHash)
	setup.L2ELB.AssertBlockDepositOnly(invalidRef.Number)
	require.Equal(preInitRef.Hash, setup.L2ELA.BlockRefByNumber(preInitRef.Number).Hash,
		"unrelated chain-A prefix changed during chain-B invalidation")
	require.Equal(initRef.Hash, setup.L2ELA.BlockRefByNumber(initRef.Number).Hash,
		"later initiating block changed during chain-B invalidation")

	dsl.CheckAll(t,
		setup.L2ACL.ReachedRefFn(safety.CrossSafe, initRef.ID(), 60),
		setup.L2BCL.ReachedFn(safety.CrossSafe, replacement.Number+2, 60),
	)
	setup.Supernode.AwaitValidatedTimestamp(initRef.Time)
	require.Equal(replacement.Hash, setup.L2ELB.BlockRefByNumber(replacement.Number).Hash,
		"replacement lineage changed after sequencing resumed")
	require.Equal(replacement.Hash, setup.L2ELB.BlockRefByNumber(replacement.Number+1).ParentHash,
		"sequencing did not continue from the replacement block")
}
