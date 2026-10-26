package proofs

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/rust/kona/tests/proofs/helpers"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// bareDeposit selects the case that submits the deposit without its 0x7E type byte.
const bareDeposit = true

// runDepositInBatchTest submits a singular batch that carries a deposit minting to an unfunded
// account and checks that neither derivation nor the deposits-only replacement ever applies it,
// and that kona-client agrees with the resulting chain. With bareDeposit, the deposit is submitted
// as a bare body without its 0x7E type byte, which passes the leading-byte batch gate and must be
// rejected by the execution engine instead; otherwise it is canonically encoded and the batch gate
// drops the whole batch.
//
// The batch goes through a GarbageChannelOut with a transaction modifier: the batcher's regular
// path filters deposit-typed transactions out of the batch, and its block modifier hook works on
// *types.Block, which cannot hold a bare body.
func runDepositInBatchTest(gt *testing.T, testCfg *helpers.TestCfg[bool]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())

	// Build L2 block 1 with one user tx.
	env.Alice.L2.ActResetTxOpts(t)
	env.Alice.L2.ActSetTxToAddr(&env.Dp.Addresses.Bob)
	env.Alice.L2.ActMakeTx(t)
	env.Sequencer.ActL2StartBlock(t)
	env.Engine.ActL2IncludeTx(env.Alice.Address())(t)
	sequencerBlock := env.Sequencer.ActL2EndBlock(t)
	// Read the sequencer's L1 info deposit before derivation replaces block 1, so the check does
	// not depend on the engine retaining the reorged-out block.
	_, sequencerL1InfoTx, err := env.Engine.SourceClient().InfoAndFirstDepositByHash(t.Ctx(), sequencerBlock.Hash)
	require.NoError(t, err)

	beneficiary := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	deposit := &optypes.DepositTx{
		SourceHash: common.HexToHash("0x01"),
		From:       beneficiary,
		To:         &beneficiary,
		Mint:       big.NewInt(1e18),
		Value:      big.NewInt(0),
		Gas:        21_000,
	}
	depositBytes, err := deposit.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, byte(optypes.DepositTxType), depositBytes[0])
	batchTx := eth.Data(depositBytes)
	if testCfg.Custom == bareDeposit {
		batchTx = depositBytes[1:]
		require.Greater(t, batchTx[0], byte(0x7f), "a bare body must not start with a type byte")
	}

	// Submit block 1 in a singular batch with the user tx replaced by the crafted deposit.
	channel, err := actionsHelpers.NewGarbageChannelOut(&actionsHelpers.GarbageChannelCfg{
		TxModifier: func([]eth.Data) []eth.Data { return []eth.Data{batchTx} },
	})
	require.NoError(t, err)
	env.Batcher.L2ChannelOut = channel
	env.Batcher.ActAddBlockByNumber(t, 1)
	env.Batcher.ActL2ChannelClose(t)
	env.Batcher.ActL2BatchSubmit(t)
	env.Miner.ActL1StartBlock(helpers.L1BlockTime)(t)
	env.Miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	env.Miner.ActL1EndBlock(t)
	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	l2SafeHead := env.Engine.SafeHeader(t)
	if testCfg.Custom == bareDeposit && env.Sd.RollupCfg.IsHolocene(l2SafeHead.Time) {
		// The engine rejects the batch transaction and the block is replaced by its deposits.
		require.Equal(t, uint64(1), bigs.Uint64Strict(l2SafeHead.Number))
		deposits, userTxs := helpers.BlockTxs(t, env.Engine, l2SafeHead.Hash())
		require.Empty(t, userTxs, "only the L1 info deposit remains")
		require.Len(t, deposits, 1, "only the L1 info deposit remains")
		l1InfoTx := deposits[0]
		require.Equal(t, predeploys.L1BlockAddr, *l1InfoTx.To)
		require.NotEqual(t, sequencerBlock.Hash, l2SafeHead.Hash(), "the sequencer's block is replaced")
		require.Equal(t, sequencerL1InfoTx.Hash(), l1InfoTx.Hash(), "same L1 info deposit as the sequencer's block")
	} else {
		// The batch is dropped, by the gate or by the engine before Holocene, so the safe head
		// only advances once a valid batch is submitted.
		require.Equal(t, uint64(0), bigs.Uint64Strict(l2SafeHead.Number))
		env.Batcher.Reset()
		require.Equal(t, uint64(1), env.BatchMineAndSync(t).Number)
		l2SafeHead = env.Engine.SafeHeader(t)
	}

	// The crafted deposit was never applied.
	balance, err := env.Engine.EthClient().BalanceAt(t.Ctx(), beneficiary, nil)
	require.NoError(t, err)
	require.Zero(t, balance.Sign())

	env.RunFaultProofProgramFromGenesis(t, bigs.Uint64Strict(l2SafeHead.Number), testCfg.CheckResult, testCfg.InputParams...)
}

func TestDepositInBatch(gt *testing.T) {
	forks := helpers.NewForkMatrix(helpers.Granite, helpers.Holocene, helpers.Isthmus, helpers.LatestFork)
	matrix := helpers.NewMatrix[bool]()
	matrix.AddDefaultTestCasesWithName("BareDeposit", bareDeposit, forks, runDepositInBatchTest)
	matrix.AddDefaultTestCasesWithName("TaggedDeposit", !bareDeposit, forks, runDepositInBatchTest)
	matrix.Run(gt)
}
