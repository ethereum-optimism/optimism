package isthmus

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/testlib/validators"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const (
	// Test vectors for Pectra features
	SET_CODE_TX_BASIC uint64 = iota
	BLOCK_HISTORY_CONSISTENCY
	EMPTY_REQUESTS_HASH
	CALLDATA_COST_INCREASE
)

// TestPectra verifies that L1 Pectra inherited features function properly on the OP Stack.
func TestPectra(t *testing.T) {
	// Define which L2 chain we'll test.
	chainIdx := uint64(0)

	// Get validators and getters for accessing the system and wallets
	walletGetter, walletValidator := validators.AcquireL2WalletWithFunds(chainIdx, types.NewBalance(big.NewInt(params.Ether)))

	// Run isthmus tests
	_, forkValidator := validators.AcquireL2WithFork(chainIdx, rollup.Isthmus)

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
		t.Run(c.name, func(t *testing.T) {
			systest.SystemTest(t,
				pectraFeatureTest(walletGetter, chainIdx, c.testVector),
				walletValidator,
				forkValidator,
			)
		})
	}
}

// pectraFeatureTest returns a SystemTestFunc that tests one of the Pectra features.
func pectraFeatureTest(
	walletGetter validators.WalletGetter,
	chainIdx uint64,
	testVector uint64,
) systest.SystemTestFunc {
	return func(t systest.T, sys system.System) {
		ctx := t.Context()

		// Get the system and wallet
		wallet := walletGetter(ctx)

		// Get the L2 client
		l2Chain := sys.L2s()[chainIdx]
		l2Client, err := l2Chain.Nodes()[0].GethClient()
		require.NoError(t, err)

		// Test various Pectra features
		switch testVector {
		case SET_CODE_TX_BASIC:
			runSetCodeTxBasicTest(ctx, t, wallet, l2Chain, l2Client)
		case BLOCK_HISTORY_CONSISTENCY:
			runBlockHistoryConsistencyTest(ctx, t, l2Client)
		case EMPTY_REQUESTS_HASH:
			runEmptyRequestsHashTest(ctx, t, l2Client)
		case CALLDATA_COST_INCREASE:
			runCalldataCostTest(ctx, t, wallet, l2Chain)
		default:
			t.Fatalf("unknown test vector: %d", testVector)
		}
	}
}

// Tests that a basic EIP-7702 transaction correctly executes on the L2 chain.
func runSetCodeTxBasicTest(ctx context.Context, t systest.T, wallet system.Wallet, l2Chain system.L2Chain, l2Client *ethclient.Client) {
	// Get wallet priv/pub key pair
	privateKey := wallet.PrivateKey()
	fromAddr := wallet.Address()

	// ================================
	// Part 1: Deploy test contract
	// ================================

	storeProgram := program.New().Sstore(0, 0xbeef).Bytes()

	// Deploy the store contract
	walletv2, err := system.NewWalletV2FromWalletAndChain(ctx, wallet, l2Chain)
	require.NoError(t, err)

	storeAddr, err := DeployProgram(ctx, walletv2, storeProgram)
	require.NoError(t, err)

	// Check if the store contract was deployed
	code, err := l2Client.CodeAt(ctx, storeAddr, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code, "Store contract not deployed")
	require.Equal(t, code, storeProgram, "Store contract code incorrect")

	// ================================
	// Part 2: Send SetCode transaction
	// ================================

	nonce, err := l2Client.PendingNonceAt(ctx, fromAddr)
	require.NoError(t, err, "Failed to fetch pending nonce")

	auth1, err := gethTypes.SignSetCode(privateKey, gethTypes.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(l2Chain.ID()),
		Address: storeAddr,
		// before the nonce is compared with the authorization in the EVM, it is incremented by 1
		Nonce: nonce + 1,
	})
	require.NoError(t, err, "Failed to sign 7702 authorization")

	opts := DefaultTxOpts(walletv2)
	setCodeTx := txplan.NewPlannedTx(
		opts,
		txplan.WithType(gethTypes.SetCodeTxType),
		txplan.WithTo(&fromAddr),
		txplan.WithGasLimit(75_000),
		txplan.WithAuthorizations([]gethTypes.SetCodeAuthorization{auth1}),
	)

	// todo: The gas estimator in this test suite doesn't yet handle the intrinsic gas of EIP-7702 transactions.
	// We hardcode the gas estimation function here to avoid the issue.
	setCodeTx.Gas.Fn(func(ctx context.Context) (uint64, error) {
		return 75_000, nil
	})

	// Fetch the receipt for the tx
	receipt, err := setCodeTx.Included.Eval(ctx)
	require.NoError(t, err)

	// Ensure the transaction was successful
	require.Equal(t, uint64(1), receipt.Status)

	// Check if the delegation was deployed
	code, err = l2Client.CodeAt(ctx, fromAddr, nil)
	require.NoError(t, err)
	wantCode := gethTypes.AddressToDelegation(auth1.Address)
	require.Equal(t, wantCode, code, "Delegation code incorrect")

	// Check if the account has its storage slot set correctly
	storageValue, err := l2Client.StorageAt(ctx, fromAddr, common.Hash{}, nil)
	require.NoError(t, err)
	require.EqualValues(t, storageValue, common.BytesToHash([]byte{0xbe, 0xef}), "Storage slot not set in delegated EOA")
}

// runBlockHistoryConsistencyTest tests that the block history contract is consistent with the chain.
func runBlockHistoryConsistencyTest(ctx context.Context, t systest.T, l2Client *ethclient.Client) {
	// Get the latest block number
	latestBlock, err := l2Client.BlockByNumber(ctx, nil)
	require.NoError(t, err)

	// Get the block history contract code
	code, err := l2Client.CodeAt(ctx, params.HistoryStorageAddress, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code, "Block history contract not deployed")

	// Get the slot containing the parent block hash
	parentHashSlotNum := (latestBlock.Number().Uint64() - 1) % (params.HistoryServeWindow - 1)

	// Turn the uint64 into a 32-byte array
	parentHashSlot := make([]byte, 32)
	binary.BigEndian.PutUint64(parentHashSlot[24:32], parentHashSlotNum)

	parentHashSlotValue, err := l2Client.StorageAt(ctx, params.HistoryStorageAddress, common.BytesToHash(parentHashSlot), latestBlock.Number())
	require.NoError(t, err)

	// Ensure the parent block hash in the contract matches the parent block hash of the latest block
	require.EqualValues(
		t,
		latestBlock.ParentHash(),
		common.BytesToHash(parentHashSlotValue),
		"Parent block hash in contract does not match parent block hash of latest block",
	)
}

// runEmptyRequestsHashTest tests that the requests hash is empty for Isthmus-activated blocks.
func runEmptyRequestsHashTest(ctx context.Context, t systest.T, l2Client *ethclient.Client) {
	// Get the latest block header
	latestBlock, err := l2Client.HeaderByNumber(ctx, nil)
	require.NoError(t, err)

	// Check that the requests hash is empty
	require.Equal(t, *latestBlock.RequestsHash, gethTypes.EmptyRequestsHash, "Requests hash is not empty on L2")
}

// runCalldataCostTest tests that the calldata cost of Isthmus transactions is correctly calculated per EIP-7623.
func runCalldataCostTest(ctx context.Context, t systest.T, wallet system.Wallet, l2Chain system.L2Chain) {
	walletv2, err := system.NewWalletV2FromWalletAndChain(ctx, wallet, l2Chain)
	require.NoError(t, err)

	dat := make([]byte, 2048)
	_, err = rand.Read(dat)
	require.NoError(t, err)

	idPrecompile := common.BytesToAddress([]byte{0x4})
	idTx := txplan.NewPlannedTx(DefaultTxOpts(walletv2), txplan.WithData(dat), txplan.WithTo(&idPrecompile))
	receipt, err := idTx.Included.Eval(ctx)
	require.NoError(t, err)

	// Ensure the transaction was successful
	require.Equal(t, uint64(1), receipt.Status)

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
	require.EqualValues(t, expectedGas, receipt.GasUsed, "Gas usage does not match expected value")
}
