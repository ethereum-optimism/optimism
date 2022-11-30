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
)

var (
	abiTrue                      = common.Hash{31: 0x01}
	errLegacyStorageSlotNotFound = errors.New("cannot find storage slot")
)

// MigrateWithdrawals will migrate a list of pending withdrawals given a StateDB.
func MigrateWithdrawals(withdrawals []*LegacyWithdrawal, db vm.StateDB, l1CrossDomainMessenger *common.Address, noCheck bool) error {
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

	versionedNonce := EncodeVersionedNonce(withdrawal.Nonce, common.Big1)
	data, err := abi.Pack(
		"relayMessage",
		versionedNonce,
		withdrawal.Sender,
		withdrawal.Target,
		value,
		new(big.Int),
		withdrawal.Data,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot abi encode relayMessage: %w", err)
	}

	w := NewWithdrawal(
		withdrawal.Nonce,
		&predeploys.L2CrossDomainMessengerAddr,
		l1CrossDomainMessenger,
		value,
		new(big.Int),
		data,
	)
	return w, nil
}
