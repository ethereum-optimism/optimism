package proofs_test

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	actionsHelpers "github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/proofs/helpers"
)

var (
	aa = common.HexToAddress("0x000000000000000000000000000000000000aaaa")
	bb = common.HexToAddress("0x000000000000000000000000000000000000bbbb")
)

func runSetCodeTxTypeTest(gt *testing.T, testCfg *helpers.TestCfg[any]) {
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

func TestSetCodeTx(gt *testing.T) {
	matrix := helpers.NewMatrix[any]()
	defer matrix.Run(gt)

	matrix.AddDefaultTestCases(
		nil,
		helpers.LatestForkOnly,
		runSetCodeTxTypeTest,
	)
}
