package ether

import (
	"encoding/json"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	// OVMETHAddress is the address of the OVM ETH predeploy.
	OVMETHAddress = common.HexToAddress("0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000")

	// maxSlot is the maximum slot we'll consider to be a non-mapping variable.
	maxSlot = new(big.Int).SetUint64(256)
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

	addresses := make([]common.Address, 0)
	for _, list := range addrLists {
		log.Info("reading address list", "list", list)
		f, err := os.Open(list)
		if err != nil {
			return wrapErr(err, "error opening address list %s", list)
		}
		logProgress := ProgressLogger(10000, "read address")
		err = IterateAddrList(f, func(address common.Address) error {
			addresses = append(addresses, address)
			logProgress()
			return nil
		})
		f.Close()
		if err != nil {
			return wrapErr(err, "error reading address list")
		}
	}

	allowances := make([]*migration.Allowance, 0)
	for _, list := range allowanceLists {
		log.Info("reading allowance list", "list", list)
		f, err := os.Open(list)
		if err != nil {
			return wrapErr(err, "error opening allowances list %s", list)
		}
		logProgress := ProgressLogger(10000, "read allowance")
		err = IterateAllowanceList(f, func(owner, spender common.Address) error {
			allowance := &migration.Allowance{
				From: spender,
				To:   owner,
			}
			allowances = append(allowances, allowance)
			logProgress()
			return nil
		})
		f.Close()
		if err != nil {
			return wrapErr(err, "error reading allowances list")
		}
	}

	err := MigrateLegacyETH(db, addresses, allowances, chainID)
	if err != nil {
		return wrapErr(err, "cannot migrate erc20 eth")
	}

	headBlock := rawdb.ReadHeadBlock(db)
	root := headBlock.Root()
	backingStateDB := state.NewDatabase(db)
	stateDB, err := state.New(root, backingStateDB, nil)
	if err != nil {
		return wrapErr(err, "error creating state DB")
	}

	log.Info("committing state DB")
	newRoot, err := stateDB.Commit(false)
	if err != nil {
		return wrapErr(err, "error writing output state DB")
	}
	log.Info("committed state DB", "root", newRoot)
	log.Info("committing trie DB")
	if err := stateDB.Database().TrieDB().Commit(newRoot, true, nil); err != nil {
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
	rawdb.WriteGenesisStateSpec(db, block.Hash(), blob)
	rawdb.WriteTd(db, block.Hash(), block.NumberU64(), block.Difficulty())
	rawdb.WriteBlock(db, block)
	rawdb.WriteReceipts(db, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(db, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(db, block.Hash())
	rawdb.WriteHeadFastBlockHash(db, block.Hash())
	rawdb.WriteHeadHeaderHash(db, block.Hash())
	rawdb.WriteChainConfig(db, block.Hash(), genesis.Config)
	return nil
}

// getOVMETHTotalSupply returns OVM ETH's total supply by reading
// the appropriate storage slot.
func getOVMETHTotalSupply(db *state.StateDB) *big.Int {
	key := getOVMETHTotalSupplySlot()
	return db.GetState(OVMETHAddress, key).Big()
}

func getOVMETHTotalSupplySlot() common.Hash {
	position := common.Big2
	key := common.BytesToHash(common.LeftPadBytes(position.Bytes(), 32))
	return key
}

// getOVMETHBalance gets a user's OVM ETH balance from state by querying the
// appropriate storage slot directly.
func getOVMETHBalance(db *state.StateDB, addr common.Address) *big.Int {
	return db.GetState(OVMETHAddress, CalcOVMETHStorageKey(addr)).Big()
}
