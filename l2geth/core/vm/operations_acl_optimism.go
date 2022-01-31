// Copyright 2019 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"errors"

	"github.com/ethereum-optimism/optimism/l2geth/common"
	"github.com/ethereum-optimism/optimism/l2geth/common/math"
	"github.com/ethereum-optimism/optimism/l2geth/params"
)

// These functions are modified to work without the access list logic.
// Access lists will be added in the future and these functions can
// be reverted to their upstream implementations.

// Modified dynamic gas cost to always return the cold cost
func gasSLoadEIP2929Optimism(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	return params.ColdSloadCostEIP2929, nil
}

func gasExtCodeCopyEIP2929Optimism(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	// memory expansion first (dynamic part of pre-2929 implementation)
	gas, err := gasExtCodeCopy(evm, contract, stack, mem, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, params.ColdAccountAccessCostEIP2929-params.WarmStorageReadCostEIP2929); overflow {
		return 0, errors.New("gas uint64 overflow")
	}
	return gas, nil
}

func gasEip2929AccountCheckOptimism(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	return params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929, nil
}

func makeCallVariantGasCallEIP2929Optimism(oldCalculator gasFunc) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		// The WarmStorageReadCostEIP2929 (100) is already deducted in the form of a constant cost, so
		// the cost to charge for cold access, if any, is Cold - Warm
		coldCost := params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
		if !contract.UseGas(coldCost) {
			return 0, ErrOutOfGas
		}
		// Now call the old calculator, which takes into account
		// - create new account
		// - transfer value
		// - memory expansion
		// - 63/64ths rule
		gas, err := oldCalculator(evm, contract, stack, mem, memorySize)
		if err != nil {
			return gas, err
		}
		// In case of a cold access, we temporarily add the cold charge back, and also
		// add it to the returned gas. By adding it to the return, it will be charged
		// outside of this function, as part of the dynamic gas, and that will make it
		// also become correctly reported to tracers.
		contract.Gas += coldCost
		return gas + coldCost, nil
	}
}

var (
	gasCallEIP2929Optimism         = makeCallVariantGasCallEIP2929Optimism(gasCall)
	gasDelegateCallEIP2929Optimism = makeCallVariantGasCallEIP2929Optimism(gasDelegateCall)
	gasStaticCallEIP2929Optimism   = makeCallVariantGasCallEIP2929Optimism(gasStaticCall)
	gasCallCodeEIP2929Optimism     = makeCallVariantGasCallEIP2929Optimism(gasCallCode)
	gasSelfdestructEIP2929Optimism = makeSelfdestructGasFnOptimism(true)
)

// makeSelfdestructGasFn can create the selfdestruct dynamic gas function for EIP-2929 and EIP-2539
func makeSelfdestructGasFnOptimism(refundsEnabled bool) gasFunc {
	gasFunc := func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		address := common.BigToAddress(stack.peek())
		gas := params.ColdAccountAccessCostEIP2929

		// if empty and transfers value
		if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(contract.Address()).Sign() != 0 {
			gas += params.CreateBySelfdestructGas
		}
		if refundsEnabled && !evm.StateDB.HasSuicided(contract.Address()) {
			evm.StateDB.AddRefund(params.SelfdestructRefundGas)
		}
		return gas, nil
	}
	return gasFunc
}
