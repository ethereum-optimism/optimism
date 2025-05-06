package interop

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/bindings"
	"github.com/ethereum-optimism/optimism/devnet-sdk/contracts/constants"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/interop"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	suptypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// BlockBuilder helps txplan to be integrated with intra block building functionality.
type BlockBuilder struct {
	t                  helpers.Testing
	sc                 *sources.EthClient
	chain              *dsl.Chain
	signer             types.Signer
	intraBlockReceipts []*types.Receipt
	receipts           []*types.Receipt
	keepBlockOpen      bool
}

func (b *BlockBuilder) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	// we need low level interaction here
	// do not submit transactions via RPC, instead directly interact with block builder
	from, err := types.Sender(b.signer, tx)
	if err != nil {
		return err
	}
	intraBlockReceipt, err := b.chain.SequencerEngine.EngineApi.IncludeTx(tx, from)
	if err == nil {
		// be aware that this receipt is not finalized...
		// which means its info may be incorrect, such as block hash
		// you must call ActL2EndBlock to seal the L2 block
		b.intraBlockReceipts = append(b.intraBlockReceipts, intraBlockReceipt)
	}
	return err
}

func (b *BlockBuilder) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	if !b.keepBlockOpen {
		// close l2 block before fetching actual receipt
		b.chain.Sequencer.ActL2EndBlock(b.t)
		b.keepBlockOpen = false
	}
	// retrospectively fill in all resulting receipts after sealing block
	for _, intraBlockReceipt := range b.intraBlockReceipts {
		receipt, _ := b.sc.TransactionReceipt(ctx, intraBlockReceipt.TxHash)
		b.receipts = append(b.receipts, receipt)
	}
	receipt, err := b.sc.TransactionReceipt(ctx, txHash)
	if err == nil {
		b.receipts = append(b.receipts, receipt)
	}
	return receipt, err
}

func TestTxPlanDeployEventLogger(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)

	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)

	aliceA := setupUser(t, is, actors.ChainA, 0)

	nonce := uint64(0)
	opts1, builder1 := DefaultTxOptsWithoutBlockSeal(t, aliceA, actors.ChainA, nonce)
	actors.ChainA.Sequencer.ActL2StartBlock(t)

	deployCalldata := common.FromHex(bindings.EventloggerBin)
	// tx submitted but not sealed in block
	deployTxWithoutSeal := txplan.NewPlannedTx(opts1, txplan.WithData(deployCalldata))
	_, err := deployTxWithoutSeal.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	latestBlock, err := deployTxWithoutSeal.AgainstBlock.Eval(t.Ctx())
	require.NoError(t, err)

	opts2, builder2 := DefaultTxOpts(t, aliceA, actors.ChainA)
	// manually set nonce because we cannot use the pending nonce
	opts2 = txplan.Combine(opts2, txplan.WithStaticNonce(nonce+1))

	deployTx := txplan.NewPlannedTx(opts2, txplan.WithData(deployCalldata))

	receipt, err := deployTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	// now the tx is actually included in L2 block, as well as included the tx submitted before
	// tx submitted and sealed in block

	// all intermediate receipts / finalized receipt must contain the contractAddress field
	// because they all deployed contract
	require.NotNil(t, receipt.ContractAddress)
	require.Equal(t, 1, len(builder1.intraBlockReceipts))
	require.Equal(t, 1, len(builder2.intraBlockReceipts))
	require.NotNil(t, builder1.intraBlockReceipts[0].ContractAddress)
	require.NotNil(t, builder2.intraBlockReceipts[0].ContractAddress)

	// different nonce so different contract address
	require.NotEqual(t, builder1.intraBlockReceipts[0].ContractAddress, builder2.intraBlockReceipts[0].ContractAddress)
	// second and the finalized contract address must be equal
	require.Equal(t, builder2.intraBlockReceipts[0].ContractAddress, receipt.ContractAddress)

	includedBlock, err := deployTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// single block advanced
	require.Equal(t, latestBlock.NumberU64()+1, includedBlock.Number)
}

func DefaultTxOpts(t helpers.Testing, user *userWithKeys, chain *dsl.Chain) (txplan.Option, *BlockBuilder) {
	sc := chain.SequencerEngine.SourceClient(t, 10)
	signer := types.LatestSignerForChainID(chain.ChainID.ToBig())
	builder := &BlockBuilder{t: t, chain: chain, sc: sc, signer: signer}
	// txplan options for tx submission and ensuring block inclusion
	return txplan.Combine(
		txplan.WithPrivateKey(user.secret),
		txplan.WithChainID(sc),
		txplan.WithAgainstLatestBlock(sc),
		txplan.WithPendingNonce(sc),
		txplan.WithEstimator(sc, false),
		txplan.WithTransactionSubmitter(builder),
		txplan.WithAssumedInclusion(builder),
		txplan.WithBlockInclusionInfo(sc),
	), builder
}

func DefaultTxOptsWithoutBlockSeal(t helpers.Testing, user *userWithKeys, chain *dsl.Chain, nonce uint64) (txplan.Option, *BlockBuilder) {
	sc := chain.SequencerEngine.SourceClient(t, 10)
	signer := types.LatestSignerForChainID(chain.ChainID.ToBig())
	builder := &BlockBuilder{t: t, chain: chain, sc: sc, keepBlockOpen: true, signer: signer}
	return txplan.Combine(
		txplan.WithPrivateKey(user.secret),
		txplan.WithChainID(sc),
		txplan.WithAgainstLatestBlock(sc),
		// nonce must be manually set since pending nonce may be incorrect
		txplan.WithNonce(nonce),
		txplan.WithEstimator(sc, false),
		txplan.WithTransactionSubmitter(builder),
		txplan.WithAssumedInclusion(builder),
		txplan.WithBlockInclusionInfo(sc),
	), builder
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

func consolidateToSafe(t helpers.Testing, actors *dsl.InteropActors, startA, startB, endA, endB uint64) {
	// Batch L2 blocks of chain A, B and submit to L1 to ensure safe head advances without a reorg.
	// Checking cross-unsafe consolidation is sufficient for sanity check but lets add safe check as well.
	actors.ChainA.Batcher.ActSubmitAll(t)
	actors.ChainB.Batcher.ActSubmitAll(t)
	actors.L1Miner.ActL1StartBlock(12)(t)
	actors.L1Miner.ActL1IncludeTx(actors.ChainA.BatcherAddr)(t)
	actors.L1Miner.ActL1IncludeTx(actors.ChainB.BatcherAddr)(t)
	actors.L1Miner.ActL1EndBlock(t)

	actors.Supervisor.SignalLatestL1(t)

	t.Log("awaiting L1-exhaust event")
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainA, endA, startA, startA, startA)
	assertHeads(t, actors.ChainB, endB, startB, startB, startB)

	t.Log("awaiting supervisor to provide L1 data")
	actors.ChainA.Sequencer.SyncSupervisor(t)
	actors.ChainB.Sequencer.SyncSupervisor(t)
	assertHeads(t, actors.ChainA, endA, startA, startA, startA)
	assertHeads(t, actors.ChainB, endB, startB, startB, startB)

	t.Log("awaiting node to sync: unsafe to local-safe")
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, actors.ChainA, endA, endA, startA, startA)
	assertHeads(t, actors.ChainB, endB, endB, startB, startB)

	t.Log("expecting supervisor to sync")
	actors.ChainA.Sequencer.SyncSupervisor(t)
	actors.ChainB.Sequencer.SyncSupervisor(t)
	assertHeads(t, actors.ChainA, endA, endA, startA, startA)
	assertHeads(t, actors.ChainB, endB, endB, startB, startB)

	t.Log("supervisor promotes cross-unsafe and safe")
	actors.Supervisor.ProcessFull(t)

	t.Log("awaiting nodes to sync: local-safe to safe")
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, actors.ChainA, endA, endA, endA, endA)
	assertHeads(t, actors.ChainB, endB, endB, endB, endB)
}

// reorgOutUnsafeAndConsolidateToSafe assume that chainY is reorged, but chainX is not.
// chainY is expected to experience cross-unsafe invalidation and reorging unsafe blocks.
// Consolidate with steps: unsafe -> cross-unsafe -> local-safe -> safe
func reorgOutUnsafeAndConsolidateToSafe(t helpers.Testing, actors *dsl.InteropActors, chainX, chainY *dsl.Chain, startX, startY, endX, endY, unsafeHeadNumAfterReorg uint64) {
	require.GreaterOrEqual(t, endY, unsafeHeadNumAfterReorg)
	// Check to make batcher happy
	require.Positive(t, endY-startY)
	require.Positive(t, endX-startX)

	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, chainX, endX, startX, endX, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	// check chain Y and and supervisor view of chain Y is consistent
	reorgedOutBlock := chainY.Sequencer.SyncStatus().UnsafeL2
	require.Equal(t, unsafeHeadNumAfterReorg+1, reorgedOutBlock.Number)
	localUnsafe, err := actors.Supervisor.LocalUnsafe(t.Ctx(), chainY.ChainID)
	require.NoError(t, err)
	require.Equal(t, reorgedOutBlock.ID(), localUnsafe)

	// now try to advance safe heads
	chainX.Batcher.ActSubmitAll(t)
	chainY.Batcher.ActSubmitAll(t)
	actors.L1Miner.ActL1StartBlock(12)(t)
	actors.L1Miner.ActL1IncludeTx(chainX.BatcherAddr)(t)
	actors.L1Miner.ActL1IncludeTx(chainY.BatcherAddr)(t)
	actors.L1Miner.ActL1EndBlock(t)

	actors.Supervisor.SignalLatestL1(t)

	t.Log("awaiting L1-exhaust event")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, chainX, endX, startX, endX, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	t.Log("awaiting supervisor to provide L1 data")
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	assertHeads(t, chainX, endX, startX, endX, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	t.Log("awaiting node to sync: unsafe to local-safe")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, chainX, endX, endX, endX, startX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, startY)

	t.Log("expecting supervisor to sync")
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	assertHeads(t, chainX, endX, endX, endX, startX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, startY)

	t.Log("supervisor promotes cross-unsafe and safe")
	actors.Supervisor.ProcessFull(t)

	// check supervisor head, expect it to be rewound
	localUnsafe, err = actors.Supervisor.LocalUnsafe(t.Ctx(), chainY.ChainID)
	require.NoError(t, err)
	require.Equal(t, unsafeHeadNumAfterReorg, localUnsafe.Number, "unsafe chain needs to be rewound")

	t.Log("awaiting nodes to sync: local-safe to safe")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, chainX, endX, endX, endX, endX)
	assertHeads(t, chainY, endY, endY, endY, endY)

	// Make sure the replaced block has different blockhash
	replacedBlock := chainY.Sequencer.SyncStatus().LocalSafeL2
	require.NotEqual(t, reorgedOutBlock.Hash, replacedBlock.Hash)
	require.Equal(t, reorgedOutBlock.Number, replacedBlock.Number)
	require.Equal(t, unsafeHeadNumAfterReorg+1, replacedBlock.Number)
}

// reorgOutUnsafeAndConsolidateToSafeBothChain assumes both chainX and chainY are reorged.
// chain{X|Y} both expected to experience cross-unsafe invalidation and reorging unsafe blocks.
// Consolidate with steps: unsafe -> cross-unsafe -> local-safe -> safe
func reorgOutUnsafeAndConsolidateToSafeBothChain(t helpers.Testing, actors *dsl.InteropActors, chainX, chainY *dsl.Chain, startX, startY, endX, endY, unsafeHeadNumAfterReorg uint64) {
	require.GreaterOrEqual(t, endY, unsafeHeadNumAfterReorg)
	// Check to make batcher happy
	require.Less(t, startX, endX)
	require.Less(t, startY, endY)

	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, chainX, endX, startX, unsafeHeadNumAfterReorg, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	l2chains := []*dsl.Chain{chainX, chainY}

	// check chains and supervisor views are consistent
	reorgedOutBlocks := []eth.L2BlockRef{}
	for _, chain := range l2chains {
		reorgedOutBlock := chain.Sequencer.SyncStatus().UnsafeL2
		require.Equal(t, unsafeHeadNumAfterReorg+1, reorgedOutBlock.Number)
		localUnsafe, err := actors.Supervisor.LocalUnsafe(t.Ctx(), chain.ChainID)
		require.NoError(t, err)
		require.Equal(t, reorgedOutBlock.ID(), localUnsafe)
		reorgedOutBlocks = append(reorgedOutBlocks, reorgedOutBlock)
	}

	// now try to advance safe heads
	chainX.Batcher.ActSubmitAll(t)
	chainY.Batcher.ActSubmitAll(t)
	actors.L1Miner.ActL1StartBlock(12)(t)
	actors.L1Miner.ActL1IncludeTx(chainX.BatcherAddr)(t)
	actors.L1Miner.ActL1IncludeTx(chainY.BatcherAddr)(t)
	actors.L1Miner.ActL1EndBlock(t)

	actors.Supervisor.SignalLatestL1(t)

	t.Log("awaiting L1-exhaust event")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, chainX, endX, startX, unsafeHeadNumAfterReorg, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	t.Log("awaiting supervisor to provide L1 data")
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	assertHeads(t, chainX, endX, startX, unsafeHeadNumAfterReorg, startX)
	assertHeads(t, chainY, endY, startY, unsafeHeadNumAfterReorg, startY)

	t.Log("awaiting node to sync: unsafe to local-safe")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, chainX, endX, endX, unsafeHeadNumAfterReorg, startX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, startY)

	t.Log("expecting supervisor to sync")
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	assertHeads(t, chainX, endX, endX, unsafeHeadNumAfterReorg, startX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, startY)

	t.Log("supervisor promotes cross-unsafe and safe")
	actors.Supervisor.ProcessFull(t)

	t.Log("awaiting nodes to sync: local-safe to safe")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)
	assertHeads(t, chainX, endX, endX, endX, endX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, unsafeHeadNumAfterReorg)

	t.Log("expecting supervisor to sync")
	chainX.Sequencer.SyncSupervisor(t)
	chainY.Sequencer.SyncSupervisor(t)
	assertHeads(t, chainX, endX, endX, endX, endX)
	assertHeads(t, chainY, endY, endY, unsafeHeadNumAfterReorg, unsafeHeadNumAfterReorg)

	t.Log("supervisor promotes cross-unsafe and safe")
	actors.Supervisor.ProcessFull(t)

	t.Log("awaiting nodes to sync: local-safe to safe")
	chainX.Sequencer.ActL2PipelineFull(t)
	chainY.Sequencer.ActL2PipelineFull(t)

	assertHeads(t, chainX, endX, endX, endX, endX)
	assertHeads(t, chainY, endY, endY, endY, endY)

	// Make sure the replaced blocks have different blockhash
	for idx, chain := range l2chains {
		reorgedOutBlock := reorgedOutBlocks[idx]
		replacedBlock := chain.Sequencer.SyncStatus().LocalSafeL2
		require.NotEqual(t, reorgedOutBlock.Hash, replacedBlock.Hash)
		require.Equal(t, reorgedOutBlock.Number, replacedBlock.Number)
		require.Equal(t, unsafeHeadNumAfterReorg+1, replacedBlock.Number)
	}
}

func TestInitAndExecMsgSameTimestamp(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)

	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
	optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)
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

	// initiating messages time <= executing message time
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
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
	optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)
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

	// initiating messages time <= executing message time
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

// TestExecMsgDifferTxIndex tests below scenario:
// Execute message that links with initiating message with: first, random or last tx of a block.
func TestExecMsgDifferTxIndex(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	// only unsafe head of each chain progresses in this code block
	var targetNum uint64
	{
		alice := setupUser(t, is, actors.ChainA, 0)
		bob := setupUser(t, is, actors.ChainB, 0)

		optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
		optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)
		actors.ChainA.Sequencer.ActL2StartBlock(t)
		// chain A progressed single unsafe block
		eventLoggerAddress := DeployEventLogger(t, optsA)

		// attempt to include multiple txs in a single L2 block
		actors.ChainA.Sequencer.ActL2StartBlock(t)
		// start with nonce as 1 because alice deployed the EventLogger

		nonce := uint64(1)
		txCount := 3 + rng.Intn(15)
		initTxs := []*txintent.IntentTx[*txintent.InitTrigger, *txintent.InteropOutput]{}
		for range txCount {
			opts, _ := DefaultTxOptsWithoutBlockSeal(t, alice, actors.ChainA, nonce)

			// Intent to initiate message(or emit event) on chain A
			tx := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](opts)
			initTxs = append(initTxs, tx)
			randomInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddress, 3, 10)
			tx.Content.Set(randomInitTrigger)

			// Trigger single event
			_, err := tx.PlannedTx.Submitted.Eval(t.Ctx())
			require.NoError(t, err)

			nonce += 1
		}
		actors.ChainA.Sequencer.ActL2EndBlock(t)

		// fetch receipts since all txs are included in the block and sealed
		for _, tx := range initTxs {
			includedBlock, err := tx.PlannedTx.IncludedBlock.Eval(t.Ctx())
			require.NoError(t, err)
			// all txCount txs are included at same block
			require.Equal(t, uint64(2), includedBlock.Number)
		}
		assertHeads(t, actors.ChainA, 2, 0, 0, 0)

		// advance chain B for satisfying the timestamp invariant
		actors.ChainB.Sequencer.ActL2EmptyBlock(t)
		assertHeads(t, actors.ChainB, 1, 0, 0, 0)

		// first, random or last tx of a single L2 block.
		indexes := []int{0, 1 + rng.Intn(txCount-1), txCount - 1}
		for blockNumDelta, index := range indexes {
			actors.ChainB.Sequencer.ActL2StartBlock(t)

			initTx := initTxs[index]
			execTx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsB)

			// Single event in every tx so index is always 0
			execTx.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &initTx.Result, 0))
			execTx.Content.DependOn(&initTx.Result)

			includedBlock, err := execTx.PlannedTx.IncludedBlock.Eval(t.Ctx())
			require.NoError(t, err)

			// each block contains single executing message
			require.Equal(t, uint64(2+blockNumDelta), includedBlock.Number)
		}
		targetNum = uint64(1 + len(indexes))
		assertHeads(t, actors.ChainB, targetNum, 0, 0, 0)
	}
	// store unsafe head of chain B to compare after consolidation
	chainBUnsafeHead := actors.ChainB.Sequencer.SyncStatus().UnsafeL2
	require.Equal(t, targetNum, chainBUnsafeHead.Number)
	require.Equal(t, uint64(4), targetNum)

	consolidateToSafe(t, actors, 0, 0, 2, 4)

	// unsafe head of chain B did not get updated
	require.Equal(t, chainBUnsafeHead, actors.ChainB.Sequencer.SyncStatus().UnsafeL2)
	// unsafe head of chain B consolidated to safe
	require.Equal(t, chainBUnsafeHead, actors.ChainB.Sequencer.SyncStatus().SafeL2)
}

// TestExpiredMessage tests below scenario:
// Execute message with current timestamp > the lower-bound expiry timestamp.
func TestExpiredMessage(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))

	expiryTime := uint64(6)
	is := dsl.SetupInterop(t, dsl.SetMessageExpiryTime(expiryTime))
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
	optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)
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

	// advance chain B to reach expiry
	targetNumblocksUntilExpiry := expiryTime / actors.ChainA.RollupCfg.BlockTime
	for range 2 + targetNumblocksUntilExpiry {
		actors.ChainB.Sequencer.ActL2EmptyBlock(t)
	}
	assertHeads(t, actors.ChainB, 2+targetNumblocksUntilExpiry, 0, 0, 0)

	// check that chain B unsafe head reached tip of expiry
	require.Equal(t, actors.ChainA.Sequencer.SyncStatus().UnsafeL2.Time+expiryTime, actors.ChainB.Sequencer.SyncStatus().UnsafeL2.Time)

	// Intent to validate message on chain B
	txB := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsB)
	txB.Content.DependOn(&txA.Result)

	// Single event in tx so index is 0
	txB.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &txA.Result, 0))

	actors.ChainB.Sequencer.ActL2StartBlock(t)
	_, err = txB.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	expiredMsgBlockNum := 2 + targetNumblocksUntilExpiry + 1
	assertHeads(t, actors.ChainB, expiredMsgBlockNum, 0, 0, 0)

	includedA, err := txA.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	includedB, err := txB.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// initiating messages time + expiryTime >= executing message time
	// BUT we intentionally break the message expiry invariant
	require.Greater(t, includedB.Time, includedA.Time+expiryTime)

	reorgOutUnsafeAndConsolidateToSafe(t, actors, actors.ChainA, actors.ChainB, 0, 0, 2, expiredMsgBlockNum, expiredMsgBlockNum-1)
}

// TestCrossPatternSameTimestamp tests below scenario:
// Transaction on B executes message from A, and vice-versa. Cross-pattern, within same timestamp:
// Four transactions happen in same timestamp:
// tx0: chainA: alice initiates message X
// tx1: chainB: bob executes message X
// tx2: chainB: bob initiates message Y
// tx3: chainA: alice executes message Y
func TestCrossPatternSameTimestamp(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	// deploy eventLogger per chain
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)
	actors.ChainB.Sequencer.ActL2StartBlock(t)
	deployOptsB, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainB, 1), actors.ChainB)
	eventLoggerAddressB := DeployEventLogger(t, deployOptsB)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	require.Equal(t, actors.ChainA.RollupCfg.Genesis.L2Time, actors.ChainB.RollupCfg.Genesis.L2Time)
	// assume all four txs land in block number 2, same time
	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	// start with nonce as 0 for both alice and bob
	nonce := uint64(0)
	optsA, builderA := DefaultTxOptsWithoutBlockSeal(t, alice, actors.ChainA, nonce)
	optsB, builderB := DefaultTxOptsWithoutBlockSeal(t, bob, actors.ChainB, nonce)

	// open blocks on both chains
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	actors.ChainB.Sequencer.ActL2StartBlock(t)

	// Intent to initiate message X on chain A
	tx0 := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](optsA)
	tx0.Content.Set(interop.RandomInitTrigger(rng, eventLoggerAddressA, 3, 10))

	_, err := tx0.PlannedTx.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	// manually update the included info since block is not sealed yet
	require.NotNil(t, builderA.intraBlockReceipts[0])
	tx0.PlannedTx.Included.Set(builderA.intraBlockReceipts[0])
	tx0.PlannedTx.IncludedBlock.Set(eth.BlockRef{Time: targetTime})

	// Intent to execute message X on chain B
	tx1 := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsB)
	tx1.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx0.Result, 0))
	tx1.Content.DependOn(&tx0.Result)

	_, err = tx1.PlannedTx.Submitted.Eval(t.Ctx())
	require.NoError(t, err)

	// Intent to initiate message Y on chain B
	// override nonce
	optsB = txplan.Combine(optsB, txplan.WithStaticNonce(nonce+1))
	tx2 := txintent.NewIntent[*txintent.InitTrigger, *txintent.InteropOutput](optsB)
	tx2.Content.Set(interop.RandomInitTrigger(rng, eventLoggerAddressB, 4, 9))
	_, err = tx2.PlannedTx.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	// manually update the included info since block is not sealed yet
	require.NotNil(t, builderB.intraBlockReceipts[0])
	tx2.PlannedTx.Included.Set(builderB.intraBlockReceipts[0])
	tx2.PlannedTx.IncludedBlock.Set(eth.BlockRef{Time: targetTime})

	// Intent to execute message Y on chain A
	// override nonce
	optsA = txplan.Combine(optsA, txplan.WithStaticNonce(nonce+1))
	tx3 := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsA)
	tx3.Content.Fn(txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx2.Result, 0))
	tx3.Content.DependOn(&tx2.Result)

	_, err = tx3.PlannedTx.Submitted.Eval(t.Ctx())
	require.NoError(t, err)

	// finally seal block
	actors.ChainA.Sequencer.ActL2EndBlock(t)
	actors.ChainB.Sequencer.ActL2EndBlock(t)
	assertHeads(t, actors.ChainA, 2, 0, 0, 0)
	assertHeads(t, actors.ChainB, 2, 0, 0, 0)

	// store unsafe head of chain A, B to compare after consolidation
	chainAUnsafeHead := actors.ChainA.Sequencer.SyncStatus().UnsafeL2
	chainBUnsafeHead := actors.ChainB.Sequencer.SyncStatus().UnsafeL2

	consolidateToSafe(t, actors, 0, 0, 2, 2)

	// unsafe heads consolidated to safe
	require.Equal(t, chainAUnsafeHead, actors.ChainA.Sequencer.SyncStatus().SafeL2)
	require.Equal(t, chainBUnsafeHead, actors.ChainB.Sequencer.SyncStatus().SafeL2)

	t.Log("Check that all tx included blocks and receipts can be fetched using the RPC")
	targetBlockNum := uint64(2)

	// check tx1 first instead of tx0 not to make tx0 submitted
	receipt, err := tx1.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetBlockNum, receipt.BlockNumber.Uint64())
	block, err := tx1.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetTime, block.Time)

	// invalidate receipt which is incorrect since it was fetched intra block
	tx0.PlannedTx.Included.Invalidate()
	tx0.PlannedTx.IncludedBlock.Invalidate()
	block, err = tx0.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetBlockNum, block.Number)
	require.Equal(t, targetTime, block.Time)

	// check tx3 first instead of tx2 not to make tx2 submitted
	receipt, err = tx3.PlannedTx.Included.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetBlockNum, receipt.BlockNumber.Uint64())
	block, err = tx3.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetTime, block.Time)

	// invalidate receipt which is incorrect since it was fetched intra block
	tx2.PlannedTx.Included.Invalidate()
	tx2.PlannedTx.IncludedBlock.Invalidate()
	block, err = tx2.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, targetBlockNum, block.Number)
	require.Equal(t, targetTime, block.Time)
}

// TestCrossPatternSameTx tests below scenario:
// Transaction on B executes message from A, and vice-versa. Cross-pattern, within same tx: inter-dependent but non-cyclic txs.
// Two transactions happen in same timestamp:
// txA: chain A: alice initiates message X and executes message Y
// txB: chain B: bob initiates message Y and executes message X
func TestCrossPatternSameTx(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	// deploy eventLogger per chain
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)
	actors.ChainB.Sequencer.ActL2StartBlock(t)
	deployOptsB, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainB, 1), actors.ChainB)
	eventLoggerAddressB := DeployEventLogger(t, deployOptsB)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	require.Equal(t, actors.ChainA.RollupCfg.Genesis.L2Time, actors.ChainB.RollupCfg.Genesis.L2Time)
	// assume all two txs land in block number 2, same time
	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)
	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
	optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)

	// open blocks on both chains
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	actors.ChainB.Sequencer.ActL2StartBlock(t)

	// speculatively build exec messages by knowing necessary info to build Message
	initX := interop.RandomInitTrigger(rng, eventLoggerAddressA, 3, 10)
	logIndexX, logIndexY := uint(0), uint(0)
	execX, err := interop.ExecTriggerFromInitTrigger(initX, logIndexX, targetNum, targetTime, actors.ChainA.ChainID)
	require.NoError(t, err)
	initY := interop.RandomInitTrigger(rng, eventLoggerAddressB, 4, 7)
	execY, err := interop.ExecTriggerFromInitTrigger(initY, logIndexY, targetNum, targetTime, actors.ChainB.ChainID)
	require.NoError(t, err)

	callsA := []txintent.Call{initX, execY}
	callsB := []txintent.Call{initY, execX}

	// Intent to initiate message X and execute message Y at chain A
	txA := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](optsA)
	txA.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: callsA})
	// Intent to initiate message Y and execute message X at chain B
	txB := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](optsB)
	txB.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: callsB})

	includedA, err := txA.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	includedB, err := txB.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// Make sure two txs both sealed in block at expected time
	require.Equal(t, includedA.Time, targetTime)
	require.Equal(t, includedA.Number, targetNum)
	require.Equal(t, includedB.Time, targetTime)
	require.Equal(t, includedB.Number, targetNum)

	assertHeads(t, actors.ChainA, targetNum, 0, 0, 0)
	assertHeads(t, actors.ChainB, targetNum, 0, 0, 0)

	// confirm speculatively built exec message X by rebuilding after txA inclusion
	_, err = txA.Result.Eval(t.Ctx())
	require.NoError(t, err)
	multiTriggerA, err := txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txA.Result, []int{int(logIndexX)})(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, multiTriggerA.Calls[logIndexX], execX)

	// confirm speculatively built exec message Y by rebuilding after txB inclusion
	_, err = txB.Result.Eval(t.Ctx())
	require.NoError(t, err)
	multiTriggerB, err := txintent.ExecuteIndexeds(constants.MultiCall3, constants.CrossL2Inbox, &txB.Result, []int{int(logIndexY)})(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, multiTriggerB.Calls[logIndexY], execY)

	// store unsafe head of chain A, B to compare after consolidation
	chainAUnsafeHead := actors.ChainA.Sequencer.SyncStatus().UnsafeL2
	chainBUnsafeHead := actors.ChainB.Sequencer.SyncStatus().UnsafeL2

	consolidateToSafe(t, actors, 0, 0, targetNum, targetNum)

	// unsafe heads consolidated to safe
	require.Equal(t, chainAUnsafeHead, actors.ChainA.Sequencer.SyncStatus().SafeL2)
	require.Equal(t, chainBUnsafeHead, actors.ChainB.Sequencer.SyncStatus().SafeL2)
}

// TestCycleInTx tests below scenario:
// Transaction executes message, then initiates it: cycle with self
func TestCycleInTx(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)

	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)

	// assume tx which multicalls exec message and init message land in block number 2
	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)
	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)

	// open blocks
	actors.ChainA.Sequencer.ActL2StartBlock(t)

	// speculatively build exec message by knowing necessary info to build Message
	init := interop.RandomInitTrigger(rng, eventLoggerAddressA, 3, 10)
	// log index of init message is 1, not 0 because exec message will firstly executed, emitting a single log
	logIndexX := uint(1)
	exec, err := interop.ExecTriggerFromInitTrigger(init, logIndexX, targetNum, targetTime, actors.ChainA.ChainID)

	require.NoError(t, err)

	// tx includes cycle with self
	calls := []txintent.Call{exec, init}

	// Intent to execute message X and initiate message X at chain A
	tx := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](optsA)
	tx.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: calls})

	included, err := tx.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// Make sure tx in block sealed at expected time
	require.Equal(t, included.Time, targetTime)
	require.Equal(t, included.Number, targetNum)

	// confirm speculatively built exec message by rebuilding after tx inclusion
	_, err = tx.Result.Eval(t.Ctx())
	require.NoError(t, err)
	exec2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx.Result, int(logIndexX))(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, exec2, exec)

	// Make batcher happy by advancing at least a single block
	actors.ChainB.Sequencer.ActL2EmptyBlock(t)

	unsafeHeadNumAfterReorg := targetNum - 1
	reorgOutUnsafeAndConsolidateToSafe(t, actors, actors.ChainB, actors.ChainA, 0, 0, 1, targetNum, unsafeHeadNumAfterReorg)
}

// submitIntent method submits txintent to a chain.
// Useful when building blocks with transactions, without block sealing.
// It receives nonce because chain does not hold updated pending nonce yet because the method assumes
// block sealing is not done yet.
var submitIntent = func(
	t helpers.StatefulTesting,
	trigger txintent.Call,
	nonce *uint64,
	user *userWithKeys,
	chain *dsl.Chain,
	intents *[]*txintent.IntentTx[txintent.Call, *txintent.InteropOutput],
) {
	opts, _ := DefaultTxOptsWithoutBlockSeal(t, user, chain, *nonce)
	intent := txintent.NewIntent[txintent.Call, *txintent.InteropOutput](opts)
	intent.Content.Set(trigger)
	_, err := intent.PlannedTx.Submitted.Eval(t.Ctx())
	require.NoError(t, err)
	*intents = append(*intents, intent)
	*nonce += 1
}

// TestCycleInBlock tests below scenario:
// Transaction executes message, then initiates it: cycle in block
// To elaborate, single block contains txs in below order:
// {exec message X tx, dummy tx, ..., dummy tx, init message X tx}
func TestCycleInBlock(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)

	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)

	// assume every tx including target exec message and init message land in block number 2
	// all other txs are dummy tx to make block include multiple tx
	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)

	// attempt to include multiple txs in a single L2 block
	actors.ChainA.Sequencer.ActL2StartBlock(t)

	nonce := uint64(0)
	txCount := uint64(2 + rng.Intn(20))

	// speculatively build exec message by knowing necessary info to build Message
	init := interop.RandomInitTrigger(rng, eventLoggerAddressA, 3, 10)
	// log index of init message is txCount - 1, not 0 because each tx before init message X emits a single log
	logIndexX := uint(txCount - 1)
	exec, err := interop.ExecTriggerFromInitTrigger(init, logIndexX, targetNum, targetTime, actors.ChainA.ChainID)
	require.NoError(t, err)

	intents := []*txintent.IntentTx[txintent.Call, *txintent.InteropOutput]{}

	// include exec message X tx first in block
	submitIntent(t, exec, &nonce, alice, actors.ChainA, &intents)
	// include dummy txs in block
	for range txCount - 2 {
		randomInitTrigger := interop.RandomInitTrigger(rng, eventLoggerAddressA, 3, 10)
		submitIntent(t, randomInitTrigger, &nonce, alice, actors.ChainA, &intents)
	}
	// include init message X last in block
	submitIntent(t, init, &nonce, alice, actors.ChainA, &intents)
	require.Equal(t, txCount, nonce)

	actors.ChainA.Sequencer.ActL2EndBlock(t)

	// Make sure tx in block sealed at expected time
	for _, intent := range intents {
		included, err := intent.PlannedTx.IncludedBlock.Eval(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, included.Time, targetTime)
		require.Equal(t, included.Number, targetNum)
	}

	// confirm speculatively built exec message by rebuilding after tx inclusion
	tx := intents[txCount-1]
	_, err = tx.Result.Eval(t.Ctx())
	require.NoError(t, err)
	// log index is 0 because tx emitted a single log
	exec2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx.Result, 0)(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, exec2, exec)

	// Make batcher happy by advancing at least a single block
	actors.ChainB.Sequencer.ActL2EmptyBlock(t)

	unsafeHeadNumAfterReorg := targetNum - 1
	reorgOutUnsafeAndConsolidateToSafe(t, actors, actors.ChainB, actors.ChainA, 0, 0, 1, targetNum, unsafeHeadNumAfterReorg)
}

// TestCycleAcrossChainsSameTimestamp tests below scenario:
// Transaction B0 exec chain A1, A0 exec B1: cycle across chains: within same timestamp
// Four transactions happen in same timestamp:
// tx0: chainA: alice executes message X
// tx1: chainB: bob executes message Y
// tx2: chainB: bob initiates message X
// tx3: chainA: alice initiates message Y
// tx0 depends on tx2 (init exec relation)
// tx3 depends on tx0 (tx order)
// tx1 depends on tx3 (init exec relation)
// tx2 depends on tx1 (tx order)
// cycle: tx0 -> tx3 -> tx1 -> tx2 -> tx0
func TestCycleAcrossChainsSameTimestamp(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)
	actors.ChainB.Sequencer.ActL2StartBlock(t)
	deployOptsB, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainB, 1), actors.ChainB)
	eventLoggerAddressB := DeployEventLogger(t, deployOptsB)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)

	// open blocks on both chains
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	actors.ChainB.Sequencer.ActL2StartBlock(t)

	// speculatively build exec message by knowing necessary info to build Message
	// log index of init messages are 1, not 0 because exec message will firstly executed, emitting a single log
	logIndexX, logIndexY := uint(1), uint(1)
	initX := interop.RandomInitTrigger(rng, eventLoggerAddressB, 3, 10)
	execX, err := interop.ExecTriggerFromInitTrigger(initX, logIndexX, targetNum, targetTime, actors.ChainB.ChainID)
	require.NoError(t, err)
	initY := interop.RandomInitTrigger(rng, eventLoggerAddressA, 2, 15)
	execY, err := interop.ExecTriggerFromInitTrigger(initY, logIndexY, targetNum, targetTime, actors.ChainA.ChainID)
	require.NoError(t, err)

	intents := []*txintent.IntentTx[txintent.Call, *txintent.InteropOutput]{}

	nonceA, nonceB := uint64(0), uint64(0)
	// tx0: Intent to execute message X at chain A
	submitIntent(t, execX, &nonceA, alice, actors.ChainA, &intents)
	// tx1: Intent to execute message Y at chain B
	submitIntent(t, execY, &nonceB, bob, actors.ChainB, &intents)
	// tx2: Intent to initiate message X at chain B
	submitIntent(t, initX, &nonceB, bob, actors.ChainB, &intents)
	// tx3: Intent to initiate message Y at chain A
	submitIntent(t, initY, &nonceA, alice, actors.ChainA, &intents)
	require.Equal(t, uint64(2), nonceA)
	require.Equal(t, uint64(2), nonceB)

	actors.ChainA.Sequencer.ActL2EndBlock(t)
	actors.ChainB.Sequencer.ActL2EndBlock(t)

	// Make sure tx in block sealed at expected time
	includedBlocks := []eth.BlockRef{}
	for _, intent := range intents {
		included, err := intent.PlannedTx.IncludedBlock.Eval(t.Ctx())
		require.NoError(t, err)
		require.Equal(t, included.Time, targetTime)
		require.Equal(t, included.Number, targetNum)
		includedBlocks = append(includedBlocks, included)
	}
	// tx0 and tx3 land in same block at chain A
	require.Equal(t, includedBlocks[0], includedBlocks[3])
	// tx1 and tx2 land in same block at chain B
	require.Equal(t, includedBlocks[1], includedBlocks[2])

	// confirm speculatively built exec message by rebuilding after tx inclusion
	tx2 := intents[2]
	_, err = tx2.Result.Eval(t.Ctx())
	require.NoError(t, err)
	// log index is 0 because tx emitted a single log
	execX2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx2.Result, 0)(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, execX2, execX)
	tx3 := intents[3]
	_, err = tx3.Result.Eval(t.Ctx())
	require.NoError(t, err)
	// log index is 0 because tx emitted a single log
	execY2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx3.Result, 0)(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, execY2, execY)

	unsafeHeadNumAfterReorg := targetNum - 1
	reorgOutUnsafeAndConsolidateToSafeBothChain(t, actors, actors.ChainA, actors.ChainB, 0, 0, targetNum, targetNum, unsafeHeadNumAfterReorg)
}

// TestCycleAcrossChainsSameTx tests below scenario:
// Transaction B0 exec chain A1, A0 exec B1: cycle across chains: within same tx: inter-dependent and cyclic
// Two transactions happen in same timestamp:
// tx0: chainA: alice executes message X, then initiates message Y
// tx1: chainB: bob executes message Y, then initiates message X
// tx0 depends on tx1 (init exec relation)
// tx1 depends on tx0 (init exec relation)
// cycle: tx0 -> tx1 -> tx0
func TestCycleAcrossChainsSameTx(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareAndVerifyInitialState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)
	optsB, _ := DefaultTxOpts(t, bob, actors.ChainB)

	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)
	actors.ChainB.Sequencer.ActL2StartBlock(t)
	deployOptsB, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainB, 1), actors.ChainB)
	eventLoggerAddressB := DeployEventLogger(t, deployOptsB)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)

	// open blocks on both chains
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	actors.ChainB.Sequencer.ActL2StartBlock(t)

	// speculatively build exec message by knowing necessary info to build Message
	initX := interop.RandomInitTrigger(rng, eventLoggerAddressB, 3, 10)
	initY := interop.RandomInitTrigger(rng, eventLoggerAddressA, 2, 15)
	// log index of init messages are 1, not 0 because exec message will firstly executed, emitting a single log
	logIndexX, logIndexY := uint(1), uint(1)
	execX, err := interop.ExecTriggerFromInitTrigger(initX, logIndexX, targetNum, targetTime, actors.ChainB.ChainID)
	require.NoError(t, err)
	execY, err := interop.ExecTriggerFromInitTrigger(initY, logIndexY, targetNum, targetTime, actors.ChainA.ChainID)
	require.NoError(t, err)

	// tx0 executes message X, then initiates message Y
	tx0 := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](optsA)
	tx0.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: []txintent.Call{execX, initY}})
	// tx1 executes message Y, then initiates message X
	tx1 := txintent.NewIntent[*txintent.MultiTrigger, *txintent.InteropOutput](optsB)
	tx1.Content.Set(&txintent.MultiTrigger{Emitter: constants.MultiCall3, Calls: []txintent.Call{execY, initX}})

	included0, err := tx0.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	included1, err := tx1.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)

	// Make sure tx in block sealed at expected time
	require.Equal(t, included0.Time, targetTime)
	require.Equal(t, included0.Number, targetNum)
	require.Equal(t, included1.Time, targetTime)
	require.Equal(t, included1.Number, targetNum)

	// confirm speculatively built exec message by rebuilding after tx inclusion
	_, err = tx0.Result.Eval(t.Ctx())
	require.NoError(t, err)
	execY2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx0.Result, int(logIndexY))(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, execY2, execY)
	_, err = tx1.Result.Eval(t.Ctx())
	require.NoError(t, err)
	execX2, err := txintent.ExecuteIndexed(constants.CrossL2Inbox, &tx1.Result, int(logIndexX))(t.Ctx())
	require.NoError(t, err)
	require.Equal(t, execX2, execX)

	unsafeHeadNumAfterReorg := targetNum - 1
	reorgOutUnsafeAndConsolidateToSafeBothChain(t, actors, actors.ChainA, actors.ChainB, 0, 0, targetNum, targetNum, unsafeHeadNumAfterReorg)
}

// TestExecMsgPointToSelf tests below scenario:
// Execute msg with identifier pointing to the exec msg itself (payload hash cannot be right)
func TestExecMsgPointToSelf(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)
	alice := setupUser(t, is, actors.ChainA, 0)

	actors.ChainA.Sequencer.ActL2EmptyBlock(t)
	assertHeads(t, actors.ChainA, 1, 0, 0, 0)

	// assume exec message pointing to self land in block number 2
	targetTime := actors.ChainA.RollupCfg.Genesis.L2Time + actors.ChainA.RollupCfg.BlockTime*2
	targetNum := uint64(2)
	optsA, _ := DefaultTxOpts(t, alice, actors.ChainA)

	// open blocks
	actors.ChainA.Sequencer.ActL2StartBlock(t)

	// manually construct identifier which makes exec message point to itself
	identifier := suptypes.Identifier{
		Origin:      constants.CrossL2Inbox,
		BlockNumber: targetNum,
		LogIndex:    uint32(0), // tx will emit single ExecutingMessage event to set log index as 0
		Timestamp:   targetTime,
		ChainID:     actors.ChainA.ChainID,
	}
	// cannot construct correct payload hash because payload hash preimage contains payload hash itself
	// we still try to test using dummy payloadHash
	payloadHash := testutils.RandomHash(rng)
	message := suptypes.Message{Identifier: identifier, PayloadHash: payloadHash}

	exec := &txintent.ExecTrigger{Executor: constants.CrossL2Inbox, Msg: message}
	// txintent for executing message pointing to itself
	tx := txintent.NewIntent[*txintent.ExecTrigger, *txintent.InteropOutput](optsA)
	tx.Content.Set(exec)

	included, err := tx.PlannedTx.IncludedBlock.Eval(t.Ctx())
	require.NoError(t, err)
	_, err = tx.PlannedTx.Success.Eval(t.Ctx())
	require.NoError(t, err)

	// Make sure tx in block sealed at expected time
	require.Equal(t, included.Time, targetTime)
	require.Equal(t, included.Number, targetNum)

	// Make batcher happy by advancing at least a single block
	actors.ChainB.Sequencer.ActL2EmptyBlock(t)

	unsafeHeadNumAfterReorg := targetNum - 1
	reorgOutUnsafeAndConsolidateToSafe(t, actors, actors.ChainB, actors.ChainA, 0, 0, 1, targetNum, unsafeHeadNumAfterReorg)
}

// TestInvalidRandomGraph tests below scenario:
// Construct random graphs of messages, with cycles or invalid executing messages
// Test outline:
// 1. Deploy EventLogger per chain.
// 2. Initialize block counts and message count per blocks per chains
// 3. Generate direct acyclic graph, and map each vertices to each messages
// 4. Inject fault
// 5. Sync EL, CL and supervisor to check cross-unsafe halts and rejects bad data
func TestInvalidRandomGraph(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	rng := rand.New(rand.NewSource(1234))

	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	actors.PrepareChainState(t)
	alice := setupUser(t, is, actors.ChainA, 0)
	bob := setupUser(t, is, actors.ChainB, 0)

	// deploy eventLogger for each chain
	actors.ChainA.Sequencer.ActL2StartBlock(t)
	deployOptsA, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainA, 1), actors.ChainA)
	eventLoggerAddressA := DeployEventLogger(t, deployOptsA)
	actors.ChainB.Sequencer.ActL2StartBlock(t)
	deployOptsB, _ := DefaultTxOpts(t, setupUser(t, is, actors.ChainB, 1), actors.ChainB)
	eventLoggerAddressB := DeployEventLogger(t, deployOptsB)

	assertHeads(t, actors.ChainA, 1, 0, 0, 0)
	assertHeads(t, actors.ChainB, 1, 0, 0, 0)

	eventLoggerAddresses := []common.Address{eventLoggerAddressA, eventLoggerAddressB}
	chains := []*dsl.Chain{actors.ChainA, actors.ChainB}
	users := []*userWithKeys{alice, bob}
	nonces := []uint64{0, 0}

	// parameters for building directed acyclic graph
	blockCnt := 10
	L2ChainCnt := 2
	maxTxCntPerBlock := 15
	// execMsgDensity controls the density of exec message txs for overall blocks
	execMsgDensity := 0.01
	require.Greater(t, blockCnt, 1)
	require.Greater(t, L2ChainCnt, 1)
	require.Greater(t, maxTxCntPerBlock, 2)

	// build directed acyclic graph
	// each vertex represents message
	type vertex struct {
		chainIdx  int
		blockNum  int
		msgIdx    int
		vertexIdx int
		init      *txintent.InitTrigger
		exec      *txintent.ExecTrigger
	}
	type edge struct {
		tail *vertex
		head *vertex
	}
	genKey := func(chainIdx, blockNum, msgIdx int) string {
		return fmt.Sprintf("%d-%d-%d", chainIdx, blockNum, msgIdx)
	}

	vertices := map[string]*vertex{}
	keys := []string{}
	msgCnts := make([][]int, L2ChainCnt)
	msgCntsPerTimestamp := []int{}
	vertexIdx := 0

	// Fix message count per block per chain
	for blockNum := range blockCnt {
		msgCntPerTimestamp := 0
		for l2Idx := range L2ChainCnt {
			// every block has at least single message
			msgCnt := 1 + rng.Intn(maxTxCntPerBlock)
			msgCnts[l2Idx] = append(msgCnts[l2Idx], msgCnt)
			for msgIdx := range msgCnt {
				v := vertex{chainIdx: l2Idx, blockNum: blockNum, msgIdx: msgIdx, vertexIdx: vertexIdx}
				key := genKey(l2Idx, blockNum, msgIdx)
				keys = append(keys, key)
				vertices[key] = &v
				vertexIdx += 1
				msgCntPerTimestamp += 1
			}
		}
		msgCntsPerTimestamp = append(msgCntsPerTimestamp, msgCntPerTimestamp)
	}
	vertexCnt := len(vertices)

	// vertices will hold init triggers. When vertex is considered to be an exec message, use them
	for _, vertex := range vertices {
		vertex.init = interop.RandomInitTrigger(rng, eventLoggerAddresses[vertex.chainIdx], rng.Intn(5), rng.Intn(50))
	}

	// build implicit dependencies for intra block messages
	implicitEdges := []edge{}
	for blockNum := range blockCnt {
		for l2Idx := range L2ChainCnt {
			for idx := range msgCnts[l2Idx][blockNum] - 1 {
				tail := vertices[genKey(l2Idx, blockNum, idx+1)]
				head := vertices[genKey(l2Idx, blockNum, idx)]
				implicitEdges = append(implicitEdges,
					edge{
						tail: tail,
						head: head,
					})
			}
		}
	}

	// we start with block number 2 because we passed genesis and block 1 was consumed by deploying EventLogger
	blockNumOffset := uint64(2)

	// build explicit dependencies
	explicitEdges := []edge{}
	// vertices are already topologically sorted, so we are safe to construct graph by choosing indexes as below manner.
	for i := 0; i < vertexCnt; i++ {
		for j := i + 1; j < vertexCnt; j++ {
			tail := vertices[keys[j]] // exec message
			head := vertices[keys[i]] // init message
			// if tail was already considered as an exec message, do not redo
			if tail.exec != nil {
				continue
			}
			// if head was already considered as an exec message, we cannot use it as an init message
			// technically we may override init message by manually generating event:
			// event ExecutingMessage which is emitted by CrossL2Inbox when validateMessage called
			if head.exec != nil {
				// because of this check, there is no exec message that points to exec message
				// although in real world, there could be this kind of scenario where exec depends on exec
				continue
			}
			if rand.Float64() < execMsgDensity {
				initMsgBlockNum := uint64(head.blockNum) + blockNumOffset
				initMsgChain := chains[head.chainIdx]
				initMsgBlockTime := initMsgChain.RollupCfg.Genesis.L2Time + initMsgChain.RollupCfg.BlockTime*initMsgBlockNum
				// speculatively build exec messages without init message execution
				exec, err := interop.ExecTriggerFromInitTrigger(head.init, uint(head.msgIdx), initMsgBlockNum, initMsgBlockTime, initMsgChain.ChainID)
				tail.exec = exec
				require.NoError(t, err)
				explicitEdges = append(explicitEdges, edge{tail: tail, head: head})
			}
		}
	}

	edges := []edge{}
	edges = append(edges, implicitEdges...)
	edges = append(edges, explicitEdges...)

	// Log every dependency
	chainNames := []string{"A", "B"}
	for _, edge := range edges {
		t.Log("edge", fmt.Sprintf("%s%d-%d -> %s%d-%d\n",
			chainNames[edge.tail.chainIdx],
			edge.tail.blockNum,
			edge.tail.msgIdx,
			chainNames[edge.head.chainIdx],
			edge.head.blockNum,
			edge.head.msgIdx,
		))
	}

	base := 0

	faultInjectionIdx := rng.Intn(len(explicitEdges) - 1)
	execMsgIncludedCnt := 0
	// assume that each transaction contains a single message
	for blockIdx := range blockCnt {
		// open block for each chains
		actors.ChainA.Sequencer.ActL2StartBlock(t)
		actors.ChainB.Sequencer.ActL2StartBlock(t)

		msgCntPerTimestamp := msgCntsPerTimestamp[blockIdx]
		intents := []*txintent.IntentTx[txintent.Call, *txintent.InteropOutput]{}
		// safe to iterate like this because vertices are already well sorted and grouped
		for idx := base; idx < base+msgCntPerTimestamp; idx++ {
			vertex := vertices[keys[idx]]
			var trigger txintent.Call
			if vertex.exec != nil {
				exec := vertex.exec
				// inject fault to trigger unsafe reorg
				if execMsgIncludedCnt == faultInjectionIdx {
					exec.Msg.PayloadHash = testutils.RandomHash(rng)
				}
				trigger = exec
				execMsgIncludedCnt += 1
			} else {
				trigger = vertex.init
			}
			nonce := &nonces[vertex.chainIdx]
			user := users[vertex.chainIdx]
			chain := chains[vertex.chainIdx]
			submitIntent(t, trigger, nonce, user, chain, &intents)
		}
		base += msgCntPerTimestamp

		// seal block for each chains
		actors.ChainA.Sequencer.ActL2EndBlock(t)
		actors.ChainB.Sequencer.ActL2EndBlock(t)
	}
	// each vertex is 1-1 to txs
	require.Equal(t, vertexCnt, int(nonces[0]+nonces[1]))

	// propagate unsafe blocks to supervisor
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)
	actors.ChainA.Sequencer.SyncSupervisor(t)
	actors.ChainB.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainB.Sequencer.ActL2PipelineFull(t)

	// supervisor cannot make progress with all unsafe blocks, and cross-unsafe halts
	targetNum := uint64(1 + blockCnt)
	statusA := actors.ChainA.Sequencer.SyncStatus()
	statusB := actors.ChainB.Sequencer.SyncStatus()
	check := targetNum == statusA.CrossUnsafeL2.ID().Number
	check = check && targetNum == statusB.CrossUnsafeL2.ID().Number

	// cross unsafe did not advance
	require.False(t, check)
}
