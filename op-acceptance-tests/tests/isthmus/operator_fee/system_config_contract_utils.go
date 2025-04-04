package operatorfee

import (
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system"
	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/isthmus"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/lmittmann/w3"
)

var l1ConfigSyncPollInterval = 30 * time.Second
var l1ConfigSyncMaxWaitTime = 4 * time.Minute

type TestParams struct {
	ID                  string
	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
	L1BaseFeeScalar     uint32
	L1BlobBaseFeeScalar uint32
}

func GetFeeParamsL1(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet system.WalletV2) (tc TestParams, err error) {
	operatorFeeConstant, err := systemConfig.OperatorFeeConstant(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get operator fee constant: %w", err)
	}
	operatorFeeScalar, err := systemConfig.OperatorFeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get operator fee scalar: %w", err)
	}
	l1BaseFeeScalar, err := systemConfig.BasefeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get l1 base fee scalar: %w", err)
	}
	l1BlobBaseFeeScalar, err := systemConfig.BlobbasefeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get l1 blob base fee scalar: %w", err)
	}
	return TestParams{
		OperatorFeeConstant: operatorFeeConstant,
		OperatorFeeScalar:   operatorFeeScalar,
		L1BaseFeeScalar:     l1BaseFeeScalar,
		L1BlobBaseFeeScalar: l1BlobBaseFeeScalar,
	}, nil
}

func GetFeeParamsL2(l2L1BlockContract *bindings.L1Block, wallet system.WalletV2) (tc TestParams, err error) {
	operatorFeeConstant, err := l2L1BlockContract.OperatorFeeConstant(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get operator fee constant: %w", err)
	}
	operatorFeeScalar, err := l2L1BlockContract.OperatorFeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get operator fee scalar: %w", err)
	}
	l1BaseFeeScalar, err := l2L1BlockContract.BaseFeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get l1 base fee scalar: %w", err)
	}
	l1BlobBaseFeeScalar, err := l2L1BlockContract.BlobBaseFeeScalar(&bind.CallOpts{BlockNumber: nil})
	if err != nil {
		return TestParams{}, fmt.Errorf("failed to get l1 blob base fee scalar: %w", err)
	}
	return TestParams{
		OperatorFeeConstant: operatorFeeConstant,
		OperatorFeeScalar:   operatorFeeScalar,
		L1BaseFeeScalar:     l1BaseFeeScalar,
		L1BlobBaseFeeScalar: l1BlobBaseFeeScalar,
	}, nil
}

func EnsureFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet system.WalletV2, tc TestParams) (err error, reset func() error) {
	preFeeParams, err := GetFeeParamsL1(systemConfig, systemConfigAddress, l2L1BlockContract, wallet)
	if err != nil {
		return fmt.Errorf("failed to get L1 fee parameters: %w", err), nil
	}
	preFeeParams.ID = tc.ID

	if preFeeParams == tc {
		// No need to update
		return nil, nil
	}

	return UpdateFeeParams(systemConfig, systemConfigAddress, l2L1BlockContract, wallet, tc), func() error {
		return UpdateFeeParams(systemConfig, systemConfigAddress, l2L1BlockContract, wallet, preFeeParams)
	}
}

func UpdateFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet system.WalletV2, tc TestParams) (err error) {

	_, err = UpdateOperatorFeeParams(systemConfig, systemConfigAddress, l2L1BlockContract, wallet, tc.OperatorFeeConstant, tc.OperatorFeeScalar)
	if err != nil {
		return fmt.Errorf("failed to update operator fee parameters: %w", err)
	}

	_, err = UpdateL1FeeParams(systemConfig, systemConfigAddress, l2L1BlockContract, wallet, tc.L1BaseFeeScalar, tc.L1BlobBaseFeeScalar)
	if err != nil {
		return fmt.Errorf("failed to update L1 fee parameters: %w", err)
	}

	// Wait for L2 nodes to sync with L1 origin where fee parameters were set
	deadline := time.Now().Add(l1ConfigSyncMaxWaitTime)

	for time.Now().Before(deadline) {

		l2FeeParams, err := GetFeeParamsL2(l2L1BlockContract, wallet)
		if err != nil {
			return fmt.Errorf("failed to get L2 fee parameters: %w", err)
		}
		l2FeeParams.ID = tc.ID

		// Check if all values match expected values
		if l2FeeParams == tc {
			break
		}

		// Use context-aware sleep
		select {
		case <-time.After(l1ConfigSyncPollInterval):
			// Continue with next iteration
		case <-wallet.Ctx().Done():
			return fmt.Errorf("context canceled while waiting for L2 nodes to sync: %w", wallet.Ctx().Err())
		}

		// Check if context is canceled
		if wallet.Ctx().Err() != nil {
			return fmt.Errorf("context canceled while waiting for L2 nodes to sync: %w", wallet.Ctx().Err())
		}
	}
	return nil
}

// UpdateOperatorFeeParams updates the operator fee parameters in the SystemConfig contract.
// It constructs and sends a transaction using txplan and returns the signed transaction, the receipt, or an error.
func UpdateOperatorFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet system.WalletV2, operatorFeeConstant uint64, operatorFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	// Construct call input
	funcSetOperatorFeeScalars := w3.MustNewFunc(`setOperatorFeeScalars(uint32, uint64)`, "")
	args, err := funcSetOperatorFeeScalars.EncodeArgs(
		operatorFeeScalar,
		operatorFeeConstant,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments for setOperatorFeeScalars: %w", err)
	}

	// Create a transaction using txplan
	opts := isthmus.DefaultTxOpts(wallet)
	ptx := txplan.NewPlannedTx(
		opts,
		txplan.WithTo(&systemConfigAddress),
		txplan.WithData(args),
	)

	_, err = ptx.Success.Eval(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("tx failed: %w", err)
	}

	// Execute the transaction and wait for inclusion
	receipt = ptx.Included.Value()

	actualOperatorFeeConstant, err := systemConfig.OperatorFeeConstant(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator fee constant: %w", err)
	}
	if operatorFeeConstant != actualOperatorFeeConstant {
		return nil, fmt.Errorf("operator fee constant mismatch: got %d, expected %d", actualOperatorFeeConstant, operatorFeeConstant)
	}

	actualOperatorFeeScalar, err := systemConfig.OperatorFeeScalar(&bind.CallOpts{BlockNumber: receipt.BlockNumber})
	if err != nil {
		return nil, fmt.Errorf("failed to get operator fee scalar: %w", err)
	}
	if operatorFeeScalar != actualOperatorFeeScalar {
		return nil, fmt.Errorf("operator fee scalar mismatch: got %d, expected %d", actualOperatorFeeScalar, operatorFeeScalar)
	}

	return receipt, nil
}

func UpdateL1FeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet system.WalletV2, l1BaseFeeScalar uint32, l1BlobBaseFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	// Construct call input
	funcSetGasConfigEcotone := w3.MustNewFunc(`setGasConfigEcotone(uint32 _basefeeScalar, uint32 _blobbasefeeScalar)`, "")
	args, err := funcSetGasConfigEcotone.EncodeArgs(
		l1BaseFeeScalar,
		l1BlobBaseFeeScalar,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to encode arguments for setGasConfigEcotone: %w", err)
	}

	// Create a transaction using txplan
	opts := isthmus.DefaultTxOpts(wallet)
	ptx := txplan.NewPlannedTx(
		opts,
		txplan.WithTo(&systemConfigAddress),
		txplan.WithData(args),
	)

	_, err = ptx.Success.Eval(wallet.Ctx())
	if err != nil {
		return nil, fmt.Errorf("tx failed: %w", err)
	}

	// Execute the transaction and wait for inclusion
	receipt = ptx.Included.Value()

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
