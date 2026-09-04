package enginetest

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// TestSequencerBuildsOnRethEngine is the GoSwitch R3 foundation gate: it runs the REAL op-node
// sequencer (helpers.SetupSequencerTest, unmodified) against the out-of-process op-reth-test-engine
// selected by OP_E2E_ACTIONS_EL, and drives the standard action-test block-building loop
// (ActL2StartBlock -> ActL2IncludeTx -> ActL2EndBlock) over the socket. It proves the switch's core
// claim: op-node's engine controller can build, seal, import, and advance L2 blocks on the
// subprocess engine, and the parking-buffer reframing of ActL2IncludeTx includes a user tx.
//
// It reads the chain back through the backend-agnostic L2Engine.LatestHeader helper (the L2Chain()
// replacement) and the standard eth client — the same surface the full switch rewrites the suite
// onto.
func TestSequencerBuildsOnRethEngine(gt *testing.T) {
	// Holocene keeps the fork surface simple (post-Cancun eip1559Params extraData, no Isthmus
	// withdrawals-root / requests-hash); fork-fan coverage is a later round.
	gt.Setenv("OP_E2E_USE_HOLOCENE", "true")
	gt.Setenv(helpers.ELSelectorEnv, "reth-test-engine")

	// SubTest (not NewDefaultTesting) avoids t.Parallel, which t.Setenv forbids.
	t := helpers.SubTest(gt)
	dp := e2eutils.MakeDeployParams(t, helpers.DefaultRollupTestParams())
	sd := e2eutils.Setup(t, dp, helpers.DefaultAlloc)
	logger := testlog.Logger(t, gethlog.LevelInfo)

	miner, engine, sequencer := helpers.SetupSequencerTest(t, sd, logger)
	defer func() { _ = engine.Close() }()
	miner.ActL1SetFeeRecipient(common.Address{'A'})
	sequencer.ActL2PipelineFull(t)

	// Genesis is the starting head, read back over the socket via the L2Chain() replacement.
	require.EqualValues(t, 0, bigs.Uint64Strict(engine.LatestHeader(t).Number))

	// Two deposit-only (empty) blocks: the sequencer opens a payload (FCU-with-attrs), the engine
	// seals it (getPayload), op-node imports it (newPayload) and advances the head (FCU).
	sequencer.ActL2EmptyBlock(t)
	sequencer.ActL2EmptyBlock(t)
	require.EqualValues(t, 2, bigs.Uint64Strict(engine.LatestHeader(t).Number))

	// A third block that includes a user transaction, exercising the parking buffer:
	// EthClient().SendTransaction parks it; ActL2IncludeTx drains it via optest_includeNextTx.
	signer := types.LatestSigner(sd.L2Cfg.Config)
	cl := engine.EthClient()
	nonce, err := cl.PendingNonceAt(t.Ctx(), dp.Addresses.Alice)
	require.NoError(t, err)
	baseFee := engine.LatestHeader(t).BaseFee
	tx := types.MustSignNewTx(dp.Secrets.Alice, signer, &types.DynamicFeeTx{
		ChainID:   sd.L2Cfg.Config.ChainID,
		Nonce:     nonce,
		GasTipCap: big.NewInt(2 * params.GWei),
		GasFeeCap: new(big.Int).Add(baseFee, big.NewInt(2*params.GWei)),
		Gas:       params.TxGas,
		To:        &dp.Addresses.Bob,
		Value:     e2eutils.Ether(1),
	})
	bobBefore, err := cl.BalanceAt(t.Ctx(), dp.Addresses.Bob, nil)
	require.NoError(t, err)

	require.NoError(t, cl.SendTransaction(t.Ctx(), tx))
	sequencer.ActL2StartBlock(t)
	engine.ActL2IncludeTx(dp.Addresses.Alice)(t)
	sequencer.ActL2EndBlock(t)

	head := engine.LatestHeader(t)
	require.EqualValues(t, 3, bigs.Uint64Strict(head.Number))

	// The user tx was executed: Alice's nonce advanced and Bob received the ether.
	aliceNonce, err := cl.NonceAt(t.Ctx(), dp.Addresses.Alice, nil)
	require.NoError(t, err)
	require.EqualValues(t, nonce+1, aliceNonce, "Alice's tx was included")

	bobAfter, err := cl.BalanceAt(t.Ctx(), dp.Addresses.Bob, nil)
	require.NoError(t, err)
	require.Equal(t, e2eutils.Ether(1), new(big.Int).Sub(bobAfter, bobBefore), "Bob received 1 ether")

	// The sealed block carries the L1-info deposit plus Alice's transaction.
	block, err := cl.BlockByNumber(t.Ctx(), head.Number)
	require.NoError(t, err)
	require.Len(t, block.Transactions(), 2, "L1-info deposit + Alice's user tx")
	require.Equal(t, uint8(types.DepositTxType), block.Transactions()[0].Type(), "block leads with the L1-info deposit")

	// op-node's real receipt fetcher validates the receipts-root by re-encoding the fetched
	// receipts (the deposit's nonce/version and the user receipt) and comparing to the header — the
	// consensus-parity gate for the OP-enriched receipt RPC.
	src := engine.SourceClient(t, 10)
	info, receipts, err := src.FetchReceipts(t.Ctx(), head.Hash())
	require.NoError(t, err, "FetchReceipts must validate the receipts-root against the header")
	require.Equal(t, head.Hash(), info.Hash())
	require.Len(t, receipts, 2)
	require.Equal(t, uint8(types.DepositTxType), receipts[0].Type, "first receipt is the L1-info deposit")
	require.Equal(t, types.ReceiptStatusSuccessful, receipts[1].Status, "Alice's tx succeeded")
	require.EqualValues(t, params.TxGas, receipts[1].GasUsed, "Alice's transfer used the intrinsic gas")

	// The direct eth_getTransactionReceipt path also returns Alice's receipt.
	rcpt, err := cl.TransactionReceipt(t.Ctx(), tx.Hash())
	require.NoError(t, err)
	require.Equal(t, types.ReceiptStatusSuccessful, rcpt.Status)
	require.Equal(t, head.Hash(), rcpt.BlockHash)
}
