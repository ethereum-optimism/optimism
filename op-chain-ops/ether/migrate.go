package ether

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

func MigrateLegacyETH(db ethdb.Database, addresses []common.Address, allowances []*migration.Allowance, chainID int) error {
	// Set of addresses that we will be migrating.
	addressesToMigrate := make(map[common.Address]bool)
	// Set of storage slots that we expect to see in the OVM ETH contract.
	storageSlotsToMigrate := make(map[common.Hash]int)
	// Chain params to use for integrity checking.
	params := ParamsByChainID[chainID]

	// Iterate over each address list, and read the addresses they
	// contain into memory. Also calculate the storage slots for each
	// address.
	for _, addr := range addresses {
		addressesToMigrate[addr] = true
		storageSlotsToMigrate[CalcOVMETHStorageKey(addr)] = 1
	}

	for _, allowance := range allowances {
		addressesToMigrate[allowance.From] = true
		// TODO: double check ordering here
		storageSlotsToMigrate[CalcAllowanceStorageKey(allowance.From, allowance.To)] = 2
	}

	headBlock := rawdb.ReadHeadBlock(db)
	root := headBlock.Root()

	// Read mint events from the database. Even though Geth's balance methods
	// are instrumented, mints from the bridge happen in the EVM and so do
	// not execute that code path. As a result, we parse mint events in order
	// to not miss any balances.
	log.Info("reading mint events from DB")
	logProgress := ProgressLogger(100, "read mint event")
	err := IterateMintEvents(db, headBlock.NumberU64(), func(address common.Address) error {
		addressesToMigrate[address] = true
		storageSlotsToMigrate[CalcOVMETHStorageKey(address)] = 1
		logProgress()
		return nil
	})
	if err != nil {
		return wrapErr(err, "error reading mint events")
	}

	// Make sure all addresses are accounted for by iterating over
	// the OVM ETH contract's state, and panicking if we miss
	// any storage keys. We also keep track of the total amount of
	// OVM ETH found, and diff that against the total supply of
	// OVM ETH specified in the contract.
	backingStateDB := state.NewDatabase(db)
	stateDB, err := state.New(root, backingStateDB, nil)
	if err != nil {
		return wrapErr(err, "error opening state DB")
	}
	storageTrie := stateDB.StorageTrie(OVMETHAddress)
	storageIt := trie.NewIterator(storageTrie.NodeIterator(nil))
	logProgress = ProgressLogger(10000, "iterating storage keys")
	totalFound := new(big.Int)
	totalSupply := getOVMETHTotalSupply(stateDB)
	for storageIt.Next() {
		_, content, _, err := rlp.Split(storageIt.Value)
		if err != nil {
			panic(err)
		}

		k := common.BytesToHash(storageTrie.GetKey(storageIt.Key))
		v := common.BytesToHash(content)
		sType := storageSlotsToMigrate[k]

		switch sType {
		case 1:
			// This slot is a balance, increment totalFound.
			totalFound = totalFound.Add(totalFound, v.Big())
		case 2:
			// This slot is an allowance, ignore it.
			continue
		default:
			slot := new(big.Int).SetBytes(k.Bytes())
			// Check if this slot is a variable. If it isn't, and it isn't a
			// known missing key, abort
			if slot.Cmp(maxSlot) == 1 && !params.KnownMissingKeys[k] {
				log.Crit("missed storage key", "k", k.String(), "v", v.String())
			}
		}

		logProgress()
	}

	// Verify that the total supply is what we expect. We allow a hardcoded
	// delta to be specified in the chain params since older regenesis events
	// had supply bugs.
	delta := new(big.Int).Set(totalSupply)
	delta = delta.Sub(delta, totalFound)
	if delta.Cmp(params.ExpectedSupplyDelta) != 0 {
		log.Crit(
			"supply mismatch",
			"migrated", totalFound,
			"supply", totalSupply,
			"delta", delta,
			"exp_delta", params.ExpectedSupplyDelta,
		)
	}

	log.Info(
		"supply verified OK",
		"migrated", totalFound.String(),
		"supply", totalSupply.String(),
		"delta", delta.String(),
		"exp_delta", params.ExpectedSupplyDelta,
	)

	log.Info("performing migration")

	log.Info("trie dumping started", "root", root)
	tr, err := backingStateDB.OpenTrie(root)
	if err != nil {
		return err
	}
	it := trie.NewIterator(tr.NodeIterator(nil))
	totalMigrated := new(big.Int)
	logAccountProgress := ProgressLogger(1000, "imported accounts")
	migratedAccounts := make(map[common.Address]bool)
	for it.Next() {
		// It's up to us to decode trie data.
		var data types.StateAccount
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}

		addrBytes := tr.GetKey(it.Key)
		addr := common.BytesToAddress(addrBytes)
		migratedAccounts[addr] = true

		// Get the OVM ETH balance based on the address's storage key.
		ovmBalance := getOVMETHBalance(stateDB, addr)

		// No accounts should have a balance in state. If they do, bail.
		if data.Balance.Sign() > 0 {
			log.Crit("account has non-zero balance in state - should never happen", "addr", addr)
		}

		// Actually perform the migration by setting the appropriate values in state.
		stateDB.SetBalance(addr, ovmBalance)
		stateDB.SetState(predeploys.LegacyERC20ETHAddr, CalcOVMETHStorageKey(addr), common.Hash{})

		// Bump the total OVM balance.
		totalMigrated = totalMigrated.Add(totalMigrated, ovmBalance)

		logAccountProgress()
	}

	// Take care of nonce zero accounts with balances. These are accounts
	// that received OVM ETH as part of the regenesis, but never actually
	// transacted on-chain.
	logNonceZeroProgress := ProgressLogger(1000, "imported zero nonce accounts")
	log.Info("importing accounts with zero-nonce balances")
	for addr := range addressesToMigrate {
		if migratedAccounts[addr] {
			continue
		}

		ovmBalance := getOVMETHBalance(stateDB, addr)
		totalMigrated = totalMigrated.Add(totalMigrated, ovmBalance)
		stateDB.AddBalance(addr, ovmBalance)
		stateDB.SetState(predeploys.LegacyERC20ETHAddr, CalcOVMETHStorageKey(addr), common.Hash{})
		logNonceZeroProgress()
	}

	// Make sure that the amount we migrated matches the amount in
	// our original state.
	if totalMigrated.Cmp(totalFound) != 0 {
		log.Crit(
			"total migrated does not equal total OVM eth found",
			"migrated", totalMigrated,
			"found", totalFound,
		)
	}

	// Set the total supply to 0
	stateDB.SetState(predeploys.LegacyERC20ETHAddr, getOVMETHTotalSupplySlot(), common.Hash{})

	return nil
}
