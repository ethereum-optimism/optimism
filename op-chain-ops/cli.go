package state_surgery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// OVMETHAddress is the address of the OVM ETH predeploy.
	OVMETHAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")

	// maxSlot is the maximum slot we'll consider to be a non-mapping variable.
	maxSlot = new(big.Int).SetUint64(256)

	emptyCodeHash = crypto.Keccak256(nil)
)

// DumpAddresses dumps address preimages in Geth's database to disk.
func DumpAddresses(dataDir string, outFile string) error {
	db := MustOpenDB(dataDir)
	f, err := os.Create(outFile)
	if err != nil {
		return wrapErr(err, "error opening outfile")
	}

	logProgress := ProgressLogger(1000, "dumped addresses")
	return IterateDBAddresses(db, func(address common.Address) error {
		_, err := f.WriteString(address.Hex() + "\n")
		if err != nil {
			return wrapErr(err, "error writing outfile")
		}
		logProgress()
		return nil
	})
}

// Migrate performs the actual state migration. It does quite a lot:
//
//  1. It uses address lists, allowance lists, Mint events, and address preimages in
//     the input state database to create a comprehensive list of storage slots in the
//     OVM ETH contract.
//  2. It iterates over the slots in OVM ETH, and compares then against the list in (1).
//     If the list doesn't match, or the total supply of OVM ETH doesn't match the sum of
//     all balance storage slots, it panics.
//  3. It performs the actual migration by copying the input state DB into a new state DB.
//  4. It imports the provided genesis into the new state DB like Geth would during geth init.
//
// It takes the following arguments:
//
//   - dataDir:        A Geth data dir.
//   - outDir:         A directory to output the migrated database to.
//   - genesis:        The new chain's genesis configuration.
//   - addrLists:      A list of address list file paths. These address lists are used to populate
//     balances from previous regenesis events.
//   - allowanceLists: A list of allowance list file paths. These allowance lists are used
//     to calculate allowance storage slots from previous regenesis events.
//   - chainID:        The chain ID of the chain being migrated.
func Migrate(dataDir, outDir string, genesis *core.Genesis, addrLists, allowanceLists []string, chainID, levelDBCacheSize, levelDBHandles int) error {
	db := MustOpenDBWithCacheOpts(dataDir, levelDBCacheSize, levelDBHandles)
	// Set of addresses that we will be migrating.
	addressesToMigrate := make(map[common.Address]bool)
	// Set of storage slots that we expect to see in the OVM ETH contract.
	storageSlotsToMigrate := make(map[common.Hash]int)
	// Chain params to use for integrity checking.
	params := ParamsByChainID[chainID]

	// Iterate over each address list, and read the addresses they
	// contain into memory. Also calculate the storage slots for each
	// address.
	for _, list := range addrLists {
		log.Info("reading address list", "list", list)
		f, err := os.Open(list)
		if err != nil {
			return wrapErr(err, "error opening address list %s", list)
		}
		logProgress := ProgressLogger(10000, "read address")
		err = IterateAddrList(f, func(address common.Address) error {
			addressesToMigrate[address] = true
			storageSlotsToMigrate[CalcOVMETHStorageKey(address)] = 1
			logProgress()
			return nil
		})
		f.Close()
		if err != nil {
			return wrapErr(err, "error reading address list")
		}
	}

	// Same as above, but for allowances.
	for _, list := range allowanceLists {
		log.Info("reading allowance list", "list", list)
		f, err := os.Open(list)
		if err != nil {
			return wrapErr(err, "error opening allowances list %s", list)
		}
		logProgress := ProgressLogger(10000, "read allowance")
		err = IterateAllowanceList(f, func(owner, spender common.Address) error {
			addressesToMigrate[owner] = true
			storageSlotsToMigrate[CalcAllowanceStorageKey(owner, spender)] = 2
			logProgress()
			return nil
		})
		f.Close()
		if err != nil {
			return wrapErr(err, "error reading allowances list")
		}
	}

	// Now, read address preimages from the database.
	log.Info("reading addresses from DB")
	logProgress := ProgressLogger(10000, "read address")
	err := IterateDBAddresses(db, func(address common.Address) error {
		addressesToMigrate[address] = true
		storageSlotsToMigrate[CalcOVMETHStorageKey(address)] = 1
		logProgress()
		return nil
	})
	if err != nil {
		return wrapErr(err, "error reading addressesToMigrate from DB")
	}

	headBlock := rawdb.ReadHeadBlock(db)
	root := headBlock.Root()

	// Read mint events from the database. Even though Geth's balance methods
	// are instrumented, mints from the bridge happen in the EVM and so do
	// not execute that code path. As a result, we parse mint events in order
	// to not miss any balances.
	log.Info("reading mint events from DB")
	logProgress = ProgressLogger(100, "read mint event")
	err = IterateMintEvents(db, headBlock.NumberU64(), func(address common.Address) error {
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

	outDB := MustOpenDB(outDir)
	outStateDB, err := state.New(common.Hash{}, state.NewDatabase(outDB), nil)
	if err != nil {
		return wrapErr(err, "error opening output state DB")
	}

	// Iterate over the Genesis allocation accounts. These will override
	// any accounts found in the state.
	log.Info("importing allocated accounts")
	logAllocProgress := ProgressLogger(1000, "allocated accounts")
	for addr, account := range genesis.Alloc {
		outStateDB.SetBalance(addr, account.Balance)
		outStateDB.SetCode(addr, account.Code)
		outStateDB.SetNonce(addr, account.Nonce)
		for key, value := range account.Storage {
			outStateDB.SetState(addr, key, value)
		}
		logAllocProgress()
	}

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

		// Skip genesis addressesToMigrate.
		if _, ok := genesis.Alloc[addr]; ok {
			logAccountProgress()
			continue
		}

		// Skip OVM ETH, though it will probably be put in the genesis. This is here as a fallback
		// in case we don't.
		if addr == OVMETHAddress {
			logAccountProgress()
			continue
		}

		addrHash := crypto.Keccak256Hash(addr[:])
		code := getCode(addrHash, data, backingStateDB)
		// Get the OVM ETH balance based on the address's storage key.
		ovmBalance := getOVMETHBalance(stateDB, addr)

		// No accounts should have a balance in state. If they do, bail.
		if data.Balance.Sign() > 0 {
			log.Crit("account has non-zero balance in state - should never happen", "addr", addr)
		}

		// Actually perform the migration by setting the appropriate values in state.
		outStateDB.SetBalance(addr, ovmBalance)
		outStateDB.SetCode(addr, code)
		outStateDB.SetNonce(addr, data.Nonce)

		// Bump the total OVM balance.
		totalMigrated = totalMigrated.Add(totalMigrated, ovmBalance)

		// Grab the storage trie.
		storageTrie, err := backingStateDB.OpenStorageTrie(addrHash, data.Root)
		if err != nil {
			return wrapErr(err, "error opening storage trie")
		}
		storageIt := trie.NewIterator(storageTrie.NodeIterator(nil))
		logStorageProgress := ProgressLogger(10000, fmt.Sprintf("imported storage slots for %s", addr))
		for storageIt.Next() {
			_, content, _, err := rlp.Split(storageIt.Value)
			if err != nil {
				panic(err)
			}

			// Update each storage slot for this account in state.
			outStateDB.SetState(
				addr,
				common.BytesToHash(storageTrie.GetKey(storageIt.Key)),
				common.BytesToHash(content),
			)

			logStorageProgress()
		}

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
		outStateDB.AddBalance(addr, ovmBalance)
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

	log.Info("committing state DB")
	newRoot, err := outStateDB.Commit(false)
	if err != nil {
		return wrapErr(err, "error writing output state DB")
	}
	log.Info("committed state DB", "root", newRoot)
	log.Info("committing trie DB")
	if err := outStateDB.Database().TrieDB().Commit(newRoot, true, nil); err != nil {
		return wrapErr(err, "error writing output trie DB")
	}
	log.Info("committed trie DB")

	// Now that the state is dumped, insert the genesis block.
	//
	// Unlike regular Geth (which panics if you try to import a genesis state with a nonzero
	// block number), the block number can be anything.
	block := genesis.ToBlock()

	// Geth block headers are immutable, so swap the root and make a new block with the
	// updated root.
	header := block.Header()
	header.Root = newRoot
	block = types.NewBlock(header, nil, nil, nil, trie.NewStackTrie(nil))
	blob, err := json.Marshal(genesis)
	if err != nil {
		log.Crit("error marshaling genesis state", "err", err)
	}

	// Write the genesis state to the database. This is taken verbatim from Geth's
	// core.Genesis struct.
	rawdb.WriteGenesisStateSpec(outDB, block.Hash(), blob)
	rawdb.WriteTd(outDB, block.Hash(), block.NumberU64(), block.Difficulty())
	rawdb.WriteBlock(outDB, block)
	rawdb.WriteReceipts(outDB, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(outDB, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(outDB, block.Hash())
	rawdb.WriteHeadFastBlockHash(outDB, block.Hash())
	rawdb.WriteHeadHeaderHash(outDB, block.Hash())
	rawdb.WriteChainConfig(outDB, block.Hash(), genesis.Config)
	return nil
}

// getOVMETHTotalSupply returns OVM ETH's total supply by reading
// the appropriate storage slot.
func getOVMETHTotalSupply(inStateDB *state.StateDB) *big.Int {
	position := common.Big2
	key := common.BytesToHash(common.LeftPadBytes(position.Bytes(), 32))
	return inStateDB.GetState(OVMETHAddress, key).Big()
}

// getCode returns a contract's code. Taken verbatim from Geth.
func getCode(addrHash common.Hash, data types.StateAccount, db state.Database) []byte {
	if bytes.Equal(data.CodeHash, emptyCodeHash) {
		return nil
	}

	code, err := db.ContractCode(
		addrHash,
		common.BytesToHash(data.CodeHash),
	)
	if err != nil {
		panic(err)
	}
	return code
}

// getOVMETHBalance gets a user's OVM ETH balance from state by querying the
// appropriate storage slot directly.
func getOVMETHBalance(inStateDB *state.StateDB, addr common.Address) *big.Int {
	return inStateDB.GetState(OVMETHAddress, CalcOVMETHStorageKey(addr)).Big()
}
