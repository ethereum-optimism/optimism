package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-service/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"

	"github.com/holiman/uint256"
)

var (
	OutFilePerm = os.FileMode(0o440)

	alfajoresChainId          uint64 = 44787
	accountOverwriteWhitelist        = map[uint64]map[common.Address]struct{}{
		// Add any addresses that should be allowed to overwrite existing accounts here.
		alfajoresChainId: {
			// Create2Deployer
			common.HexToAddress("0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2"): {},
		},
	}
)

func applyStateMigrationChanges(config *genesis.DeployConfig, genesis *core.Genesis, dbPath string) (*types.Header, error) {
	log.Info("Opening Celo database", "dbPath", dbPath)

	ldb, err := openDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open DB: %w", err)
	}
	log.Info("Loaded Celo L1 DB", "db", ldb)

	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)

	// Grab the header number.
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}
	log.Info("Reading chain tip num from database", "number", num)

	// Grab the full header.
	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "header", header)

	// We need to update the chain config to set the correct hardforks.
	genesisHash := rawdb.ReadCanonicalHash(ldb, 0)
	cfg := rawdb.ReadChainConfig(ldb, genesisHash)
	if cfg == nil {
		log.Crit("chain config not found")
	}
	log.Info("Read chain config from database", "config", cfg)

	// Set up the backing store.
	// TODO(pl): Do we need the preimages setting here?
	underlyingDB := state.NewDatabaseWithConfig(ldb, &triedb.Config{Preimages: true})

	// Open up the state database.
	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open StateDB: %w", err)
	}

	// Apply the changes to the state DB.
	err = applyAllocsToState(db, genesis, cfg)
	if err != nil {
		return nil, err
	}

	migrationBlock := new(big.Int).Add(header.Number, common.Big1)

	// We're done messing around with the database, so we can now commit the changes to the DB.
	// Note that this doesn't actually write the changes to disk.
	log.Info("Committing state DB")
	newRoot, err := db.Commit(migrationBlock.Uint64(), true)
	if err != nil {
		return nil, err
	}

	baseFee := new(big.Int).SetUint64(params.InitialBaseFee)
	if header.BaseFee != nil {
		baseFee = header.BaseFee
	}
	// Create the header for the Cel2 transition block.
	cel2Header := &types.Header{
		ParentHash:      header.Hash(),
		UncleHash:       types.EmptyUncleHash,
		Coinbase:        predeploys.SequencerFeeVaultAddr,
		Root:            newRoot,
		TxHash:          types.EmptyTxsHash,
		ReceiptHash:     types.EmptyReceiptsHash,
		Bloom:           types.Bloom{},
		Difficulty:      new(big.Int).Set(common.Big0),
		Number:          migrationBlock,
		GasLimit:        header.GasLimit,
		GasUsed:         0,
		Time:            header.Time + 5,
		Extra:           []byte("CeL2 migration"),
		MixDigest:       common.Hash{},
		Nonce:           types.BlockNonce{},
		BaseFee:         baseFee,
		WithdrawalsHash: &types.EmptyWithdrawalsHash,
		BlobGasUsed:     new(uint64),
		ExcessBlobGas:   new(uint64),
	}
	log.Info("Build Cel2 migration header", "header", cel2Header)

	// Create the Cel2 transition block from the header. Note that there are no transactions,
	// uncle blocks, or receipts in the Cel2 transition block.
	cel2Block := types.NewBlock(cel2Header, nil, nil, nil, trie.NewStackTrie(nil))

	// We did it!
	log.Info(
		"Built Cel2 migration block",
		"hash", cel2Block.Hash(),
		"root", cel2Block.Root(),
		"number", cel2Block.NumberU64(),
	)

	log.Info("Committing trie DB")
	if err := db.Database().TrieDB().Commit(newRoot, true); err != nil {
		return nil, err
	}

	// Next we write the Cel2 genesis block to the database.
	rawdb.WriteTd(ldb, cel2Block.Hash(), cel2Block.NumberU64(), cel2Block.Difficulty())
	rawdb.WriteBlock(ldb, cel2Block)
	rawdb.WriteReceipts(ldb, cel2Block.Hash(), cel2Block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, cel2Block.Hash(), cel2Block.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, cel2Block.Hash())
	rawdb.WriteHeadHeaderHash(ldb, cel2Block.Hash())

	// Mark the first CeL2 block as finalized
	rawdb.WriteFinalizedBlockHash(ldb, cel2Block.Hash())

	// Set the standard options.
	cfg.LondonBlock = cel2Block.Number()
	cfg.BerlinBlock = cel2Block.Number()
	cfg.ArrowGlacierBlock = cel2Block.Number()
	cfg.GrayGlacierBlock = cel2Block.Number()
	cfg.MergeNetsplitBlock = cel2Block.Number()
	cfg.TerminalTotalDifficulty = big.NewInt(0)
	cfg.TerminalTotalDifficultyPassed = true
	cfg.ShanghaiTime = &cel2Header.Time
	cfg.CancunTime = &cel2Header.Time

	// Set the Optimism options.
	cfg.BedrockBlock = cel2Block.Number()
	// Enable Regolith from the start of Bedrock
	cfg.RegolithTime = new(uint64) // what are those? do we need those?
	cfg.Optimism = &params.OptimismConfig{
		EIP1559Denominator:       config.EIP1559Denominator,
		EIP1559DenominatorCanyon: config.EIP1559DenominatorCanyon,
		EIP1559Elasticity:        config.EIP1559Elasticity,
	}
	cfg.CanyonTime = &cel2Header.Time
	cfg.EcotoneTime = &cel2Header.Time
	cfg.FjordTime = &cel2Header.Time
	cfg.Cel2Time = &cel2Header.Time

	// Write the chain config to disk.
	// TODO(pl): Why do we need to write this with the genesis hash, not `cel2Block.Hash()`?`
	rawdb.WriteChainConfig(ldb, genesisHash, cfg)
	marhslledConfig, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal chain config to JSON: %w", err)
	}
	log.Info("Wrote updated chain config", "config", string(marhslledConfig))

	// We're done!
	log.Info(
		"Wrote CeL2 migration block",
		"height", cel2Header.Number,
		"root", cel2Header.Root.String(),
		"hash", cel2Header.Hash().String(),
		"timestamp", cel2Header.Time,
	)

	// Close the database handle
	if err := ldb.Close(); err != nil {
		return nil, err
	}

	return cel2Header, nil
}

// applyAllocsToState applies the account allocations from the allocation file to the state database.
// It creates new accounts, sets their nonce, balance, code, and storage values.
// If an account already exists, it adds the balance of the new account to the existing balance.
// If the code of an existing account is different from the code in the genesis block, it logs a warning.
// This changes the state root, so `Commit` needs to be called after this function.
func applyAllocsToState(db *state.StateDB, genesis *core.Genesis, config *params.ChainConfig) error {
	log.Info("Starting to migrate OP contracts into state DB")

	accountCounter := 0
	overwriteCounter := 0
	for k, v := range genesis.Alloc {
		accountCounter++

		balance := uint256.MustFromBig(v.Balance)

		if db.Exist(k) {
			// If the account already has balance, add it to the balance of the new account
			balance = balance.Add(balance, db.GetBalance(k))

			currentCode := db.GetCode(k)
			equalCode := bytes.Equal(currentCode, v.Code)
			if currentCode != nil && !equalCode {
				if whitelist, exists := accountOverwriteWhitelist[config.ChainID.Uint64()]; exists {
					if _, ok := whitelist[k]; ok {
						log.Info("Account already exists with different code and is whitelisted, overwriting...", "address", k)
					} else {
						log.Warn("Account already exists with different code and is not whitelisted, overwriting...", "address", k, "oldCode", db.GetCode(k), "newCode", v.Code)
					}
				} else {
					log.Warn("Account already exists with different code and no whitelist exists", "address", k, "oldCode", db.GetCode(k), "newCode", v.Code)
				}

				overwriteCounter++
			}
		}
		db.CreateAccount(k)

		db.SetNonce(k, v.Nonce)
		db.SetBalance(k, balance)
		db.SetCode(k, v.Code)
		for key, value := range v.Storage {
			db.SetState(k, key, value)
		}

		log.Info("Moved account", "address", k)
	}
	log.Info("Migrated OP contracts into state DB", "copiedAccounts", accountCounter, "overwrittenAccounts", overwriteCounter)
	return nil
}
