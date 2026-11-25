package proofs

import (
	"math/big"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
	"github.com/ethereum-optimism/optimism/op-service/testutils/parityreport"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// The tests in this file are the proof layer of the derivation parity suite. Each one crafts L1
// batch data on which op-node and kona derive different safe chains. op-node derives the chain in
// process, and kona-client must then prove op-node's safe head.
//
// Every test runs two cases on each fork:
//   - crafted: the crafted L1 data. kona-client rejects op-node's honest claim and accepts the
//     claim of the chain kona derives, which pins today's difference. kona-host rejects any claim
//     when the client fails, so the second run is what shows that kona derives a different chain.
//     When either client is fixed, this case fails: flip it to expect the claim to pass, and
//     record it as agreeing.
//   - control: the same scenario without the crafted change. kona-client accepts op-node's claim,
//     which shows that the difference comes from the crafted change and not from the setup.
//
// Both cases claim the block at which the clients diverge, and that block carries a user
// transaction, so that a deposits-only block cannot hide the difference. With
// DERIVATION_PARITY_REPORT_DIR set, each passing subtest records its outcome for the parity report
// (op-service/testutils/parityreport).

// divergenceHoloceneWithFork describes the one difference only this layer shows. The others come
// from parityreport, which every layer takes them from.
const divergenceHoloceneWithFork = "Holocene activating in the same L1 block as an earlier fork: kona switches to the Holocene derivation stages, op-node keeps the pre-Holocene stages"

type parityCase struct {
	// crafted selects the crafted L1 data instead of the control.
	crafted bool
	// forks names the forks of the case in its parity record when the matrix fork does not, such
	// as when the case schedules forks after genesis.
	forks string
}

// parityForks spans Holocene, which changes how op-node handles an invalid batch, and starts after
// Fjord, so that every case can use span batches and brotli.
var parityForks = helpers.NewForkMatrix(helpers.Granite, helpers.Holocene, helpers.Isthmus, helpers.LatestFork)

func runParityMatrix(gt *testing.T, forks helpers.ForkMatrix, forksLabel string, run helpers.RunTest[parityCase]) {
	matrix := helpers.NewMatrix[parityCase]()
	matrix.AddTestCase("HonestClaim-Crafted", parityCase{crafted: true, forks: forksLabel}, forks, run, helpers.ExpectError(helpers.ErrClaimNotValid))
	matrix.AddTestCase("HonestClaim-Control", parityCase{forks: forksLabel}, forks, run, helpers.ExpectNoError())
	matrix.Run(gt)
}

// recordParity records the outcome of the subtest for the parity report once the subtest has
// passed: the crafted case diverges with the given difference, and the control agrees.
func recordParity(gt *testing.T, testCfg *helpers.TestCfg[parityCase], divergence string) {
	record := parityreport.Record{
		Layer: parityreport.LayerProof,
		Group: strings.SplitN(gt.Name(), "/", 2)[0],
		Case:  "control",
		Forks: testCfg.Hardfork.Name,
		Agree: true,
	}
	if testCfg.Custom.forks != "" {
		record.Forks = testCfg.Custom.forks
	}
	if testCfg.Custom.crafted {
		record.Case = "crafted"
		record.Agree = false
		record.Divergence = divergence
	}
	gt.Cleanup(func() {
		if !gt.Failed() && !gt.Skipped() {
			parityreport.Write(gt, record)
		}
	})
}

// requireKonaCannotDerive checks that kona-client accepts the claim that the chain cannot reach
// the block from the L1 data: kona drops the batch that op-node derives the block from.
func requireKonaCannotDerive(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, l2BlockNum uint64) {
	env.RunFaultProofProgram(t, l2BlockNum, helpers.ExpectNoError(), helpers.WithL2Claim(eth.InvalidTransitionHash))
}

// buildUserBlock has the sequencer build the next L2 block with one transaction from Alice that
// carries calldata, and returns the block.
func buildUserBlock(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, calldata []byte) *eth.ExecutionPayload {
	return buildUserBlockEnvelope(t, env, calldata).ExecutionPayload
}

func buildUserBlockEnvelope(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, calldata []byte) *eth.ExecutionPayloadEnvelope {
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)(t)
	if len(calldata) > 0 {
		env.Alice.L2.ActSetTxCalldata(calldata)(t)
	}
	env.Alice.L2.ActMakeTx(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
	ref := env.Sequencer.ActL2EndBlock(t)
	envelope, err := env.Engine.EngineClient(t, env.Sd.RollupCfg).PayloadByNumber(t.Ctx(), ref.Number)
	require.NoError(t, err)
	require.Len(t, actionsHelpers.PayloadUserTxIndexes(envelope.ExecutionPayload), 1)
	return envelope
}

// submitChannel posts the channel in a new L1 block and has op-node derive from it.
func submitChannel(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, channel *actionsHelpers.CraftedChannel, data []byte) {
	env.Batcher.ActL2BatchSubmitRaw(t, channel.Frame(t, data))
	env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)
}

// requireSafeHead checks that op-node's safe head is the sequencer's block.
func requireSafeHead(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, block *eth.ExecutionPayload) {
	safeHead := env.Engine.SafeHeader(t)
	require.Equal(t, uint64(block.BlockNumber), bigs.Uint64Strict(safeHead.Number), "safe head number")
	require.Equal(t, block.BlockHash, safeHead.Hash(), "the safe head is the sequencer's block")
}

// requireDepositsOnly checks that op-node's safe block at the height of the sequencer's block
// replaces it with a block that holds only the L1 info deposit.
func requireDepositsOnly(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, block *eth.ExecutionPayload) {
	safe, err := env.Engine.EthClient().HeaderByNumber(t.Ctx(), new(big.Int).SetUint64(uint64(block.BlockNumber)))
	require.NoError(t, err)
	require.NotEqual(t, block.BlockHash, safe.Hash(), "the sequencer's block is replaced")
	deposits, userTxs := env.Engine.BlockTxs(t, safe.Hash())
	require.Empty(t, userTxs, "the replacement block has no user transactions")
	require.Len(t, deposits, 1, "the replacement block holds only the L1 info deposit")
}

// TestDerivationParityUnderivableSpanBatch posts a channel whose first batch is a span batch that
// decodes but cannot be derived, because its one transaction is typed as a deposit, followed by a
// valid batch for the same block. Both clients reject the first batch. op-node then reads the next
// batch of the channel and derives the block from it. kona treats the failure as the end of the
// channel and drops the valid batch with it, so it cannot derive the block that op-node claims.
// A batch that fails to decode ends the channel in both clients; only a span batch that decodes
// and then fails span batch derivation shows the difference. The control posts the channel without
// that span batch.
func TestDerivationParityUnderivableSpanBatch(gt *testing.T) {
	runParityMatrix(gt, parityForks, "", runUnderivableSpanBatchTest)
}

func runUnderivableSpanBatchTest(gt *testing.T, testCfg *helpers.TestCfg[parityCase]) {
	recordParity(gt, testCfg, parityreport.DivergenceSpanBatchDerivationFailure)
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())
	block := buildUserBlock(t, env, nil)

	channel := actionsHelpers.NewCraftedChannel(t)
	if testCfg.Custom.crafted {
		channel.AddDepositTypedSpanBatch(t, env.Sd.RollupCfg, block)
	}
	channel.AddSpanBatch(t, env.Sd.RollupCfg, block)
	submitChannel(t, env, channel, channel.Zlib(t))

	// op-node derives the block from the valid batch in both cases.
	requireSafeHead(t, env, block)

	env.RunFaultProofProgram(t, uint64(block.BlockNumber), testCfg.CheckResult, testCfg.InputParams...)
	if testCfg.Custom.crafted {
		requireKonaCannotDerive(t, env, uint64(block.BlockNumber))
	}
}

// TestDerivationParitySpanBatchFee posts a span batch in which 2^128 is added to the max fee per
// gas of the block's one transaction, without signing the transaction again. kona rebuilds the
// transaction from the low 128 bits of the fee, which restores the signed fee, so it derives the
// sequencer's block. op-node rebuilds the transaction with the full fee, and the execution engine,
// which holds a fee in 128 bits, rejects the block's attributes, so op-node's block holds only its
// deposits: from Holocene, as the deposits-only replacement of the batch's block, and before
// Holocene, once the sequencing window of the dropped batch expires. The control posts the span
// batch with the signed fee.
func TestDerivationParitySpanBatchFee(gt *testing.T) {
	runParityMatrix(gt, parityForks, "", runSpanBatchFeeTest)
}

func runSpanBatchFeeTest(gt *testing.T, testCfg *helpers.TestCfg[parityCase]) {
	recordParity(gt, testCfg, parityreport.DivergenceSpanBatchFee)
	t := actionsHelpers.NewDefaultTesting(gt)
	// A short sequencing window lets the test expire it before Holocene.
	tp := helpers.NewTestParams(func(p *e2eutils.TestParams) { p.SequencerWindowSize = 4 })
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())
	envelope := buildUserBlockEnvelope(t, env, nil)
	block := envelope.ExecutionPayload

	batched := block
	if testCfg.Custom.crafted {
		batched = withFeeAbove128Bits(t, block)
	}
	channel := actionsHelpers.NewCraftedChannel(t)
	channel.AddSpanBatch(t, env.Sd.RollupCfg, batched)
	submitChannel(t, env, channel, channel.Zlib(t))

	holocene := env.Sd.RollupCfg.IsHolocene(uint64(block.Timestamp))
	if !holocene {
		if testCfg.Custom.crafted {
			// Before Holocene, op-node drops a batch whose block the engine rejects.
			require.Zero(t, bigs.Uint64Strict(env.Engine.SafeHeader(t).Number), "op-node drops the batch")
		}
		// Expire the batch's sequencing window, so that op-node fills the block's height with a
		// deposits-only block when it has no valid batch for it.
		for range tp.SequencerWindowSize {
			env.Miner.ActEmptyBlock(t)
		}
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)
		require.GreaterOrEqual(t, bigs.Uint64Strict(env.Engine.SafeHeader(t).Number), uint64(block.BlockNumber))
	}

	claimParams := testCfg.InputParams
	if testCfg.Custom.crafted {
		requireDepositsOnly(t, env, block)
		// kona derives the sequencer's block, and kona-host reads the state after it from the
		// engine to compute its output root. op-node's reorg took that block out of the engine,
		// so fix op-node's honest claim first, then make the sequencer's block the engine's head
		// again.
		claimParams = append(slices.Clone(claimParams), honestClaim(t, env, uint64(block.BlockNumber)))
		restoreBlock(t, env, envelope)
	} else if holocene {
		requireSafeHead(t, env, block)
	} else {
		canonical, err := env.Engine.EthClient().HeaderByNumber(t.Ctx(), new(big.Int).SetUint64(uint64(block.BlockNumber)))
		require.NoError(t, err)
		require.Equal(t, block.BlockHash, canonical.Hash(), "op-node keeps the sequencer's block")
	}

	env.RunFaultProofProgram(t, uint64(block.BlockNumber), testCfg.CheckResult, claimParams...)
	if testCfg.Custom.crafted {
		// The engine's chain is the sequencer's again, so the default claim is kona's.
		env.RunFaultProofProgram(t, uint64(block.BlockNumber), helpers.ExpectNoError())
	}
}

// honestClaim returns op-node's claim for the L2 block, as RunFaultProofProgram builds it from the
// engine's current chain.
func honestClaim(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, l2BlockNum uint64) helpers.FixtureInputParam {
	var inputs helpers.FixtureInputs
	helpers.WithSuperDefaults(t, l2BlockNum, env.Sequencer.L2Verifier, env.Engine)(&inputs)
	return helpers.WithL2Claim(inputs.L2Claim)
}

// restoreBlock inserts the block into the engine again and makes it the head of the engine's chain.
func restoreBlock(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, envelope *eth.ExecutionPayloadEnvelope) {
	engine := env.Engine.EngineClient(t, env.Sd.RollupCfg)
	block := envelope.ExecutionPayload
	status, err := engine.NewPayload(t.Ctx(), block, envelope.ParentBeaconBlockRoot)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, status.Status)
	result, err := engine.ForkchoiceUpdate(t.Ctx(), &eth.ForkchoiceState{
		HeadBlockHash:      block.BlockHash,
		SafeBlockHash:      block.ParentHash,
		FinalizedBlockHash: block.ParentHash,
	}, nil)
	require.NoError(t, err)
	require.Equal(t, eth.ExecutionValid, result.PayloadStatus.Status)
}

// withFeeAbove128Bits returns a copy of the block in which 2^128 is added to the max fee per gas
// of the block's user transaction, which keeps its signature.
func withFeeAbove128Bits(t actionsHelpers.Testing, block *eth.ExecutionPayload) *eth.ExecutionPayload {
	changed := *block
	changed.Transactions = slices.Clone(block.Transactions)
	i := actionsHelpers.PayloadUserTxIndexes(&changed)[0]
	tx := actionsHelpers.DecodePayloadTx(t, &changed, i)
	require.Equal(t, uint8(types.DynamicFeeTxType), tx.Type())
	v, r, s := tx.RawSignatureValues()
	changed.Transactions[i] = actionsHelpers.EncodeTx(t, types.NewTx(&types.DynamicFeeTx{
		ChainID:    tx.ChainId(),
		Nonce:      tx.Nonce(),
		GasTipCap:  tx.GasTipCap(),
		GasFeeCap:  new(big.Int).Add(tx.GasFeeCap(), new(big.Int).Lsh(big.NewInt(1), 128)),
		Gas:        tx.Gas(),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
		V:          v,
		R:          r,
		S:          s,
	}))
	return &changed
}

// brotliBreakAt is where TestDerivationParityBrotliFormatError breaks the compressed channel.
// op-node reads a channel through a 4096-byte buffer, and its brotli decoder hands out what it has
// decoded whenever it runs out of input, so op-node keeps the batches it decoded from its first
// read. kona decompresses the whole channel in one call, which hands out nothing when the stream
// fails before its end. The clients therefore differ only when the stream breaks past op-node's
// first read, after a batch that it holds in full.
const brotliBreakAt = 4200

// TestDerivationParityBrotliFormatError posts a brotli channel with a batch for each of two
// blocks, whose compressed stream has a format error in the second batch, past op-node's first
// read of the channel. op-node keeps the first batch and derives its block. kona discards the
// whole channel, so it cannot derive the block that op-node claims. The second block's
// transaction carries incompressible calldata, so that its batch runs past op-node's first read.
// The control posts the channel without the format error.
func TestDerivationParityBrotliFormatError(gt *testing.T) {
	runParityMatrix(gt, parityForks, "", runBrotliFormatErrorTest)
}

func runBrotliFormatErrorTest(gt *testing.T, testCfg *helpers.TestCfg[parityCase]) {
	recordParity(gt, testCfg, parityreport.DivergenceBrotliFormatError)
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())
	first := buildUserBlock(t, env, nil)
	second := buildUserBlock(t, env, testutils.RandomData(rand.New(rand.NewSource(1)), 6000))

	channel := actionsHelpers.NewCraftedChannel(t)
	channel.AddSingularBatch(t, env.Sd.RollupCfg, first)
	channel.AddSingularBatch(t, env.Sd.RollupCfg, second)
	// The control posts the same stream without the format error.
	if testCfg.Custom.crafted {
		submitChannel(t, env, channel, channel.BrotliWithFormatError(t, brotliBreakAt))
		// op-node derives the first block and loses the second.
		requireSafeHead(t, env, first)
	} else {
		submitChannel(t, env, channel, channel.BrotliFlushedTo(t, brotliBreakAt))
		requireSafeHead(t, env, second)
	}

	env.RunFaultProofProgram(t, uint64(first.BlockNumber), testCfg.CheckResult, testCfg.InputParams...)
	if testCfg.Custom.crafted {
		requireKonaCannotDerive(t, env, uint64(first.BlockNumber))
	}
}

// TestDerivationParityHoloceneWithEarlierFork schedules Granite and Holocene after genesis, so
// that the pipeline's L1 origin crosses both activations in one step. op-node switches its
// derivation stages only for the first fork that the step activates, so it keeps deriving with
// the pre-Holocene batch queue and channel bank. kona checks Holocene on every step and switches to
// the Holocene stages.
//
// After the activation, the test posts a channel with an invalid batch for a block, whose parent
// hash is wrong, followed by a valid batch for the same block. The pre-Holocene batch queue drops
// the invalid batch and derives the block from the next one. The Holocene batch stage drops the
// invalid batch and flushes the rest of its channel, so kona cannot derive the block that op-node
// claims. The control posts the channel without the invalid batch.
func TestDerivationParityHoloceneWithEarlierFork(gt *testing.T) {
	runParityMatrix(gt, helpers.NewForkMatrix(helpers.Fjord), "granite and holocene after fjord, in one L1 block", runHoloceneWithEarlierForkTest)
}

func runHoloceneWithEarlierForkTest(gt *testing.T, testCfg *helpers.TestCfg[parityCase]) {
	recordParity(gt, testCfg, divergenceHoloceneWithFork)
	t := actionsHelpers.NewDefaultTesting(gt)
	// Granite activates at L2 block 1 and Holocene at L2 block 2, both before L1 block 1.
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg(), func(dc *genesis.DeployConfig) {
		granite, holocene := dc.L2BlockTime, 2*dc.L2BlockTime
		dc.SetForkTimeOffset(forks.Granite, &granite)
		dc.SetForkTimeOffset(forks.Holocene, &holocene)
	})
	cfg := env.Sd.RollupCfg

	// Blocks 1 and 2 activate Granite and Holocene. Block 3 carries the user transaction.
	env.Sequencer.ActL2EmptyBlock(t)
	env.Sequencer.ActL2EmptyBlock(t)
	block := buildUserBlock(t, env, nil)
	require.True(t, cfg.IsHolocene(uint64(block.Timestamp)))
	parent, err := env.Engine.EngineClient(t, cfg).PayloadByNumber(t.Ctx(), uint64(block.BlockNumber)-1)
	require.NoError(t, err)
	grandparent, err := env.Engine.EngineClient(t, cfg).PayloadByNumber(t.Ctx(), uint64(block.BlockNumber)-2)
	require.NoError(t, err)

	// Post the activation blocks in L1 block 1, which moves op-node's pipeline across both
	// activations in one step.
	activation := actionsHelpers.NewCraftedChannel(t)
	activation.AddSpanBatch(t, cfg, grandparent.ExecutionPayload, parent.ExecutionPayload)
	submitChannel(t, env, activation, activation.Zlib(t))
	l1Genesis := env.Miner.L1Chain().GetHeaderByNumber(0)
	l1Block1 := env.Miner.L1Chain().GetHeaderByNumber(1)
	require.Equal(t, forks.Granite, cfg.IsActivationBlock(l1Genesis.Time, l1Block1.Time), "op-node's stage switch names only Granite")
	require.False(t, cfg.IsHolocene(l1Genesis.Time))
	require.True(t, cfg.IsHolocene(l1Block1.Time))
	requireSafeHead(t, env, parent.ExecutionPayload)

	channel := actionsHelpers.NewCraftedChannel(t)
	if testCfg.Custom.crafted {
		invalid := *block
		invalid.ParentHash = common.Hash{0x01}
		channel.AddSingularBatch(t, cfg, &invalid)
	}
	channel.AddSingularBatch(t, cfg, block)
	submitChannel(t, env, channel, channel.Zlib(t))

	// op-node derives the block from the valid batch in both cases.
	requireSafeHead(t, env, block)

	env.RunFaultProofProgram(t, uint64(block.BlockNumber), testCfg.CheckResult, testCfg.InputParams...)
	if testCfg.Custom.crafted {
		requireKonaCannotDerive(t, env, uint64(block.BlockNumber))
	}
}
