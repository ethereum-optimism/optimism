package ether

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/util"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// OVMETHAddress is the address of the OVM ETH predeploy.
	OVMETHAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")

	OVMETHIgnoredSlots = map[common.Hash]bool{
		// Total Supply
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"): true,
		// Name
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"): true,
		// Symbol
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"): true,
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"): true,
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000006"): true,
	}
)

// MigrateLegacyETH checks that the given list of addresses and allowances represents all storage
// slots in the LegacyERC20ETH contract. We don't have to filter out extra addresses like we do for
// withdrawals because we'll simply carry the balance of a given address to the new system, if the
// account is extra then it won't have any balance and nothing will happen. For each valid balance,
// this method will migrate into state. This method does the checking as part of the migration loop
// in order to avoid having to iterate over state twice. This saves approximately 40 minutes during
// the mainnet migration.
func MigrateLegacyETH(db *state.StateDB, addresses []common.Address, allowances []*crossdomain.Allowance, chainID int, noCheck bool, commit bool) error {
	// Chain params to use for integrity checking.
	params := crossdomain.ParamsByChainID[chainID]
	if params == nil {
		return fmt.Errorf("no chain params for %d", chainID)
	}

	// Log the chain params for debugging purposes.
	log.Info("Chain params", "chain-id", chainID, "supply-delta", params.ExpectedSupplyDelta)

	return doMigration(db, addresses, allowances, params.ExpectedSupplyDelta, noCheck, commit)
}

func doMigration(db *state.StateDB, addresses []common.Address, allowances []*crossdomain.Allowance, expSupplyDiff *big.Int, noCheck bool, commit bool) error {
	// We'll need to maintain a list of all addresses that we've seen along with all of the storage
	// slots based on the witness data.
	slotsAddrs := make(map[common.Hash]common.Address)
	slotTypes := make(map[common.Hash]int)

	// For each known address, compute its balance key and add it to the list of addresses.
	// Mint events are instrumented as regular ETH events in the witness data, so we no longer
	// need to iterate over mint events during the migration.
	for _, addr := range addresses {
		sk := CalcOVMETHStorageKey(addr)
		slotTypes[sk] = 1
		slotsAddrs[sk] = addr
	}

	// For each known allowance, compute its storage key and add it to the list of addresses.
	for _, allowance := range allowances {
		slotTypes[CalcAllowanceStorageKey(allowance.From, allowance.To)] = 2
	}

	// Add the old SequencerEntrypoint because someone sent it ETH a long time ago and it has a
	// balance but none of our instrumentation could easily find it. Special case.
	sequencerEntrypointAddr := common.HexToAddress("0x4200000000000000000000000000000000000005")
	slotTypes[CalcOVMETHStorageKey(sequencerEntrypointAddr)] = 1

	// Migrate the OVM_ETH to ETH.
	log.Info("Migrating legacy ETH to ETH", "num-accounts", len(addresses))
	totalMigrated := new(big.Int)
	logAccountProgress := util.ProgressLogger(1000, "imported OVM_ETH storage slot")
	var innerErr error
	err := db.ForEachStorage(predeploys.LegacyERC20ETHAddr, func(key, value common.Hash) bool {
		defer logAccountProgress()

		// We can safely ignore specific slots (totalSupply, name, symbol).
		if OVMETHIgnoredSlots[key] {
			return true
		}

		// Look up the slot type.
		slotType, ok := slotTypes[key]
		if !ok {
			log.Error("unknown storage slot in state", "slot", key.String())
			if !noCheck {
				innerErr = fmt.Errorf("unknown storage slot in state: %s", key.String())
				return false
			}
		}

		switch slotType {
		case 1:
			// Balance slot.
			bal := value.Big()
			totalMigrated.Add(totalMigrated, bal)
			addr := slotsAddrs[key]

			// There should never be any balances in state, so verify that here.
			if db.GetBalance(addr).Sign() > 0 {
				log.Error("account has non-zero balance in state - should never happen", "addr", addr)
				if !noCheck {
					innerErr = fmt.Errorf("account has non-zero balance in state - should never happen: %s", addr)
					return false
				}
			}

			if !commit {
				return true
			}

			// Set the balance, and delete the legacy slot.
			db.SetBalance(addr, bal)
			db.SetState(predeploys.LegacyERC20ETHAddr, key, common.Hash{})
		case 2:
			// Allowance slot. Nothing to do here.
			return true
		default:
			// Should never happen.
			log.Error("unknown slot type", "slot", key.String(), "type", slotType)
			if !noCheck {
				innerErr = fmt.Errorf("unknown slot type: %d", slotType)
				return false
			}
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("failed to iterate over OVM_ETH storage: %w", err)
	}
	if innerErr != nil {
		return fmt.Errorf("error in migration: %w", innerErr)
	}

	// Make sure that the total supply delta matches the expected delta. This is equivalent to
	// checking that the total migrated is equal to the total found, since we already performed the
	// same check against the total found (a = b, b = c => a = c).
	totalSupply := getOVMETHTotalSupply(db)
	delta := new(big.Int).Sub(totalSupply, totalMigrated)
	if delta.Cmp(expSupplyDiff) != 0 {
		if noCheck {
			log.Error(
				"supply mismatch",
				"migrated", totalMigrated.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", expSupplyDiff.String(),
			)
		} else {
			log.Error(
				"supply mismatch",
				"migrated", totalMigrated.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", expSupplyDiff.String(),
			)
			return fmt.Errorf("supply mismatch: exp delta %s != %s", expSupplyDiff.String(), delta.String())
		}
	}

	// Supply is verified.
	log.Info(
		"supply verified OK",
		"migrated", totalMigrated.String(),
		"supply", totalSupply.String(),
		"delta", delta.String(),
		"exp_delta", expSupplyDiff.String(),
	)

	// Set the total supply to 0. We do this because the total supply is necessarily going to be
	// different than the sum of all balances since we no longer track balances inside the contract
	// itself. The total supply is going to be weird no matter what, might as well set it to zero
	// so it's explicitly weird instead of implicitly weird.
	if commit {
		db.SetState(predeploys.LegacyERC20ETHAddr, getOVMETHTotalSupplySlot(), common.Hash{})
		log.Info("Set the totalSupply to 0")
	}

	return nil
}
