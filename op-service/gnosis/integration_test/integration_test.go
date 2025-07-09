package integration_test

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/gnosis"
)

const privateKey = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"

func TestGnosisClient(t *testing.T) {
	// Use anvil's first account private key
	signerAddress := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	recipientAddress := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")

	privateKeyEcdsa, err := crypto.HexToECDSA(privateKey)
	require.NoError(t, err)

	// Create Safe gnosis
	lgr := log.NewLogger(os.Stdout, log.DefaultCLIConfig())
	rpcUrl, ethClient := devnet.DefaultAnvilRPC(t, lgr)
	safeAddress, testContractAddr := deploySafeContracts(t, rpcUrl, privateKey)

	gnosisClient, err := gnosis.NewGnosisClient(
		lgr,
		rpcUrl,
		[]string{privateKey},
		safeAddress,
		gnosis.WithCustomTxMgr(func(cfg *txmgr.CLIConfig) {
			cfg.ReceiptQueryInterval = 1 * time.Second
			cfg.NumConfirmations = 1
			cfg.TxSendTimeout = 5 * time.Second
		}),
	)
	require.NoError(t, err)
	defer gnosisClient.Close()

	ctx := context.Background()

	t.Run("GetOwners", func(t *testing.T) {
		owners, err := gnosisClient.GetOwners(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(owners))
		require.Equal(t, signerAddress, owners[0])

		isOwner, err := gnosisClient.IsOwner(ctx, signerAddress)
		require.NoError(t, err)
		require.True(t, isOwner, "Signer %s is not an owner of the Safe", signerAddress.Hex())
	})

	t.Run("ExecuteTransaction", func(t *testing.T) {
		// Fund the Safe
		nonce, err := ethClient.PendingNonceAt(ctx, signerAddress)
		require.NoError(t, err)

		fundAmount := big.NewInt(2000000000000000000) // 2 ETH
		fundTx := types.NewTransaction(nonce, safeAddress, fundAmount, 100000, big.NewInt(20000000000), nil)
		signedFundTx, err := types.SignTx(fundTx, types.NewEIP155Signer(gnosisClient.ChainID), privateKeyEcdsa)
		require.NoError(t, err)

		err = ethClient.SendTransaction(ctx, signedFundTx)
		require.NoError(t, err)

		// Wait for funding transaction
		fundingReceipt := waitForTransaction(t, ctx, ethClient, signedFundTx.Hash())
		require.Equal(t, uint64(1), fundingReceipt.Status, "Funding transaction should succeed")

		// Get initial Safe state
		initialNonce, err := gnosisClient.GetNonce(ctx)
		require.NoError(t, err)

		initialSafeBalance, err := ethClient.BalanceAt(ctx, safeAddress, nil)
		require.NoError(t, err)
		require.Greater(t, initialSafeBalance.Int64(), int64(0), "Safe balance should be greater than 0")

		// Create Safe transaction to send 1 ETH to another address
		sendAmount := big.NewInt(1000000000000000000) // 1 ETH
		receipt, err := gnosisClient.SendTransaction(ctx, recipientAddress, sendAmount, nil, 0)
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status) // Success

		// Verify the Safe state changes
		finalNonce, err := gnosisClient.GetNonce(ctx)
		require.NoError(t, err)
		finalSafeBalance, err := ethClient.BalanceAt(ctx, safeAddress, nil)

		require.NoError(t, err)
		require.Less(t, finalSafeBalance.Int64(), initialSafeBalance.Int64(), "Safe balance should decrease after successful transaction")
		require.Equal(t, initialSafeBalance.Int64()-finalSafeBalance.Int64(), sendAmount.Int64())
		expectedNonce := new(big.Int).Add(initialNonce, big.NewInt(1))
		require.Equal(t, expectedNonce, finalNonce, "Safe nonce should increment after successful transaction")
	})

	t.Run("TestDelegateCall", func(t *testing.T) {
		// Create ABI for TestDelegateCall
		testContractABI, err := abi.JSON(strings.NewReader(`[
			{"inputs":[{"type":"uint256","name":"_value"}],"name":"setTestValue","type":"function"},
			{"inputs":[],"name":"getTestValue","type":"function","outputs":[{"type":"uint256"}],"stateMutability":"view"}
		]`))
		require.NoError(t, err)

		calldata, err := testContractABI.Pack("setTestValue", big.NewInt(123))
		require.NoError(t, err)

		// Execute via delegatecall (operation = 1)
		receipt, err := gnosisClient.SendTransaction(ctx, testContractAddr, big.NewInt(0), calldata, gnosis.OperationDelegateCall)
		require.NoError(t, err)
		require.Equal(t, uint64(1), receipt.Status)

		// Calculate the storage slot used by the contract: keccak256("TestDelegateCall.testValue")
		storageKey := crypto.Keccak256Hash([]byte("TestDelegateCall.testValue"))

		// Read Safe's storage at that slot
		safeStorageValue, err := ethClient.StorageAt(ctx, safeAddress, storageKey, nil)
		require.NoError(t, err)

		// Convert bytes to big.Int
		safeValue := new(big.Int).SetBytes(safeStorageValue)
		require.Zero(t, safeValue.Cmp(big.NewInt(123)), "Safe's storage should contain the value 123")

		// Verify test contract's own storage was NOT modified (using function call)
		getValueCalldata, err := testContractABI.Pack("getTestValue")
		require.NoError(t, err)

		contractStorageResult, err := ethClient.CallContract(ctx, ethereum.CallMsg{
			To:   &testContractAddr, // Call against test contract address
			Data: getValueCalldata,
		}, nil)
		require.NoError(t, err)

		var contractValue *big.Int
		err = testContractABI.UnpackIntoInterface(&contractValue, "getTestValue", contractStorageResult)
		require.NoError(t, err)
		require.True(t, contractValue.Cmp(big.NewInt(0)) == 0, "Test contract's storage should remain unchanged (expected 0, got %s)", contractValue.String())
	})
}
