package isthmus

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/devnet-sdk/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func SendValueTx(ctx context.Context, chainID *big.Int, client *ethclient.Client, from system.Wallet, to common.Address, value *big.Int, send bool) (receipt *gethTypes.Receipt, tx *gethTypes.Transaction, err error) {
	if value.Sign() == 0 || value.Sign() == -1 {
		return nil, nil, fmt.Errorf("value is 0 or negative")
	}

	// Get pending nonce
	nonce, err := client.PendingNonceAt(ctx, from.Address())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get pending nonce: %w", err)
	}

	// Calculate gas parameters using utility function
	gasLimit, gasTipCap, gasFeeCap, err := CalculateGasParams(ctx, client, from.Address(), to, value, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate gas parameters: %w", err)
	}

	// Create transaction
	txData := &gethTypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &to,
		Value:     value,
		Data:      nil,
	}

	// Sign transaction
	tx = gethTypes.NewTx(txData)
	signedTx, err := gethTypes.SignTx(tx, gethTypes.LatestSignerForChainID(chainID), from.PrivateKey())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	if !send {
		return nil, signedTx, nil
	}

	// Send transaction
	err = client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction receipt with timeout
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	receipt, err = waitForTransaction(ctx, client, signedTx.Hash())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to wait for transaction: %w", err)
	}
	if receipt == nil {
		return nil, nil, fmt.Errorf("receipt is nil")
	}
	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		return nil, nil, fmt.Errorf("expected successful transaction (1), instead got status: %d", receipt.Status)
	}

	return receipt, tx, nil
}

// CalculateGasParams calculates appropriate gas parameters for a transaction
func CalculateGasParams(ctx context.Context, client *ethclient.Client, from common.Address, to common.Address, value *big.Int, data []byte) (estimatedGas uint64, gasTipCap *big.Int, gasFeeCap *big.Int, err error) {
	// Get current block header for base fee
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to get header: %w", err)
	}

	// Get suggested gas tip
	gasTipCap, err = client.SuggestGasTipCap(ctx)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("failed to get suggested gas tip: %w", err)
	}

	// Calculate gas fee cap (2 * baseFee + tip)
	gasFeeCap = new(big.Int).Add(
		new(big.Int).Mul(header.BaseFee, big.NewInt(2)),
		gasTipCap,
	)

	// Estimate gas limit
	estimatedGas, err = client.EstimateGas(ctx, ethereum.CallMsg{
		From:  from,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		// Return error but also provide a fallback estimated gas in case caller wants to continue
		return 300000, gasTipCap, gasFeeCap, fmt.Errorf("failed to estimate gas: %w", err)
	}

	return estimatedGas, gasTipCap, gasFeeCap, nil
}

// waitForTransaction polls for a transaction receipt until it is available or the context is canceled.
// It's a simpler version of the functionality in SimpleTxManager.
func waitForTransaction(ctx context.Context, client *ethclient.Client, hash common.Hash) (*gethTypes.Receipt, error) {
	ticker := time.NewTicker(500 * time.Millisecond) // Poll every 500ms
	defer ticker.Stop()

	// Record starting block number
	startBlockNum, err := client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting block number: %w", err)
	}

	for {
		receipt, err := client.TransactionReceipt(ctx, hash)
		if receipt != nil && err == nil {
			return receipt, nil
		} else if err != nil && !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
		}

		select {
		case <-ctx.Done():
			// Get current block number to calculate progress
			// Create a new context for this query since the original is canceled
			queryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			currentBlockNum, blockErr := client.BlockNumber(queryCtx)
			cancel() // Always cancel the context to avoid leaks

			if blockErr != nil {
				// If we can't get the current block number, just return the original error
				return nil, fmt.Errorf("context error: %w (could not determine block progress: %w)", ctx.Err(), blockErr)
			}

			blockProgress := int64(currentBlockNum) - int64(startBlockNum)
			return nil, fmt.Errorf("transaction %s not found after %d blocks: %w", hash.Hex(), blockProgress, ctx.Err())
		case <-ticker.C:
			// Continue polling
		}
	}
}

func ReturnRemainingFunds(t systest.T, ctx context.Context, chainID *big.Int, client *ethclient.Client, from system.Wallet, to system.Wallet, logger log.Logger) {
	remainingBalance, err := client.BalanceAt(ctx, from.Address(), nil)
	require.NoError(t, err)

	estimatedGas, _, gasFeeCap, err := CalculateGasParams(ctx, client, from.Address(), to.Address(), big.NewInt(int64(0)), nil)
	// Calculate the gas cost to subtract from the remaining balance
	gasCost := new(big.Int).Mul(big.NewInt(int64(estimatedGas)), gasFeeCap)
	// Subtract the gas cost from the remaining balance to avoid "insufficient funds" error
	balanceAfterGasCost := new(big.Int).Sub(remainingBalance, gasCost)
	balanceAfterGasCost = new(big.Int).Sub(balanceAfterGasCost, big.NewInt(1000000))
	require.NoError(t, err)
	logger.Info("Cleanup: Returning remaining funds from wallet", "from", from.Address().Hex(), "to", to.Address().Hex())
	if balanceAfterGasCost.Sign() > 0 {
		_, _, err = SendValueTx(t.Context(), chainID, client, from, to.Address(), balanceAfterGasCost, true)
		require.NoError(t, err, "Return fund transaction failed")
	}
}

func NewTestWallet(ctx context.Context, chain system.Chain) (system.Wallet, error) {
	// create new test wallet
	testWalletPrivateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	testWalletPrivateKeyBytes := crypto.FromECDSA(testWalletPrivateKey)
	testWalletPrivateKeyHex := hex.EncodeToString(testWalletPrivateKeyBytes)
	testWalletPublicKey := testWalletPrivateKey.Public()
	testWalletPublicKeyECDSA, ok := testWalletPublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Failed to assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	testWalletAddress := crypto.PubkeyToAddress(*testWalletPublicKeyECDSA)
	testWallet, err := system.NewWallet(
		testWalletPrivateKeyHex,
		types.Address(testWalletAddress),
		chain,
	)
	return testWallet, err
}
