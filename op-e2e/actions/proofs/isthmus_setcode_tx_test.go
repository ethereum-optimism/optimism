package proofs_test

import (
	"bytes"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

func Test_ProgramAction_SetCodeTx(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddDefaultTestCases(
		nil,
		helpers.LatestForkOnly,
		runSetCodeTxTypeTest,
	)
}

func runSetCodeTxTypeTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	var (
		aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
		bb = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
	)

	t := actionsHelpers.NewDefaultTesting(gt)

	// hardcoded because it's not available until after we need it
	bobAddr := common.HexToAddress("0x14dC79964da2C08b23698B3D3cc7Ca32193d9955")

	// Create 2 contracts, (1) writes 42 to slot 42, (2) calls (1)
	store42Program := program.New().Sstore(0x42, 0x42)
	callBobProgram := program.New().Call(nil, bobAddr, 1, 0, 0, 0, 0)

	alloc := *actionsHelpers.DefaultAlloc
	alloc.L2Alloc = make(map[common.Address]types.Account)
	alloc.L2Alloc[aa] = types.Account{
		Code: store42Program.Bytes(),
	}
	alloc.L2Alloc[bb] = types.Account{
		Code: callBobProgram.Bytes(),
	}

	testCfg.Allocs = &alloc

	tp := helpers.NewTestParams()
	env := helpers.NewL2FaultProofEnv(t, testCfg, tp, helpers.NewBatcherCfg())
	require.Equal(gt, env.Bob.Address(), bobAddr)

	cl := env.Engine.EthClient()

	env.Sequencer.ActL2PipelineFull(t)
	env.Miner.ActEmptyBlock(t)
	env.Sequencer.ActL2StartBlock(t)

	aliceSecret := env.Alice.L2.Secret()
	bobSecret := env.Bob.L2.Secret()

	chainID := env.Sequencer.RollupCfg.L2ChainID

	// Sign authorization tuples.
	// The way the auths are combined, it becomes
	// 1. tx -> addr1 which is delegated to 0xaaaa
	// 2. addr1:0xaaaa calls into addr2:0xbbbb
	// 3. addr2:0xbbbb  writes to storage
	auth1, err := types.SignSetCode(aliceSecret, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(chainID),
		Address: bb,
		Nonce:   1,
	})
	require.NoError(gt, err, "failed to sign auth1")
	auth2, err := types.SignSetCode(bobSecret, types.SetCodeAuthorization{
		Address: aa,
		Nonce:   0,
	})
	require.NoError(gt, err, "failed to sign auth2")

	txdata := &types.SetCodeTx{
		ChainID:   uint256.MustFromBig(chainID),
		Nonce:     0,
		To:        env.Alice.Address(),
		Gas:       500000,
		GasFeeCap: uint256.NewInt(5000000000),
		GasTipCap: uint256.NewInt(2),
		AuthList:  []types.SetCodeAuthorization{auth1, auth2},
	}
	signer := types.NewIsthmusSigner(chainID)
	tx := types.MustSignNewTx(aliceSecret, signer, txdata)

	err = cl.SendTransaction(t.Ctx(), tx)
	require.NoError(gt, err, "failed to send set code tx")

	_, err = env.Engine.EngineApi.IncludeTx(tx, env.Alice.Address())
	require.NoError(t, err, "failed to include set code tx")

	env.Sequencer.ActL2EndBlock(t)

	// Verify delegation designations were deployed.
	bobCode, err := cl.PendingCodeAt(t.Ctx(), env.Bob.Address())
	require.NoError(gt, err, "failed to get bob code")
	want := types.AddressToDelegation(auth2.Address)
	if !bytes.Equal(bobCode, want) {
		t.Fatalf("addr1 code incorrect: got %s, want %s", common.Bytes2Hex(bobCode), common.Bytes2Hex(want))
	}
	aliceCode, err := cl.PendingCodeAt(t.Ctx(), env.Alice.Address())
	require.NoError(gt, err, "failed to get alice code")
	want = types.AddressToDelegation(auth1.Address)
	if !bytes.Equal(aliceCode, want) {
		t.Fatalf("addr2 code incorrect: got %s, want %s", common.Bytes2Hex(aliceCode), common.Bytes2Hex(want))
	}

	// Verify delegation executed the correct code.
	fortyTwo := common.BytesToHash([]byte{0x42})
	actual, err := cl.PendingStorageAt(t.Ctx(), env.Bob.Address(), fortyTwo)
	require.NoError(gt, err, "failed to get addr1 storage")

	if !bytes.Equal(actual, fortyTwo[:]) {
		t.Fatalf("addr2 storage wrong: expected %d, got %d", fortyTwo, actual)
	}

	// batch submit to L1. batcher should submit span batches.
	env.BatchAndMine(t)

	env.Sequencer.ActL1HeadSignal(t)
	env.Sequencer.ActL2PipelineFull(t)

	latestBlock, err := cl.BlockByNumber(t.Ctx(), nil)
	require.NoError(t, err, "error fetching latest block")

	env.RunFaultProofProgram(t, latestBlock.NumberU64(), testCfg.CheckResult, testCfg.InputParams...)
}

// TestInvalidSetCodeTxBatch tests that batches that include SetCodeTxs are dropped before Isthmus
func Test_ProgramAction_InvalidSetCodeTxBatch(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	matrix.AddDefaultTestCases(
		nil,
		helpers.NewForkMatrix(helpers.Holocene),
		testInvalidSetCodeTxBatch,
	)
	matrix.Run(gt)
}

func testInvalidSetCodeTxBatch(gt *testing.T, testCfg *helpers.TestCfg[any]) {
	t := actionsHelpers.NewDefaultTesting(gt)
	env := helpers.NewL2FaultProofEnv(t, testCfg, helpers.NewTestParams(), helpers.NewBatcherCfg())
	sequencer := env.Sequencer
	miner := env.Miner
	batcher := env.Batcher

	sequencer.ActL2EmptyBlock(t)
	u1 := sequencer.L2Unsafe()
	sequencer.ActL2EmptyBlock(t) // we'll inject the setcode tx in this block's batch

	rng := rand.New(rand.NewSource(0))
	setcodetx := testutils.RandomSetCodeTx(rng, types.NewPragueSigner(env.Sd.RollupCfg.L2ChainID))
	batcher.ActL2BatchBuffer(t)
	batcher.ActL2BatchBuffer(t, func(block *types.Block) *types.Block {
		// inject user tx into upgrade batch
		return block.WithBody(types.Body{Transactions: append(block.Transactions(), setcodetx)})
	})
	batcher.ActL2ChannelClose(t)
	batcher.ActL2BatchSubmit(t)
	miner.ActL1StartBlock(12)(t)
	miner.ActL1IncludeTxByHash(env.Batcher.LastSubmitted.Hash())(t)
	miner.ActL1EndBlock(t)

	sequencer.ActL1HeadSignal(t)
	sequencer.ActL2PipelineFull(t)

	l2safe := sequencer.L2Safe()
	s2block := env.Engine.L2Chain().GetBlockByHash(l2safe.Hash)
	require.Len(t, s2block.Transactions(), 1, "safe head should only contain l1 info deposit")
	require.Equal(t, u1, l2safe, "expected last block to be reorgd out due to setcode tx")

	recs := env.Logs.FindLogs(testlog.NewMessageFilter("sequencers may not embed any SetCode transactions before Isthmus"))
	require.Len(t, recs, 1)

	env.RunFaultProofProgram(t, l2safe.Number, testCfg.CheckResult, testCfg.InputParams...)
}
