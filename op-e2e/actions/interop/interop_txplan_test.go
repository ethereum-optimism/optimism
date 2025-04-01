package interop

import (
	"context"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

type txSubmitter struct {
	t       helpers.Testing
	chain   *dsl.Chain
	from    common.Address
	receipt *types.Receipt
}

func (ts *txSubmitter) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	// we need low level interaction here
	// do not submit transactions via RPC, instead directly interact with block builder
	receipt, err := ts.chain.SequencerEngine.EngineApi.IncludeTx(tx, ts.from)
	if err == nil {
		// be aware that this receipt is not finalized...
		// which means its info may be incorrect, such as block hash
		// you must call ActL2EndBlock to seal the L2 block
		ts.receipt = receipt
	}
	return err
}

type receiptGetter struct {
	t     helpers.Testing
	chain *dsl.Chain
	sc    *sources.EthClient
}

func (rg *receiptGetter) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	// close l2 block before fetching actual receipt
	rg.chain.Sequencer.ActL2EndBlock(rg.t)
	return rg.sc.TransactionReceipt(ctx, txHash)
}

func TestTxPlanDeployEventLogger(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)

	aliceA := setupUser(t, is, actors.ChainA, 0)

	l2sc := actors.ChainA.SequencerEngine.SourceClient(t, 10)

	submitter1 := &txSubmitter{t: t, chain: actors.ChainA, from: aliceA.address}
	// txplan options for only tx submission, not ensuring block inclusion
	opts1 := txplan.Combine(
		txplan.WithPrivateKey(aliceA.secret),
		txplan.WithChainID(l2sc),
		txplan.WithAgainstLatestBlock(l2sc),
		txplan.WithPendingNonce(l2sc),
		txplan.WithEstimator(l2sc, false),
		txplan.WithTransactionSubmitter(submitter1),
	)

	actors.ChainA.Sequencer.ActL2StartBlock(t)

	deployCalldata := common.FromHex(bindings.EventloggerBin)
	// tx submitted but not sealed in block
	deployTxWithoutSeal := txplan.NewPlannedTx(opts1, txplan.WithData(deployCalldata))
	_, err := deployTxWithoutSeal.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	latestBlock, err := deployTxWithoutSeal.AgainstBlock.Eval(t.Ctx())
	require.NoError(t, err)

	getter := &receiptGetter{t: t, chain: actors.ChainA, sc: l2sc}
	submitter2 := &txSubmitter{t: t, chain: actors.ChainA, from: aliceA.address}
	// txplan options for tx submission and ensuring block inclusion
	opts2 := txplan.Combine(
		txplan.WithPrivateKey(aliceA.secret),
		txplan.WithChainID(l2sc),
		txplan.WithAgainstLatestBlock(l2sc),
		// no pending nonce
		txplan.WithEstimator(l2sc, false),
		txplan.WithTransactionSubmitter(submitter2),
		txplan.WithAssumedInclusion(getter),
		txplan.WithBlockInclusionInfo(l2sc),
	)
	deployTx := txplan.NewPlannedTx(opts2, txplan.WithData(deployCalldata))
	// manually set nonce because we cannot use the pending nonce
	nonce, err := deployTxWithoutSeal.Nonce.Get()
	require.NoError(t, err)
	deployTx.Nonce.Set(nonce + 1)

	// tx submitted and sealed in block
	// now the tx is actually included in L2 block, as well as included the tx submitted before
	receipt, err := deployTx.Included.Eval(t.Ctx())
	require.NoError(t, err)

	// all intermediate receipts / finalized receipt must contain the contractAddress field
	// because they all deployed contract
	require.NotNil(t, receipt.ContractAddress)
	require.NotNil(t, submitter1.receipt.ContractAddress)
	require.NotNil(t, submitter2.receipt.ContractAddress)

	// different nonce so different contract address
	require.NotEqual(t, submitter1.receipt.ContractAddress, submitter2.receipt.ContractAddress)
	// second and the finalized contract address must be equal
	require.Equal(t, submitter2.receipt.ContractAddress, receipt.ContractAddress)

	includedBlock, err := deployTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// single block advanced
	require.Equal(t, latestBlock.NumberU64()+1, includedBlock.Number)
}

func DefaultTxOpts(t helpers.Testing, user *userWithKeys, chain *dsl.Chain) txplan.Option {
	sc := chain.SequencerEngine.SourceClient(t, 10)
	getter := &receiptGetter{t: t, chain: chain, sc: sc}
	submitter := &txSubmitter{t: t, chain: chain, from: user.address}
	// txplan options for tx submission and ensuring block inclusion
	return txplan.Combine(
		txplan.WithPrivateKey(user.secret),
		txplan.WithChainID(sc),
		txplan.WithAgainstLatestBlock(sc),
		txplan.WithPendingNonce(sc),
		txplan.WithEstimator(sc, false),
		txplan.WithTransactionSubmitter(submitter),
		txplan.WithAssumedInclusion(getter),
		txplan.WithBlockInclusionInfo(sc),
	)
}

func DeployEventLogger(t helpers.Testing, opts txplan.Option) common.Address {
	deployCalldata := common.FromHex(bindings.EventloggerBin)
	deployTx := txplan.NewPlannedTx(opts, txplan.WithData(deployCalldata))
	receipt, err := deployTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	require.NotNil(t, receipt.ContractAddress)
	eventLoggerAddress := receipt.ContractAddress
	return eventLoggerAddress
}

func TestInitAndExecMsgSameTimestamp(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA := DefaultTxOpts(t, alice, actors.ChainA)
	optsB := DefaultTxOpts(t, bob, actors.ChainB)
	actors.ChainA.Sequencer.ActL2StartBlock(t)

	// chain A progressed single unsafe block
	eventLoggerAddress := DeployEventLogger(t, optsA)
	// Also match chain B
	actors.ChainB.Sequencer.ActL2EmptyBlock(t)

	// Intent to initiate message(or emit event) on chain A
	txA := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](optsA)
	randomInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10)
	txA.Content.Set(randomInitTrigger)

	// Trigger single event
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	_, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)

	assertHeads(t, actors.ChainA, 2, 0, 0, 0)

	// Ingest the new unsafe-block event
	actors.ChainA.Sequencer.SyncSupervisor(t)
	// Verify as cross-unsafe with supervisor
	actors.Supervisor.ProcessFull(t)
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainA, 2, 0, 2, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	// Ingest the new unsafe-block event
	actors.ChainB.Sequencer.SyncSupervisor(t)
	// Verify as cross-unsafe with supervisor
	actors.Supervisor.ProcessFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainB, 1, 0, 1, 0)

	// Intent to validate message on chain B
	txB := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsB)
	txB.Content.DependOn(&txA.Result)

	// Single event in tx so index is 0
	txB.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &txA.Result, 0))

	actors.ChainB.Sequencer.ActL2StartBlock(t)
	_, err = txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)

	includedA, err := txA.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	includedB, err := txB.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// initating messages time <= executing message time
	require.Equal(t, includedA.Time, includedB.Time)

	assertHeads(t, actors.ChainB, 2, 0, 1, 0)

	// Ingest the new unsafe-block event
	actors.ChainB.Sequencer.SyncSupervisor(t)
	// Verify as cross-unsafe with supervisor
	actors.Supervisor.ProcessFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, actors.ChainB, 2, 0, 2, 0)
}

func TestBreakTimestampInvariant(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)

	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA := DefaultTxOpts(t, alice, actors.ChainA)
	optsB := DefaultTxOpts(t, bob, actors.ChainB)
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	// chain A progressed single unsafe block
	eventLoggerAddress := DeployEventLogger(t, optsA)

	// Intent to initiate message(or emit event) on chain A
	txA := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](optsA)
	randomInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10)
	txA.Content.Set(randomInitTrigger)

	// Trigger single event
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	_, err := txA.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainA, 2, 0, 0, 0)

	// make supervisor know chainA's unsafe blocks
	actors.ChainA.Sequencer.SyncSupervisor(t)

	// Intent to validate message on chain B
	txB := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsB)
	txB.Content.DependOn(&txA.Result)

	// Single event in tx so index is 0
	txB.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &txA.Result, 0))

	actors.ChainB.Sequencer.ActL2StartBlock(t)
	_, err = txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	includedA, err := txA.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	includedB, err := txB.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// initating messages time <= executing message time
	// BUT we intentionally break the timestamp invariant
	require.Greater(t, includedA.Time, includedB.Time)

	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	actors.ChainB.Batcher.ActSubmitAll(t)
	actors.L1Miner.ActL1StartBlock(12)(t)
	actors.L1Miner.ActL1IncludeTx(actors.ChainB.BatcherAddr)(t)
	actors.L1Miner.ActL1EndBlock(t)

	actors.Supervisor.SignalLatestL1(t)
	t.Log("awaiting L1-exhaust event")
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	t.Log("awaiting supervisor to provide L1 data")
	actors.ChainB.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)
	t.Log("awaiting node to sync")
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	reorgedOutBlock := actors.ChainB.Sequencer.SyncStatus().LocalSafeL2
	require.Equal(t, uint64(1), reorgedOutBlock.Number)

	t.Log("Expecting supervisor to sync and catch local-safe dependency issue")
	actors.ChainB.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)

	assertHeads(t, actors.ChainB, 1, 1, 0, 0)

	// check supervisor head, expect it to be rewound
	localUnsafe, err := actors.Supervisor.LocalUnsafe(t.Ctx(), actors.ChainB.ChainID)
	require.NoError(t, err)
	require.Equal(t, uint64(0), localUnsafe.Number, "unsafe chain needs to be rewound")

	// Make the op-node do the processing to build the replacement
	t.Log("Expecting op-node to build replacement block")
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)

	// Make sure the replaced block has different blockhash
	replacedBlock := actors.ChainB.Sequencer.SyncStatus().LocalSafeL2
	require.NotEqual(t, reorgedOutBlock.Hash, replacedBlock.Hash)

	// but reached block number as 1
	assertHeads(t, actors.ChainB, 1, 1, 1, 1)
}
