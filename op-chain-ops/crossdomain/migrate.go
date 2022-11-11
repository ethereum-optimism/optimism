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
	abiTrue = common.Hash{31: 0x01}
	//errLegacyStorageSlotNotFound = errors.New("cannot find storage slot")
)

// MigrateWithdrawals will migrate a list of pending withdrawals given a StateDB.
func MigrateWithdrawals(withdrawals []*LegacyWithdrawal, db vm.StateDB, l1CrossDomainMessenger, l1StandardBridge *common.Address) error {
	for i, legacy := range withdrawals {
		legacySlot, err := legacy.StorageSlot()
		if err != nil {
			return err
		}

		legacyValue := db.GetState(predeploys.LegacyMessagePasserAddr, legacySlot)
		if legacyValue != abiTrue {
			// TODO: Re-enable this once we have the exact data we need on mainnet.
			// This is disabled because the data file we're using for testing was
			// generated after the database dump, which means that there are extra
			// storage slots in the state that don't show up in the withdrawals list.
			// return fmt.Errorf("%w: %s", errLegacyStorageSlotNotFound, legacySlot)
			continue
		}

		withdrawal, err := MigrateWithdrawal(legacy, l1CrossDomainMessenger, l1StandardBridge)
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
func MigrateWithdrawal(withdrawal *LegacyWithdrawal, l1CrossDomainMessenger, l1StandardBridge *common.Address) (*Withdrawal, error) {
	value := new(big.Int)

	isFromL2StandardBridge := *withdrawal.Sender == predeploys.L2StandardBridgeAddr

	if withdrawal.Target == nil {
		return nil, errors.New("withdrawal target cannot be nil")
	}

	isToL1StandardBridge := *withdrawal.Target == *l1StandardBridge

	if isFromL2StandardBridge && isToL1StandardBridge {
		abi, err := bindings.L1StandardBridgeMetaData.GetAbi()
		if err != nil {
			return nil, err
		}

		method, err := abi.MethodById(withdrawal.Data)
		if err != nil {
			return nil, err
		}
		if method.Name == "finalizeETHWithdrawal" {
			data, err := method.Inputs.Unpack(withdrawal.Data[4:])
			if err != nil {
				return nil, err
			}
			// bounds check
			if len(data) < 3 {
				return nil, errors.New("not enough data")
			}
			var ok bool
			value, ok = data[2].(*big.Int)
			if !ok {
				return nil, errors.New("not big.Int")
			}
		}
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
