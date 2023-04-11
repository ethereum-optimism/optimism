package crossdomain

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	abiTrue                      = common.Hash{31: 0x01}
	errLegacyStorageSlotNotFound = errors.New("cannot find storage slot")
)

// MigrateWithdrawals will migrate a list of pending withdrawals given a StateDB.
func MigrateWithdrawals(withdrawals SafeFilteredWithdrawals, db vm.StateDB, l1CrossDomainMessenger *common.Address, noCheck bool) error {
	for i, legacy := range withdrawals {
		legacySlot, err := legacy.StorageSlot()
		if err != nil {
			return err
		}

		if !noCheck {
			legacyValue := db.GetState(predeploys.LegacyMessagePasserAddr, legacySlot)
			if legacyValue != abiTrue {
				return fmt.Errorf("%w: %s", errLegacyStorageSlotNotFound, legacySlot)
			}
		}

		withdrawal, err := MigrateWithdrawal(legacy, l1CrossDomainMessenger)
		if err != nil {
			return err
		}

		slot, err := withdrawal.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute withdrawal storage slot: %w", err)
		}

		db.SetState(predeploys.L2ToL1MessagePasserAddr, slot, abiTrue)
		log.Info("Migrated withdrawal", "number", i, "slot", slot)
	}
	return nil
}

// MigrateWithdrawal will turn a LegacyWithdrawal into a bedrock
// style Withdrawal.
func MigrateWithdrawal(withdrawal *LegacyWithdrawal, l1CrossDomainMessenger *common.Address) (*Withdrawal, error) {
	// Attempt to parse the value
	value, err := withdrawal.Value()
	if err != nil {
		return nil, fmt.Errorf("cannot migrate withdrawal: %w", err)
	}

	abi, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	// Migrated withdrawals are specified as version 0. Both the
	// L2ToL1MessagePasser and the CrossDomainMessenger use the same
	// versioning scheme. Both should be set to version 0
	versionedNonce := EncodeVersionedNonce(withdrawal.XDomainNonce, new(big.Int))
	// Encode the call to `relayMessage` on the `CrossDomainMessenger`.
	// The minGasLimit can safely be 0 here.
	data, err := abi.Pack(
		"relayMessage",
		versionedNonce,
		withdrawal.XDomainSender,
		withdrawal.XDomainTarget,
		value,
		new(big.Int),
		[]byte(withdrawal.XDomainData),
	)
	if err != nil {
		return nil, fmt.Errorf("cannot abi encode relayMessage: %w", err)
	}

	gasLimit := MigrateWithdrawalGasLimit(data)

	w := NewWithdrawal(
		versionedNonce,
		&predeploys.L2CrossDomainMessengerAddr,
		l1CrossDomainMessenger,
		value,
		new(big.Int).SetUint64(gasLimit),
		data,
	)
	return w, nil
}

func MigrateWithdrawalGasLimit(data []byte) uint64 {
	// Compute the cost of the calldata
	dataCost := uint64(0)
	for _, b := range data {
		if b == 0 {
			dataCost += params.TxDataZeroGas
		} else {
			dataCost += params.TxDataNonZeroGasEIP2028
		}
	}

	// Set the outer gas limit. This cannot be zero
	gasLimit := dataCost + 200_000
	// Cap the gas limit to be 25 million to prevent creating withdrawals
	// that go over the block gas limit.
	if gasLimit > 25_000_000 {
		gasLimit = 25_000_000
	}

	return gasLimit
}
