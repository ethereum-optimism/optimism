package pectra

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

const (
	SET_CODE_TX_BASIC uint64 = iota
	BLOCK_HISTORY_CONSISTENCY
	EMPTY_REQUESTS_HASH
	CALLDATA_COST_INCREASE
)

func TestPectra(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	require := t.Require()

	err := dsl.RequiresL2Fork(t.Ctx(), sys, 0, rollup.Isthmus)
	require.NoError(err, "Isthmus fork must be active for Pectra features")

	alice := sys.FunderL2.NewFundedEOA(eth.OneTenthEther)

	cases := []struct {
		name       string
		testVector uint64
	}{
		{"SetCodeBasic", SET_CODE_TX_BASIC},
		{"BlockHistoryConsistency", BLOCK_HISTORY_CONSISTENCY},
		{"EmptyRequestsHash", EMPTY_REQUESTS_HASH},
		{"CalldataCostIncrease", CALLDATA_COST_INCREASE},
	}

	for _, c := range cases {
		t.Run(c.name, func(t devtest.T) {
			runPectraFeatureTest(t, sys, alice, c.testVector)
		})
	}
}

func runPectraFeatureTest(t devtest.T, sys *presets.Minimal, alice *dsl.EOA, testVector uint64) {
	require := t.Require()

	switch testVector {
	case SET_CODE_TX_BASIC:
		runSetCodeTxBasicTest(t, alice, sys)
	case BLOCK_HISTORY_CONSISTENCY:
		runBlockHistoryConsistencyTest(t, sys.L2EL)
	case EMPTY_REQUESTS_HASH:
		runEmptyRequestsHashTest(t, sys.L2EL)
	case CALLDATA_COST_INCREASE:
		runCalldataCostTest(t, alice)
	default:
		require.Fail("unknown test vector: %d", testVector)
	}
}

func runSetCodeTxBasicTest(t devtest.T, alice *dsl.EOA, sys *presets.Minimal) {
	require := t.Require()

	// ================================
	// Part 1: Deploy test contract
	// ================================

	storeProgram := program.New().Sstore(0, 0xbeef).Bytes()

	// Deploy the store contract
	program := program.New().ReturnViaCodeCopy(storeProgram)

	storeAddr := deployProgram(t, alice, program.Bytes())
	t.Logf("Contract deployed at address: %s", storeAddr.Hex())

	l2Client := sys.L2EL.Escape().EthClient()

	latestBlock, err := l2Client.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(err)
	t.Logf("Latest block: %d, hash: %s", latestBlock.NumberU64(), latestBlock.Hash().Hex())

	// Check if the store contract was deployed
	code, err := l2Client.CodeAtHash(t.Ctx(), storeAddr, latestBlock.Hash())
	require.NoError(err)
	t.Logf("Contract code length at latest block: %d", len(code))
	require.NotEmpty(code, "Store contract not deployed")
	require.Equal(code, storeProgram, "Store contract code incorrect")

	// ================================
	// Part 2: Send SetCode transaction
	// ================================

	fromAddr := alice.Address()
	nonce, err := l2Client.PendingNonceAt(t.Ctx(), fromAddr)
	require.NoError(err, "Failed to fetch pending nonce")

	auth1, err := gethTypes.SignSetCode(alice.Key().Priv(), gethTypes.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(sys.L2Chain.ChainID().ToBig()),
		Address: storeAddr,
		// before the nonce is compared with the authorization in the EVM, it is incremented by 1
		Nonce: nonce + 1,
	})
	require.NoError(err, "Failed to sign 7702 authorization")

	setCodeTxOpts := txplan.Combine(
		alice.Plan(),
		txplan.WithType(gethTypes.SetCodeTxType),
		txplan.WithTo(&fromAddr),
		txplan.WithGasLimit(75_000),
		txplan.WithAuthorizations([]gethTypes.SetCodeAuthorization{auth1}),
	)

	setCodeTx := txplan.NewPlannedTx(setCodeTxOpts)
	// todo: The gas estimator in this test suite doesn't yet handle the intrinsic gas of EIP-7702 transactions.
	// We hardcode the gas estimation function here to avoid the issue.
	setCodeTx.Gas.Fn(func(ctx context.Context) (uint64, error) {
		return 75_000, nil
	})

	// Fetch the receipt for the tx
	receipt, err := setCodeTx.Included.Eval(t.Ctx())
	require.NoError(err)

	// Ensure the transaction was successful
	require.Equal(uint64(1), receipt.Status)

	newBlock, err := l2Client.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(err)

	// Check if the delegation was deployed
	code, err = l2Client.CodeAtHash(t.Ctx(), fromAddr, newBlock.Hash())
	require.NoError(err)
	wantCode := gethTypes.AddressToDelegation(auth1.Address)
	require.Equal(wantCode, code, "Delegation code incorrect")

	// Check if the account has its storage slot set correctly
	storageValue, err := l2Client.GetStorageAt(t.Ctx(), fromAddr, common.Hash{}, newBlock.Hash().String())
	require.NoError(err)
	require.EqualValues(storageValue, common.BytesToHash([]byte{0xbe, 0xef}), "Storage slot not set in delegated EOA")
}

func runBlockHistoryConsistencyTest(t devtest.T, l2EL *dsl.L2ELNode) {
	require := t.Require()

	l2Client := l2EL.Escape().EthClient()

	// Get the latest block number
	latestBlock, err := l2Client.InfoByLabel(t.Ctx(), eth.Unsafe)
	require.NoError(err)
	require.Greater(latestBlock.NumberU64(), uint64(1), "Need at least 2 blocks for history test")

	// Get the block history contract code
	code, err := l2Client.CodeAtHash(t.Ctx(), params.HistoryStorageAddress, latestBlock.Hash())
	require.NoError(err)
	require.NotEmpty(code, "Block history contract not deployed")

	// Get the slot containing the parent block hash
	parentHashSlotNum := (latestBlock.NumberU64() - 1) % (params.HistoryServeWindow - 1)

	// Turn the uint64 into a 32-byte array
	parentHashSlot := make([]byte, 32)
	binary.BigEndian.PutUint64(parentHashSlot[24:32], parentHashSlotNum)
	parentHashSlotValue, err := l2Client.GetStorageAt(t.Ctx(), params.HistoryStorageAddress, common.BytesToHash(parentHashSlot), latestBlock.Hash().String())
	require.NoError(err)

	// Ensure the parent block hash in the contract matches the parent block hash of the latest block
	require.EqualValues(
		latestBlock.ParentHash(),
		parentHashSlotValue,
		"Parent block hash in contract does not match parent block hash of latest block",
	)
}

func runEmptyRequestsHashTest(t devtest.T, l2EL *dsl.L2ELNode) {
	require := t.Require()

	// Get the latest block header
	var rawHeader struct {
		RequestsHash *common.Hash `json:"requestsHash,omitempty"`
	}
	err := l2EL.Escape().L2EthClient().RPC().CallContext(t.Ctx(), &rawHeader, "eth_getBlockByNumber", "latest", false)
	require.NoError(err, "Failed to get raw block header")

	// Check that the requests hash is empty
	if rawHeader.RequestsHash != nil {
		require.Equal(*rawHeader.RequestsHash, gethTypes.EmptyRequestsHash, "Requests hash is not empty on L2")
	} else {
		require.Fail("RequestsHash field not found - chain may not be Isthmus-activated")
	}
}

func runCalldataCostTest(t devtest.T, alice *dsl.EOA) {
	require := t.Require()

	dat := make([]byte, 2048)
	_, err := rand.Read(dat)
	require.NoError(err)

	idPrecompile := common.BytesToAddress([]byte{0x4})
	idTxOpts := txplan.Combine(
		alice.Plan(),
		txplan.WithData(dat),
		txplan.WithTo(&idPrecompile),
	)

	idTx := txplan.NewPlannedTx(idTxOpts)
	receipt, err := idTx.Included.Eval(t.Ctx())
	require.NoError(err)
	require.Equal(uint64(1), receipt.Status)

	// ID Precompile:
	//   data_word_size = (data_size + 31) / 32
	//   id_static_gas = 15
	//   id_dynamic_gas = 3 * data_word_size
	// EIP-7623:
	//   total_cost_floor_per_token = 10
	//   standard_token_cost = 4
	//   tokens_in_calldata = zero_bytes_in_calldata + nonzero_bytes_in_calldata * 4
	//   calldata_cost = standard_token_cost * tokens_in_calldata
	//
	// Expected gas usage is:
	// 21_000 (base cost) + max(id_static_gas + id_dynamic_gas + calldata_cost, total_cost_floor_per_token * tokens_in_calldata)
	var zeros, nonZeros int
	for _, b := range dat {
		if b == 0 {
			zeros++
		} else {
			nonZeros++
		}
	}
	tokensInCalldata := zeros + nonZeros*4

	expectedGas := 21_000 + max(15+3*((len(dat)+31)/32)+4*tokensInCalldata, 10*tokensInCalldata)
	require.EqualValues(expectedGas, receipt.GasUsed, "Gas usage does not match expected value")

	t.Log("Calldata cost test completed successfully",
		"gasUsed", receipt.GasUsed,
		"expectedGas", expectedGas,
		"zeros", zeros,
		"nonZeros", nonZeros)
}

func deployProgram(t devtest.T, user *dsl.EOA, bytecode []byte) common.Address {
	require := t.Require()

	deployTxOpts := txplan.Combine(
		user.Plan(),
		txplan.WithData(bytecode),
		txplan.WithGasLimit(1_000_000),
	)

	deployTx := txplan.NewPlannedTx(deployTxOpts)
	receipt, err := deployTx.Included.Eval(t.Ctx())
	require.NoError(err)

	t.Logf("Deployment receipt: Status=%d, GasUsed=%d, CumulativeGasUsed=%d",
		receipt.Status, receipt.GasUsed, receipt.CumulativeGasUsed)
	t.Logf("Deployment transaction: BlockNumber=%d, TransactionIndex=%d",
		receipt.BlockNumber.Uint64(), receipt.TransactionIndex)

	require.Equal(uint64(1), receipt.Status, "Contract deployment failed")
	require.NotNil(receipt.ContractAddress, "Contract address not set in receipt")

	// Log more details about the deployment
	if receipt.ContractAddress != (common.Address{}) {
		t.Logf("Contract address from receipt: %s", receipt.ContractAddress.Hex())
	}

	return receipt.ContractAddress
}
