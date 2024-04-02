package crossdomain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/bobanetwork/boba/boba-bindings/bindings"
	"github.com/bobanetwork/boba/boba-bindings/predeploys"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/accounts/abi"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/erigon/params"
	"github.com/ledgerwatch/log/v3"
)

var (
	abiTrue                      = common.Hash{31: 0x01}
	errLegacyStorageSlotNotFound = errors.New("cannot find storage slot")
)

// MigrateWithdrawals will migrate a list of pending withdrawals given a genesis.
func MigrateWithdrawals(withdrawals SafeFilteredWithdrawals, g *types.Genesis, l1CrossDomainMessenger *common.Address, noCheck bool) error {
	for i, legacy := range withdrawals {
		legacySlot, err := legacy.StorageSlot()
		if err != nil {
			return err
		}

		if !noCheck {
			legacyValue := common.Hash{}
			if g.Alloc[predeploys.LegacyMessagePasserAddr].Storage != nil {
				legacyValue = g.Alloc[predeploys.LegacyMessagePasserAddr].Storage[legacySlot]
			}
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

		if g.Alloc[predeploys.L2ToL1MessagePasserAddr].Storage == nil {
			g.Alloc[predeploys.L2ToL1MessagePasserAddr] = types.GenesisAccount{
				Constructor: g.Alloc[predeploys.L2ToL1MessagePasserAddr].Constructor,
				Code:        g.Alloc[predeploys.L2ToL1MessagePasserAddr].Code,
				Storage: map[common.Hash]common.Hash{
					slot: abiTrue,
				},
				Balance: g.Alloc[predeploys.L2ToL1MessagePasserAddr].Balance,
				Nonce:   g.Alloc[predeploys.L2ToL1MessagePasserAddr].Nonce,
			}
		} else {
			g.Alloc[predeploys.L2ToL1MessagePasserAddr].Storage[slot] = abiTrue
		}
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

	abi, err := abi.JSON(strings.NewReader(bindings.L1CrossDomainMessengerABI))
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
