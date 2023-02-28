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

	ignoredSlots = map[common.Hash]bool{
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

func MigrateLegacyETH(db *state.StateDB, addresses []common.Address, chainID int, noCheck bool) error {
	// Chain params to use for integrity checking.
	params := crossdomain.ParamsByChainID[chainID]
	if params == nil {
		return fmt.Errorf("no chain params for %d", chainID)
	}

	// Log the chain params for debugging purposes.
	log.Info("Chain params", "chain-id", chainID, "supply-delta", params.ExpectedSupplyDelta)

	// Deduplicate the list of addresses by converting to a map.
	deduped := make(map[common.Address]bool)
	for _, addr := range addresses {
		deduped[addr] = true
	}

	// Migrate the legacy ETH to ETH.
	log.Info("Migrating legacy ETH to ETH", "num-accounts", len(addresses))
	totalMigrated := new(big.Int)
	logAccountProgress := util.ProgressLogger(1000, "imported accounts")
	for addr := range deduped {
		// No accounts should have a balance in state. If they do, bail.
		if db.GetBalance(addr).Sign() > 0 {
			if noCheck {
				log.Error("account has non-zero balance in state - should never happen", "addr", addr)
			} else {
				log.Crit("account has non-zero balance in state - should never happen", "addr", addr)
			}
		}

		// Pull out the OVM ETH balance.
		ovmBalance := getOVMETHBalance(db, addr)

		// Actually perform the migration by setting the appropriate values in state.
		db.SetBalance(addr, ovmBalance)
		db.SetState(predeploys.LegacyERC20ETHAddr, CalcOVMETHStorageKey(addr), common.Hash{})

		// Bump the total OVM balance.
		totalMigrated = totalMigrated.Add(totalMigrated, ovmBalance)

		// Log progress.
		logAccountProgress()
	}

	// Make sure that the total supply delta matches the expected delta. This is equivalent to
	// checking that the total migrated is equal to the total found, since we already performed the
	// same check against the total found (a = b, b = c => a = c).
	totalSupply := getOVMETHTotalSupply(db)
	delta := new(big.Int).Sub(totalSupply, totalMigrated)
	if delta.Cmp(params.ExpectedSupplyDelta) != 0 {
		if noCheck {
			log.Error(
				"supply mismatch",
				"migrated", totalMigrated.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", params.ExpectedSupplyDelta.String(),
			)
		} else {
			log.Crit(
				"supply mismatch",
				"migrated", totalMigrated.String(),
				"supply", totalSupply.String(),
				"delta", delta.String(),
				"exp_delta", params.ExpectedSupplyDelta.String(),
			)
		}
	}

	// Set the total supply to 0. We do this because the total supply is necessarily going to be
	// different than the sum of all balances since we no longer track balances inside the contract
	// itself. The total supply is going to be weird no matter what, might as well set it to zero
	// so it's explicitly weird instead of implicitly weird.
	db.SetState(predeploys.LegacyERC20ETHAddr, getOVMETHTotalSupplySlot(), common.Hash{})
	log.Info("Set the totalSupply to 0")

	// Fin.
	return nil
}
