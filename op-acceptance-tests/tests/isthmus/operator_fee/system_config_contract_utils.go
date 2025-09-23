package operatorfee

// NOTE: These utility functions have been converted from devnet-sdk to op-devstack types
// but are currently unused by tests. They would need implementation updates if used.

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
)

type TestParams struct {
	ID                  string
	OperatorFeeScalar   uint32
	OperatorFeeConstant uint64
	L1BaseFeeScalar     uint32
	L1BlobBaseFeeScalar uint32
}

func GetFeeParamsL1(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA) (tc TestParams, err error) {
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

func GetFeeParamsL2(l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA) (tc TestParams, err error) {
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

func EnsureFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA, tc TestParams) (err error, reset func() error) {
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

func UpdateFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA, tc TestParams) (err error) {
	return fmt.Errorf("not implemented for op-devstack - utility function not used by tests")
}

// UpdateOperatorFeeParams updates the operator fee parameters in the SystemConfig contract.
// It constructs and sends a transaction using txplan and returns the signed transaction, the receipt, or an error.
func UpdateOperatorFeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA, operatorFeeConstant uint64, operatorFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	return nil, fmt.Errorf("not implemented for op-devstack - utility function not used by tests")
}

func UpdateL1FeeParams(systemConfig *bindings.SystemConfig, systemConfigAddress common.Address, l2L1BlockContract *bindings.L1Block, wallet *dsl.EOA, l1BaseFeeScalar uint32, l1BlobBaseFeeScalar uint32) (receipt *gethTypes.Receipt, err error) {
	return nil, fmt.Errorf("not implemented for op-devstack - utility function not used by tests")
}
