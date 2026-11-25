package proofs

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"

	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
)

// postExecLagoonOffset schedules Lagoon so that blocks 1 to 3 come before it.
const postExecLagoonOffset = uint64(4)

// TestPostExecBatchLagoonGate checks that derivation gates the PostExec (0x7D) transaction on
// each block's own timestamp. Each case batches one span from before Lagoon to after it: a PostExec
// transaction in a block before Lagoon drops the span, and one in a block after it is accepted.
func TestPostExecBatchLagoonGate(gt *testing.T) {
	genesisFork := helpers.NewForkMatrix(helpers.Karst)
	matrix := helpers.NewMatrix[bool]()
	matrix.AddTestCase("PreLagoon", true, genesisFork, runPostExecLagoonGate, helpers.ExpectNoError())
	matrix.AddDefaultTestCasesWithName("PostLagoon", false, genesisFork, runPostExecLagoonGate)
	matrix.Run(gt)
}

// runPostExecLagoonGate builds a block with a contract creation before Lagoon and another after
// the Lagoon activation block, and batches them in one span with a PostExec transaction appended to
// the first block if testCfg.Custom is set, else to the second.
func runPostExecLagoonGate(gt *testing.T, testCfg *helpers.TestCfg[bool]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	preLagoon := testCfg.Custom
	batcherCfg := helpers.NewBatcherCfg(func(b *actionsHelpers.BatcherCfg) {
		b.DataAvailabilityType = batcherFlags.CalldataType
		b.ForceSubmitSpanBatch = true
	})
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), batcherCfg, func(dc *genesis.DeployConfig) {
		dc.L1PragueTimeOffset = ptr(hexutil.Uint64(0))
		dc.SetForkTimeOffset(forks.Lagoon, ptr(postExecLagoonOffset))
	})
	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL1HeadSignal(t)

	first, _ := buildCreationBlock(t, env)
	require.False(t, env.Sd.RollupCfg.IsLagoon(first.Time), "block %d must come before Lagoon", first.Number)
	activation := env.Sequencer.ActBuildL2ToFork(t, forks.Lagoon)
	require.True(t, env.Sd.RollupCfg.IsLagoonActivationBlock(activation.Time), "block %d must activate Lagoon", activation.Number)
	last, creation := buildCreationBlock(t, env)
	sequenced := txReceipt(t, env, creation, last.Hash)

	target := last
	if preLagoon {
		target = first
	}
	for env.Batcher.L2BufferedBlock.Number < last.Number {
		env.Batcher.ActL2BatchBuffer(t, actionsHelpers.WithBlockModifier(func(payload *eth.ExecutionPayload) *eth.ExecutionPayload {
			if uint64(payload.BlockNumber) != target.Number {
				return payload
			}
			return appendPostExecTx(t, target.Number)(payload)
		}))
	}

	if preLagoon {
		// BatchMineAndSync requires the safe head to reach the unsafe head, which it does not when
		// the span is dropped.
		env.BatchAndMine(t)
		env.Sequencer.ActL1HeadSignal(t)
		env.Sequencer.ActL2PipelineFull(t)
		require.Len(t, env.Logs.FindLogs(testlog.NewMessageFilter("sequencers may not embed any PostExec transactions before SDM")), 1)
		require.Equal(t, env.Sd.RollupCfg.Genesis.L2.Number, env.Sequencer.L2Safe().Number, "derivation must drop the span")

		// The L1 chain up to its head holds no other batch for the block, and the sequencing window
		// has not expired, so the honest claim for the block is the invalid transition.
		env.RunFaultProofProgram(t, first.Number, testCfg.CheckResult, helpers.WithL2Claim(eth.InvalidTransitionHash))
		// The default claim commits to the sequencer's block, which a program that kept the creation
		// but stripped the PostExec transaction would reach.
		env.RunFaultProofProgram(t, first.Number, helpers.ExpectError(helpers.ErrClaimNotValid))
		return
	}

	safe := env.BatchMineAndSync(t)
	require.Equal(t, last.Number, safe.Number, "derivation must accept every block of the span")
	require.Equal(t, sequenced.GasUsed-1, txReceipt(t, env, creation, safe.Hash).GasUsed, "the derived block must refund 1 gas to the creation")
	envelope, err := env.Engine.SourceClient().PayloadByHash(t.Ctx(), safe.Hash)
	require.NoError(t, err)
	txs := envelope.ExecutionPayload.Transactions
	require.Len(t, txs, 3, "the derived block must hold the L1 info deposit, the creation and the PostExec transaction")
	require.True(t, optypes.IsPostExecTxBytes(txs[2]), "the PostExec transaction must come last")

	env.RunFaultProofProgram(t, last.Number, testCfg.CheckResult, testCfg.InputParams...)
}

// buildCreationBlock builds a block whose only user transaction is a contract creation by Alice
// with init code 0x00. It uses more gas than its EIP-7623 floor, so an SDM producer could refund
// it. The L1 info deposit comes first, so the creation has index 1.
func buildCreationBlock(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv) (eth.L2BlockRef, *types.Transaction) {
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(nil)(t)
	env.Alice.L2.ActSetTxCalldata([]byte{byte(vm.STOP)})(t)
	tx := env.Alice.L2.ActMakeTx(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
	return env.Sequencer.ActL2EndBlock(t), tx
}

// txReceipt returns the receipt of tx, which must be in the block with hash blockHash.
func txReceipt(t actionsHelpers.Testing, env *helpers.L2FaultProofEnv, tx *types.Transaction, blockHash common.Hash) *optypes.Receipt {
	r, err := env.Engine.SourceClient().TransactionReceipt(t.Ctx(), tx.Hash())
	require.NoError(t, err)
	require.Equal(t, blockHash, r.BlockHash, "the transaction must be in block %s", blockHash)
	return r
}

// appendPostExecTx appends a PostExec transaction for block number that refunds 1 gas to the
// transaction at index 1. The refund is valid from Lagoon on.
func appendPostExecTx(t actionsHelpers.Testing, number uint64) func(*eth.ExecutionPayload) *eth.ExecutionPayload {
	data, err := rlp.EncodeToBytes(&optypes.PostExecPayload{
		Version:          optypes.PostExecPayloadVersion,
		BlockNumber:      number,
		GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 1}},
	})
	require.NoError(t, err)
	tx, err := (&optypes.PostExecTx{Data: data}).MarshalBinary()
	require.NoError(t, err)
	return func(payload *eth.ExecutionPayload) *eth.ExecutionPayload {
		payload.Transactions = append(payload.Transactions, tx)
		return payload
	}
}
