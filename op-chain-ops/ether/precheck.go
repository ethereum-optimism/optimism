package ether

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/util"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// PreCheckBalances checks that the given list of addresses and allowances represents all storage
// slots in the LegacyERC20ETH contract. We don't have to filter out extra addresses like we do for
// withdrawals because we'll simply carry the balance of a given address to the new system, if the
// account is extra then it won't have any balance and nothing will happen.
func PreCheckBalances(ldb ethdb.Database, db *state.StateDB, addresses []common.Address, allowances []*crossdomain.Allowance, chainID int, noCheck bool) ([]common.Address, error) {
	// Chain params to use for integrity checking.
	params := crossdomain.ParamsByChainID[chainID]
	if params == nil {
		return nil, fmt.Errorf("no chain params for %d", chainID)
	}

	// We'll need to maintain a list of all addresses that we've seen along with all of the storage
	// slots based on the witness data.
	addrs := make([]common.Address, 0)
	slotsInp := make(map[common.Hash]int)

	// For each known address, compute its balance key and add it to the list of addresses.
	// Mint events are instrumented as regular ETH events in the witness data, so we no longer
	// need to iterate over mint events during the migration.
	for _, addr := range addresses {
		addrs = append(addrs, addr)
		slotsInp[CalcOVMETHStorageKey(addr)] = 1
	}

	// For each known allowance, compute its storage key and add it to the list of addresses.
	for _, allowance := range allowances {
		addrs = append(addrs, allowance.From)
		slotsInp[CalcAllowanceStorageKey(allowance.From, allowance.To)] = 2
	}

	// Add the old SequencerEntrypoint because someone sent it ETH a long time ago and it has a
	// balance but none of our instrumentation could easily find it. Special case.
	sequencerEntrypointAddr := common.HexToAddress("0x4200000000000000000000000000000000000005")
	addrs = append(addrs, sequencerEntrypointAddr)
	slotsInp[CalcOVMETHStorageKey(sequencerEntrypointAddr)] = 1

	// Build a mapping of every storage slot in the LegacyERC20ETH contract, except the list of
	// slots that we know we can ignore (totalSupply, name, symbol).
	var count int
	slotsAct := make(map[common.Hash]common.Hash)
	progress := util.ProgressLogger(1000, "Read OVM_ETH storage slot")
	err := db.ForEachStorage(predeploys.LegacyERC20ETHAddr, func(key, value common.Hash) bool {
		progress()

		// We can safely ignore specific slots (totalSupply, name, symbol).
		if ignoredSlots[key] {
			return true
		}

		// Slot exists, so add it to the map.
		slotsAct[key] = value
		count++
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("cannot iterate over LegacyERC20ETHAddr: %w", err)
	}

	// Log how many slots were iterated over.
	log.Info("Iterated legacy balances", "count", count)

	// Iterate over the list of known slots and check that we have a slot for each one. We'll also
	// keep track of the total balance to be migrated and throw if the total supply exceeds the
	// expected supply delta.
	totalFound := new(big.Int)
	var unknown bool
	for slot := range slotsAct {
		slotType, ok := slotsInp[slot]
		if !ok {
			if noCheck {
				log.Error("ignoring unknown storage slot in state", "slot", slot.String())
			} else {
				unknown = true
				log.Error("unknown storage slot in state", "slot", slot.String())
				continue
			}
		}

		// Add balances to the total found.
		switch slotType {
		case 1:
			// Balance slot.
			totalFound.Add(totalFound, slotsAct[slot].Big())
		case 2:
			// Allowance slot.
			continue
		default:
			// Should never happen.
			if noCheck {
				log.Error("unknown slot type", "slot", slot, "type", slotType)
			} else {
				log.Crit("unknown slot type: %d", slotType)
			}
		}
	}
	if unknown {
		return nil, errors.New("unknown storage slots in state (see logs for details)")
	}

	// Verify the supply delta. Recorded total supply in the LegacyERC20ETH contract may be higher
	// than the actual migrated amount because self-destructs will remove ETH supply in a way that
	// cannot be reflected in the contract. This is fine because self-destructs just mean the L2 is
	// actually *overcollateralized* by some tiny amount.
	totalSupply := getOVMETHTotalSupply(db)
	delta := new(big.Int).Sub(totalSupply, totalFound)
	if delta.Cmp(params.ExpectedSupplyDelta) != 0 {
		if noCheck {
			log.Error(
				"supply mismatch",
				"migrated", totalFound.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", params.ExpectedSupplyDelta.String(),
			)
		} else {
			log.Crit(
				"supply mismatch",
				"migrated", totalFound.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", params.ExpectedSupplyDelta.String(),
			)
		}
	}

	// Supply is verified.
	log.Info(
		"supply verified OK",
		"migrated", totalFound.String(),
		"supply", totalSupply.String(),
		"delta", delta.String(),
		"exp_delta", params.ExpectedSupplyDelta.String(),
	)

	// We know we have at least a superset of all addresses here since we know that we have every
	// storage slot. It's fine to have extras because they won't have any balance.
	return addrs, nil
}
