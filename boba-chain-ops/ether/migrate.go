package ether

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/crossdomain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/util"
	"github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/log/v3"
	"golang.org/x/sync/errgroup"
)

const (
	// BalanceSlot is an ordinal used to represent slots corresponding to OVM_ETH
	// balances in the state.
	BalanceSlot = 1

	// AllowanceSlot is an ordinal used to represent slots corresponding to OVM_ETH
	// allowances in the state.
	AllowanceSlot = 2
)

var (
	// OVMETHAddress is the address of the OVM ETH predeploy.
	OVMETHAddress = common.HexToAddress("0x4200000000000000000000000000000000000006")

	ignoredSlots = map[common.Hash]bool{
		// Total Supply
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"): true,
		// Name
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000003"): true,
		// Symbol
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000004"): true,
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000005"): true,
		common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000006"): true,
		// To avoid the loop import. The amdin slot is hardcoded here as the same in helpers.go
		common.HexToHash("0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103"): true,
	}

	// sequencerEntrypointAddr is the address of the OVM sequencer entrypoint contract.
	sequencerEntrypointAddr = common.HexToAddress("0x4200000000000000000000000000000000000005")
)

// accountData is a wrapper struct that contains the balance and address of an account.
// It gets passed via channel to the collector process.
type accountData struct {
	balance    *big.Int
	legacySlot common.Hash
	address    common.Address
}

// MigrateBalances migrates all balances in the LegacyERC20ETH contract into state. It performs checks
// in parallel with mutations in order to reduce overall migration time.
func MigrateBalances(g *types.Genesis, addresses []common.Address, allowances []*crossdomain.Allowance, noCheck bool) error {
	// Chain params to use for integrity checking.
	chainID := int(g.Config.ChainID.Uint64())
	params := crossdomain.ParamsByChainID[chainID]
	if params == nil {
		return fmt.Errorf("no chain params for %d", chainID)
	}

	return doMigration(g, addresses, allowances, params.ExpectedSupplyDelta, noCheck)
}

func doMigration(g *types.Genesis, addresses []common.Address, allowances []*crossdomain.Allowance, expDiff *big.Int, noCheck bool) error {
	m := &sync.Mutex{}

	// We'll need to maintain a list of all addresses that we've seen along with all of the storage
	// slots based on the witness data.
	slotsAddrs := make(map[common.Hash]common.Address)
	slotsInp := make(map[common.Hash]int)

	// For each known address, compute its balance key and add it to the list of addresses.
	// Mint events are instrumented as regular ETH events in the witness data, so we no longer
	// need to iterate over mint events during the migration.
	for _, addr := range addresses {
		sk := CalcOVMETHStorageKey(addr)
		slotsAddrs[sk] = addr
		slotsInp[sk] = BalanceSlot
	}

	// For each known allowance, compute its storage key and add it to the list of addresses.
	for _, allowance := range allowances {
		sk := CalcAllowanceStorageKey(allowance.From, allowance.To)
		slotsAddrs[sk] = allowance.From
		slotsInp[sk] = AllowanceSlot
	}

	// Add the old SequencerEntrypoint because someone sent it ETH a long time ago and it has a
	// balance but none of our instrumentation could easily find it. Special case.
	entrySK := CalcOVMETHStorageKey(sequencerEntrypointAddr)
	slotsAddrs[entrySK] = sequencerEntrypointAddr
	slotsInp[entrySK] = BalanceSlot

	// Channel to receive storage slot keys and values from each iteration job.
	outCh := make(chan accountData)

	// Channel that gets closed when the collector is done.
	doneCh := make(chan struct{})

	// Create a map of accounts we've seen so that we can filter out duplicates.
	seenAccounts := make(map[common.Address]bool)

	// Keep track of the total migrated supply.
	totalFound := new(big.Int)

	// Kick off a background process to collect
	// values from the channel and add them to the map.
	var count int
	progress := util.ProgressLogger(1000, "Migrated OVM_ETH storage slot")
	go func() {
		defer func() { doneCh <- struct{}{} }()

		for account := range outCh {
			m.Lock()
			progress()

			// Filter out duplicate accounts. See the below note about keyspace iteration for
			// why we may have to filter out duplicates.
			if seenAccounts[account.address] {
				log.Info("skipping duplicate account during iteration", "addr", account.address)
				continue
			}

			// Accumulate addresses and total supply.
			totalFound = new(big.Int).Add(totalFound, account.balance)

			SetBalance(g, account.address, account.balance, account.legacySlot)

			count++
			seenAccounts[account.address] = true
			m.Unlock()
		}
	}()

	err := IterateState(g, func(key, value common.Hash) error {
		// We can safely ignore specific slots (totalSupply, name, symbol).
		if ignoredSlots[key] {
			return nil
		}

		slotType, ok := slotsInp[key]
		if !ok {
			log.Error("unknown storage slot in state", "slot", key.String())
			if !noCheck {
				return fmt.Errorf("unknown storage slot in state: %s", key.String())
			}
		}

		// No accounts should have a balance in state. If they do, bail.
		addr, ok := slotsAddrs[key]
		if !ok {
			log.Crit("could not find address in map - should never happen")
		}

		m.Lock()
		bal := GetBalance(g, addr)
		m.Unlock()

		if bal.Sign() != 0 {
			log.Error(
				"account has non-zero balance in state - should never happen",
				"addr", addr,
				"balance", bal.String(),
			)
			if !noCheck {
				return fmt.Errorf("account has non-zero balance in state - should never happen: %s", addr.String())
			}
		}

		// Add balances to the total found.
		switch slotType {
		case BalanceSlot:
			// Send the data to the channel.
			outCh <- accountData{
				balance:    value.Big(),
				legacySlot: key,
				address:    addr,
			}
		case AllowanceSlot:
			// Allowance slot. Do nothing here.
		default:
			// Should never happen.
			if noCheck {
				log.Error("unknown slot type", "slot", key, "type", slotType)
				return nil
			} else {
				log.Error("unknown slot type %d, should never happen", slotType)
				return fmt.Errorf("unknown slot type %d, should never happen", slotType)
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Close the outCh to cancel the collector. The collector will signal that it's done
	// using doneCh. Any values waiting to be read from outCh will be read before the
	// collector exits.
	close(outCh)
	<-doneCh

	// Log how many slots were iterated over.
	log.Info("Iterated legacy balances", "count", count)

	// Verify the supply delta. Recorded total supply in the LegacyERC20ETH contract may be higher
	// than the actual migrated amount because self-destructs will remove ETH supply in a way that
	// cannot be reflected in the contract. This is fine because self-destructs just mean the L2 is
	// actually *overcollateralized* by some tiny amount.
	totalSupply := g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()].Big()
	delta := new(big.Int).Sub(totalSupply, totalFound)
	if delta.Cmp(expDiff) != 0 {
		log.Error(
			"supply mismatch",
			"migrated", totalFound.String(),
			"supply", totalSupply.String(),
			"delta", delta.String(),
			"exp_delta", expDiff.String(),
		)
		if !noCheck {
			return fmt.Errorf("supply mismatch: %s", delta.String())
		}
	}

	// Supply is verified.
	log.Info(
		"supply verified OK",
		"migrated", totalFound.String(),
		"supply", totalSupply.String(),
		"delta", delta.String(),
		"exp_delta", expDiff.String(),
	)

	// Set the total supply to 0. We do this because the total supply is necessarily going to be
	// different than the sum of all balances since we no longer track balances inside the contract
	// itself. The total supply is going to be weird no matter what, might as well set it to zero
	// so it's explicitly weird instead of implicitly weird.
	SetTotalSupply(g)
	log.Info("Set the totalSupply to 0")

	return nil
}

func IterateState(g *types.Genesis, cb func(key, value common.Hash) error) error {
	// Deep copy
	storage := make(map[common.Hash]common.Hash)
	for key, value := range g.Alloc[predeploys.LegacyERC20ETHAddr].Storage {
		storage[key] = value
	}

	var eg errgroup.Group
	for key, value := range storage {
		innerKey, innerValue := key, value
		eg.Go(func() error {
			return cb(innerKey, innerValue)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}

func SetBalance(g *types.Genesis, addr common.Address, balance *big.Int, slot common.Hash) {
	// Set the balance to balance field
	accountState := g.Alloc[addr]
	accountState.Balance = balance
	g.Alloc[addr] = accountState
	// Remove the balance from OVM_ETH storage
	OVM_ETHStorage := g.Alloc[predeploys.LegacyERC20ETHAddr].Storage
	delete(OVM_ETHStorage, slot)
}

func SetTotalSupply(g *types.Genesis) {
	g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[CalcOVMETHTotalSupplyKey()] = common.Hash{}
}

func GetBalance(g *types.Genesis, addr common.Address) *big.Int {
	if _, ok := g.Alloc[addr]; !ok {
		return new(big.Int)
	}
	return g.Alloc[addr].Balance
}
