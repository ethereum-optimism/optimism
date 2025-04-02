package operatorfee

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
)

// DefaultFeeUpdateOptions creates the standard options for fee update transactions
func DefaultFeeUpdateOptions(wallet system.WalletV2, destAddress common.Address) txplan.Option {
	return txplan.Combine(
		txplan.WithPrivateKey(wallet.PrivateKey()),
		txplan.WithChainID(wallet.Client()),
		txplan.WithAgainstLatestBlock(wallet.Client()),
		txplan.WithTo(&destAddress),
		txplan.WithValue(big.NewInt(0)),
		txplan.WithPendingNonce(wallet.Client()),
		txplan.WithEstimator(wallet.Client(), true),
		txplan.WithRetryInclusion(wallet.Client(), 10, retry.Fixed(3*time.Second)),
		txplan.WithTransactionSubmitter(wallet.Client()),
		txplan.WithBlockInclusionInfo(wallet.Client()),
	)
}

// CheckOperatorFeeParamValues checks if the operator fee parameters in the SystemConfig contract match the expected values at a specific block.
func CheckOperatorFeeParamValues(systemConfig *bindings.SystemConfig, blockNumber *big.Int, expectedOperatorFeeConstant uint64, expectedOperatorFeeScalar uint32) error {
	operatorFeeConstant, err := systemConfig.OperatorFeeConstant(&bind.CallOpts{BlockNumber: blockNumber})
	if err != nil {
		return fmt.Errorf("failed to get operator fee constant: %w", err)
	}
	if operatorFeeConstant != expectedOperatorFeeConstant {
		return fmt.Errorf("operator fee constant mismatch: got %d, expected %d", operatorFeeConstant, expectedOperatorFeeConstant)
	}

	operatorFeeScalar, err := systemConfig.OperatorFeeScalar(&bind.CallOpts{BlockNumber: blockNumber})
	if err != nil {
		return fmt.Errorf("failed to get operator fee scalar: %w", err)
	}
	if operatorFeeScalar != expectedOperatorFeeScalar {
		return fmt.Errorf("operator fee scalar mismatch: got %d, expected %d", operatorFeeScalar, expectedOperatorFeeScalar)
	}
	return nil
}

// UpdateOperatorFeeParams updates the operator fee parameters in the SystemConfig contract.
// It constructs and sends a transaction using txplan and returns the signed transaction, the receipt, or an error.
func UpdateOperatorFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, wallet system.WalletV2, operatorFeeConstant uint64, operatorFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	// Construct call input
	funcSetOperatorFeeScalars := w3.MustNewFunc(`setOperatorFeeScalars(uint32 _operatorFeeScalar, uint64 _operatorFeeConstant)`, "")
	args, err := funcSetOperatorFeeScalars.EncodeArgs(
		operatorFeeScalar,
		operatorFeeConstant,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments for setOperatorFeeScalars: %w", err)
	}

	// Create a transaction using txplan
	ptx := txplan.NewPlannedTx(
		DefaultFeeUpdateOptions(wallet, systemConfigAddress),
		txplan.WithData(args),
	)

	// Execute the transaction and wait for inclusion
	receipt, err = ptx.Included.Eval(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("failed to execute transaction or wait for inclusion: %w", err)
	}
	if receipt == nil {
		// This should theoretically not happen if Eval returns nil error, but check defensively
		return nil, errors.New("transaction included but receipt is unexpectedly nil")
	}
	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		// Include Tx Hash in error for easier debugging
		return nil, fmt.Errorf("transaction failed with status %d (tx: %s)", receipt.Status, receipt.TxHash.Hex())
	}

	// Verify the operator fee scalars were set correctly in the contract
	err = CheckOperatorFeeParamValues(systemConfig, receipt.BlockNumber, operatorFeeConstant, operatorFeeScalar)
	if err != nil {
		return nil, fmt.Errorf("failed to verify operator fee param values after tx confirmation: %w", err)
	}

	return receipt, nil
}

func UpdateL1FeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, wallet system.WalletV2, l1BaseFeeScalar uint32, l1BlobBaseFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	// Construct call input
	funcSetGasConfigEcotone := w3.MustNewFunc(`setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobbasefeeScalar)`, "")
	args, err := funcSetGasConfigEcotone.EncodeArgs(
		l1BaseFeeScalar,
		l1BlobBaseFeeScalar,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments for setGasConfigEcotone: %w", err)
	}

	// Get the current nonce for the wallet
	nonce, err := wallet.Client().PendingNonceAt(wallet.Ctx(), wallet.Address())
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get chain ID
	chainID, err := wallet.Client().ChainID(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	// Estimate gas
	gasLimit, err := wallet.Client().EstimateGas(wallet.Ctx(), ethereum.CallMsg{
		From:  wallet.Address(),
		To:    &systemConfigAddress,
		Data:  args,
		Value: big.NewInt(0),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to estimate gas: %w", err)
	}
	// Add a buffer to the gas limit
	gasLimit = uint64(float64(gasLimit) * 1.2)

	header, err := wallet.GethClient().HeaderByNumber(wallet.Ctx(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest header: %w", err)
	}

	// suggest gas tip cap based on header
	gasTipCap := big.NewInt(1)

	gasFeeCap := new(big.Int).Add(
		gasTipCap,
		new(big.Int).Mul(header.BaseFee, big.NewInt(2)),
	)

	// Create the transaction
	tx := gethTypes.NewTx(&gethTypes.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       gasLimit,
		To:        &systemConfigAddress,
		Value:     big.NewInt(0),
		Data:      args,
	})

	// Sign the transaction
	signedTx, err := gethTypes.SignTx(tx, gethTypes.LatestSignerForChainID(chainID), wallet.PrivateKey())
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send the transaction
	err = wallet.Client().SendTransaction(wallet.Ctx(), signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to send transaction: %w", err)
	}

	// Wait for transaction to be mined
	receipt, err = bind.WaitMined(wallet.Ctx(), wallet.GethClient(), signedTx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction to be mined: %w", err)
	}

	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("transaction failed with status %d (tx: %s)", receipt.Status, receipt.TxHash.Hex())
	}

	// Verify the L1 fee parameters were set correctly
	l1BaseFeeScalarActual, err := systemConfig.BasefeeScalar(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 base fee scalar: %w", err)
	}
	if l1BaseFeeScalarActual != l1BaseFeeScalar {
		return nil, fmt.Errorf("l1 base fee scalar mismatch: got %d, expected %d", l1BaseFeeScalarActual, l1BaseFeeScalar)
	}

	blobBaseFeeScalar, err := systemConfig.BlobbasefeeScalar(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		return nil, fmt.Errorf("failed to get l1 blob base fee scalar: %w", err)
	}
	if blobBaseFeeScalar != l1BlobBaseFeeScalar {
		return nil, fmt.Errorf("l1 blob base fee scalar mismatch: got %d, expected %d", blobBaseFeeScalar, l1BlobBaseFeeScalar)
	}

	return receipt, nil
}
