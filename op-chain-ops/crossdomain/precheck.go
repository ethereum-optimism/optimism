package crossdomain

import (
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrUnknownSlotInMessagePasser = errors.New("unknown slot in legacy message passer")
	ErrMissingSlotInWitness       = errors.New("missing storage slot in witness data")
)

// PreCheckWithdrawals checks that the given list of withdrawals represents all withdrawals made
// in the legacy system and filters out any extra withdrawals not included in the legacy system.
func PreCheckWithdrawals(db *state.StateDB, withdrawals DangerousUnfilteredWithdrawals) (SafeFilteredWithdrawals, error) {
	// Convert each withdrawal into a storage slot, and build a map of those slots.
	slotsInp := make(map[common.Hash]*LegacyWithdrawal)
	for _, wd := range withdrawals {
		slot, err := wd.StorageSlot()
		if err != nil {
			return nil, fmt.Errorf("cannot check withdrawals: %w", err)
		}

		slotsInp[slot] = wd
	}

	// Build a mapping of the slots of all messages actually sent in the legacy system.
	var count int
	var innerErr error
	slotsAct := make(map[common.Hash]bool)
	err := db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		// When a message is inserted into the LegacyMessagePasser, it is stored with the value
		// of the ABI encoding of "true". Although there should not be any other storage slots, we
		// can safely ignore anything that is not "true".
		if value != abiTrue {
			// Should not happen!
			innerErr = fmt.Errorf("%w: key: %s, val: %s", ErrUnknownSlotInMessagePasser, key.String(), value.String())
			return true
		}

		// Slot exists, so add it to the map.
		slotsAct[key] = true
		count++
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("cannot iterate over LegacyMessagePasser: %w", err)
	}
	if innerErr != nil {
		return nil, innerErr
	}

	// Log the number of messages we found.
	log.Info("Iterated legacy messages", "count", count)

	// Iterate over the list of actual slots and check that we have an input message for each one.
	for slot := range slotsAct {
		_, ok := slotsInp[slot]
		if !ok {
			return nil, ErrMissingSlotInWitness
		}
	}

	// Iterate over the list of input messages and check that we have a known slot for each one.
	// We'll filter out any extra messages that are not in the legacy system.
	filtered := make(SafeFilteredWithdrawals, 0)
	for slot := range slotsInp {
		_, ok := slotsAct[slot]
		if !ok {
			log.Info("filtering out unknown input message", "slot", slot.String())
			continue
		}

		wd := slotsInp[slot]
		if wd.MessageSender != predeploys.L2CrossDomainMessengerAddr {
			log.Info("filtering out message from sender other than the L2XDM", "sender", wd.MessageSender)
			continue
		}

		filtered = append(filtered, wd)
	}

	// At this point, we know that the list of filtered withdrawals MUST be exactly the same as the
	// list of withdrawals in the state. If we didn't have enough withdrawals, we would've errored
	// out, and if we had too many, we would've filtered them out.
	return filtered, nil
}
