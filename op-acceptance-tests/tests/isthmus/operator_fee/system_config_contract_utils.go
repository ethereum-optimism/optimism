package isthmus

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/devnet-sdk/testing/systest"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/lmittmann/w3"
	"github.com/stretchr/testify/require"
)

func UpdateOperatorFeeParams(t systest.T, l1ChainID *big.Int, client *ethclient.Client, systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, wallet system.Wallet, operatorFeeConstant uint64, operatorFeeScalar uint32, logger log.Logger) (*gethTypes.Transaction, *gethTypes.Receipt) {
	ctx := t.Context()
	logger.Info("Updating operator fee params",
		"constant", operatorFeeConstant,
		"scalar", operatorFeeScalar)

	nonce, err := client.PendingNonceAt(ctx, wallet.Address())
	require.NoError(t, err)
	logger.Debug("Using nonce",
		"nonce", nonce,
		"wallet", wallet.Address().Hex())

	// Construct call input
	logger.Debug("Constructing function call to setOperatorFeeScalars")
	funcSetOperatorFeeScalars := w3.MustNewFunc(`setOperatorFeeScalars(uint32 _operatorFeeScalar, uint64 _operatorFeeConstant)`, "")
	args, err := funcSetOperatorFeeScalars.EncodeArgs(
		operatorFeeScalar,
		operatorFeeConstant,
	)
	require.NoError(t, err)

	// Calculate gas parameters
	gasLimit, gasTipCap, gasFeeCap, err := CalculateGasParams(ctx, client, wallet.Address(), systemConfigAddress, big.NewInt(0), args)
	if err != nil {
		logger.Warn("Error calculating gas parameters", "error", err)
	}

	tx := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		To:        &systemConfigAddress,
		Gas:       gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
		Value:     big.NewInt(0),
		Data:      args,
	})
	signer := gethTypes.NewLondonSigner(l1ChainID)
	signedTx, err := gethTypes.SignTx(tx, signer, wallet.PrivateKey())
	require.NoError(t, err)
	logger.Debug("Transaction signed", "hash", signedTx.Hash().Hex())

	logger.Info("Sending transaction to the network")
	err = client.SendTransaction(context.Background(), signedTx)
	require.NoError(t, err)

	// Wait for transaction receipt with timeout
	logger.Info("Waiting for transaction confirmation")
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	receipt, err := waitForTransaction(ctx, client, signedTx.Hash())
	require.NoError(t, err, "Failed to wait for transaction receipt")
	require.NotNil(t, receipt)
	require.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber,
		"gasUsed", receipt.GasUsed)

	// Verify the operator fee scalars were set correctly
	RequireOperatorFeeParamValues(t, systemConfig, receipt.BlockNumber, operatorFeeConstant, operatorFeeScalar)

	return tx, receipt
}

func RequireOperatorFeeParamValues(t systest.T, systemConfig *bindings.SystemConfig, blockNumber *big.Int, expectedOperatorFeeConstant uint64, expectedOperatorFeeScalar uint32) {
	operatorFeeConstant, err := systemConfig.OperatorFeeConstant(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, operatorFeeConstant, expectedOperatorFeeConstant, "operator fee constant should match expectations")

	operatorFeeScalar, err := systemConfig.OperatorFeeScalar(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, operatorFeeScalar, expectedOperatorFeeScalar, "operator fee scalar should match expectations")
}

func RequireL1FeeParamValues(t systest.T, systemConfig *bindings.SystemConfig, blockNumber *big.Int, expectedL1BaseFeeScalar uint32, expectedL1BlobBaseFeeScalar uint32) {
	l1BaseFeeScalar, err := systemConfig.BasefeeScalar(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, l1BaseFeeScalar, expectedL1BaseFeeScalar, "l1 base fee scalar should match expectations")

	blobBaseFeeScalar, err := systemConfig.BlobbasefeeScalar(&bind.CallOpts{BlockNumber: blockNumber})
	require.NoError(t, err)
	require.Equal(t, blobBaseFeeScalar, expectedL1BlobBaseFeeScalar, "l1 blob base fee scalar should match expectations")
}

func UpdateL1FeeParams(t systest.T, l1ChainID *big.Int, client *ethclient.Client, systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, wallet system.Wallet, l1BaseFeeScalar uint32, l1BlobBaseFeeScalar uint32, logger log.Logger) (*gethTypes.Transaction, *gethTypes.Receipt) {
	ctx := t.Context()
	logger.Info("Updating L1 fee params",
		"base fee scalar", l1BaseFeeScalar,
		"blob base fee scalar", l1BlobBaseFeeScalar)

	nonce, err := client.PendingNonceAt(ctx, wallet.Address())
	require.NoError(t, err)
	logger.Debug("Using nonce",
		"nonce", nonce,
		"wallet", wallet.Address().Hex())

	// Construct call input
	logger.Debug("Constructing function call to setGasConfigEcotone")
	funcSetGasConfigEcotone := w3.MustNewFunc(`setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobbasefeeScalar)`, "")
	args, err := funcSetGasConfigEcotone.EncodeArgs(
		l1BaseFeeScalar,
		l1BlobBaseFeeScalar,
	)
	require.NoError(t, err)

	// Calculate gas parameters
	gasLimit, gasTipCap, gasFeeCap, err := CalculateGasParams(ctx, client, wallet.Address(), systemConfigAddress, big.NewInt(0), args)
	if err != nil {
		logger.Warn("Error calculating gas parameters", "error", err)
	}

	tx := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		To:        &systemConfigAddress,
		Gas:       gasLimit,
		GasFeeCap: gasFeeCap,
		GasTipCap: gasTipCap,
		Nonce:     nonce,
		Value:     big.NewInt(0),
		Data:      args,
	})
	signer := gethTypes.NewLondonSigner(l1ChainID)
	signedTx, err := gethTypes.SignTx(tx, signer, wallet.PrivateKey())
	require.NoError(t, err)
	logger.Debug("Transaction signed", "hash", signedTx.Hash().Hex())

	logger.Info("Sending transaction to the network")
	err = client.SendTransaction(context.Background(), signedTx)
	require.NoError(t, err)

	// Wait for transaction receipt with timeout
	logger.Info("Waiting for transaction confirmation")
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()
	receipt, err := waitForTransaction(ctx, client, signedTx.Hash())
	require.NoError(t, err, "Failed to wait for transaction receipt")
	require.NotNil(t, receipt)
	require.Equal(t, gethTypes.ReceiptStatusSuccessful, receipt.Status)
	logger.Info("Transaction confirmed",
		"block", receipt.BlockNumber,
		"gasUsed", receipt.GasUsed)

	// Verify the operator fee scalars were set correctly
	RequireL1FeeParamValues(t, systemConfig, receipt.BlockNumber, l1BaseFeeScalar, l1BlobBaseFeeScalar)

	return tx, receipt
}
