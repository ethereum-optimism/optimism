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

// Constants used by `CrossDomainMessenger.baseGas`
var (
	RelayConstantOverhead            uint64 = 200_000
	RelayPerByteDataCost             uint64 = params.TxDataNonZeroGasEIP2028
	MinGasDynamicOverheadNumerator   uint64 = 64
	MinGasDynamicOverheadDenominator uint64 = 63
	RelayCallOverhead                uint64 = 40_000
	RelayReservedGas                 uint64 = 40_000
	RelayGasCheckBuffer              uint64 = 5_000
)

// MigrateWithdrawals will migrate a list of pending withdrawals given a StateDB.
func MigrateWithdrawals(
	withdrawals SafeFilteredWithdrawals,
	db vm.StateDB,
	l1CrossDomainMessenger *common.Address,
	noCheck bool,
	chainID *big.Int,
) error {
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

		withdrawal, err := MigrateWithdrawal(legacy, l1CrossDomainMessenger, chainID)
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
func MigrateWithdrawal(
	withdrawal *LegacyWithdrawal,
	l1CrossDomainMessenger *common.Address,
	chainID *big.Int,
) (*Withdrawal, error) {
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

	gasLimit := MigrateWithdrawalGasLimit(data, chainID)

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

// MigrateWithdrawalGasLimit computes the gas limit for the migrated withdrawal.
// The chain id is used to determine the overhead.
func MigrateWithdrawalGasLimit(data []byte, chainID *big.Int) uint64 {
	// Compute the upper bound on the gas limit. This could be more
	// accurate if individual 0 bytes and non zero bytes were accounted
	// for.
	dataCost := uint64(len(data)) * RelayPerByteDataCost

	// Goerli has a lower gas limit than other chains.
	var overhead uint64
	if chainID.Cmp(big.NewInt(420)) == 0 {
		overhead = uint64(200_000)
	} else {
		// Mimic `baseGas` from `CrossDomainMessenger.sol`
		overhead = uint64(
			// Constant overhead
			RelayConstantOverhead +
				// Dynamic overhead (EIP-150)
				// We use a constant 1 million gas limit due to the overhead of simulating all migrated withdrawal
				// transactions during the migration. This is a conservative estimate, and if a withdrawal
				// uses more than the minimum gas limit, it will fail and need to be replayed with a higher
				// gas limit.
				(MinGasDynamicOverheadNumerator*1_000_000)/MinGasDynamicOverheadDenominator +
				// Gas reserved for the worst-case cost of 3/5 of the `CALL` opcode's dynamic gas
				// factors. (Conservative)
				RelayCallOverhead +
				// Relay reserved gas (to ensure execution of `relayMessage` completes after the
				// subcontext finishes executing) (Conservative)
				RelayReservedGas +
				// Gas reserved for the execution between the `hasMinGas` check and the `CALL`
				// opcode. (Conservative)
				RelayGasCheckBuffer,
		)
	}

	// Set the outer minimum gas limit. This cannot be zero
	gasLimit := dataCost + overhead

	// Cap the gas limit to be 25 million to prevent creating withdrawals
	// that go over the block gas limit.
	if gasLimit > 25_000_000 {
		gasLimit = 25_000_000
	}

	return gasLimit
}
