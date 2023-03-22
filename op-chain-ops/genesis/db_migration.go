package genesis

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/ether"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var (
	abiTrue  = common.Hash{31: 0x01}
	abiFalse = common.Hash{}
	// BedrockTransitionBlockExtraData represents the extradata
	// set in the very first bedrock block. This value must be
	// less than 32 bytes long or it will create an invalid block.
	BedrockTransitionBlockExtraData = []byte("BEDROCK")
)

type MigrationResult struct {
	TransitionHeight    uint64
	TransitionTimestamp uint64
	TransitionBlockHash common.Hash
}

// MigrateDB will migrate an l2geth legacy Optimism database to a Bedrock database.
func MigrateDB(ldb ethdb.Database, config *DeployConfig, l1Block *types.Block, migrationData *crossdomain.MigrationData, commit, noCheck bool) (*MigrationResult, error) {
	// Grab the hash of the tip of the legacy chain.
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)

	// Grab the header number.
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}

	// Grab the full header.
	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "number", *num)

	// Ensure that the extradata is valid.
	if size := len(BedrockTransitionBlockExtraData); size > 32 {
		return nil, fmt.Errorf("transition block extradata too long: %d", size)
	}

	// We write special extra data into the Bedrock transition block to indicate that the migration
	// has already happened. If we detect this extra data, we can skip the migration.
	if bytes.Equal(header.Extra, BedrockTransitionBlockExtraData) {
		log.Info("Detected migration already happened", "root", header.Root, "blockhash", header.Hash())

		return &MigrationResult{
			TransitionHeight:    *num,
			TransitionTimestamp: header.Time,
			TransitionBlockHash: hash,
		}, nil
	}

	// Ensure that the timestamp for the Bedrock transition block is greater than the timestamp of
	// the last legacy block.
	if uint64(config.L2OutputOracleStartingTimestamp) <= header.Time {
		return nil, fmt.Errorf(
			"output oracle starting timestamp (%d) is less than the header timestamp (%d)", config.L2OutputOracleStartingTimestamp, header.Time,
		)
	}

	// Ensure that the timestamp for the Bedrock transition block is greater than 0, not implicitly
	// guaranteed by the above check because the above converted the timestamp to a uint64.
	if config.L2OutputOracleStartingTimestamp <= 0 {
		return nil, fmt.Errorf(
			"output oracle starting timestamp (%d) cannot be <= 0", config.L2OutputOracleStartingTimestamp,
		)
	}

	dbFactory := func() (*state.StateDB, error) {
		// Set up the backing store.
		underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
			Preimages: true,
			Cache:     1024,
		})

		// Open up the state database.
		db, err := state.New(header.Root, underlyingDB, nil)
		if err != nil {
			return nil, fmt.Errorf("cannot open StateDB: %w", err)
		}

		return db, nil
	}

	db, err := dbFactory()
	if err != nil {
		return nil, fmt.Errorf("cannot create StateDB: %w", err)
	}

	// Before we do anything else, we need to ensure that all of the input configuration is correct
	// and nothing is missing. We'll first verify the contract configuration, then we'll verify the
	// witness data for the migration. We operate under the assumption that the witness data is
	// untrusted and must be verified explicitly before we can use it.

	// Generate and verify the configuration for storage variables to be set on L2.
	storage, err := NewL2StorageConfig(config, l1Block)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage config: %w", err)
	}

	// Generate and verify the configuration for immutable variables to be set on L2.
	immutable, err := NewL2ImmutableConfig(config, l1Block)
	if err != nil {
		return nil, fmt.Errorf("cannot create immutable config: %w", err)
	}

	// Convert all input messages into legacy messages. Note that this list is not yet filtered and
	// may be missing some messages or have some extra messages.
	unfilteredWithdrawals, invalidMessages, err := migrationData.ToWithdrawals()
	if err != nil {
		return nil, fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	log.Info("Read withdrawals from witness data", "unfiltered", len(unfilteredWithdrawals), "invalid", len(invalidMessages))

	// We now need to check that we have all of the withdrawals that we expect to have. An error
	// will be thrown if there are any missing messages, and any extra messages will be removed.
	var filteredWithdrawals crossdomain.SafeFilteredWithdrawals
	if !noCheck {
		log.Info("Checking withdrawals...")
		filteredWithdrawals, err = crossdomain.PreCheckWithdrawals(db, unfilteredWithdrawals, invalidMessages)
		if err != nil {
			return nil, fmt.Errorf("withdrawals mismatch: %w", err)
		}
	} else {
		log.Info("Skipping checking withdrawals")
		filteredWithdrawals = crossdomain.SafeFilteredWithdrawals(unfilteredWithdrawals)
	}

	// At this point we've fully verified the witness data for the migration, so we can begin the
	// actual migration process. This involves modifying parts of the legacy database and inserting
	// a transition block.

	// We need to wipe the storage of every predeployed contract EXCEPT for the GovernanceToken,
	// WETH9, the DeployerWhitelist, the LegacyMessagePasser, and LegacyERC20ETH. We have verified
	// that none of the legacy storage (other than the aforementioned contracts) is accessible and
	// therefore can be safely removed from the database. Storage must be wiped before anything
	// else or the ERC-1967 proxy storage slots will be removed.
	if err := WipePredeployStorage(db); err != nil {
		return nil, fmt.Errorf("cannot wipe storage: %w", err)
	}

	// Next order of business is to convert all predeployed smart contracts into proxies so they
	// can be easily upgraded later on. In the legacy system, all upgrades to predeployed contracts
	// required hard forks which was a huge pain. Note that we do NOT put the GovernanceToken or
	// WETH9 contracts behind proxies because we do not want to make these easily upgradable.
	log.Info("Converting predeployed contracts to proxies")
	if err := SetL2Proxies(db); err != nil {
		return nil, fmt.Errorf("cannot set L2Proxies: %w", err)
	}

	// Here we update the storage of each predeploy with the new storage variables that we want to
	// set on L2 and update the implementations for all predeployed contracts that are behind
	// proxies (NOT the GovernanceToken or WETH9).
	log.Info("Updating implementations for predeployed contracts")
	if err := SetImplementations(db, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set implementations: %w", err)
	}

	// We need to update the code for LegacyERC20ETH. This is NOT a standard predeploy because it's
	// deployed at the 0xdeaddeaddead... address and therefore won't be updated by the previous
	// function call to SetImplementations.
	log.Info("Updating code for LegacyERC20ETH")
	if err := SetLegacyETH(db, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set legacy ETH: %w", err)
	}

	// Now we migrate legacy withdrawals from the LegacyMessagePasser contract to their new format
	// in the Bedrock L2ToL1MessagePasser contract. Note that we do NOT delete the withdrawals from
	// the LegacyMessagePasser contract. Here we operate on the list of withdrawals that we
	// previously filtered and verified.
	log.Info("Starting to migrate withdrawals", "no-check", noCheck)
	err = crossdomain.MigrateWithdrawals(filteredWithdrawals, db, &config.L1CrossDomainMessengerProxy, noCheck)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate withdrawals: %w", err)
	}

	// Finally we migrate the balances held inside the LegacyERC20ETH contract into the state trie.
	// We also delete the balances from the LegacyERC20ETH contract. Unlike the steps above, this step
	// combines the check and mutation steps into one in order to reduce migration time.
	log.Info("Starting to migrate ERC20 ETH")
	err = ether.MigrateBalances(db, dbFactory, migrationData.Addresses(), migrationData.OvmAllowances, int(config.L1ChainID), noCheck)
	if err != nil {
		return nil, fmt.Errorf("failed to migrate OVM_ETH: %w", err)
	}

	// We're done messing around with the database, so we can now commit the changes to the DB.
	// Note that this doesn't actually write the changes to disk.
	log.Info("Committing state DB")
	newRoot, err := db.Commit(true)
	if err != nil {
		return nil, err
	}

	// Create the header for the Bedrock transition block.
	bedrockHeader := &types.Header{
		ParentHash:  header.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    predeploys.SequencerFeeVaultAddr,
		Root:        newRoot,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      new(big.Int).Add(header.Number, common.Big1),
		GasLimit:    (uint64)(config.L2GenesisBlockGasLimit),
		GasUsed:     0,
		Time:        uint64(config.L2OutputOracleStartingTimestamp),
		Extra:       BedrockTransitionBlockExtraData,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	// Create the Bedrock transition block from the header. Note that there are no transactions,
	// uncle blocks, or receipts in the Bedrock transition block.
	bedrockBlock := types.NewBlock(bedrockHeader, nil, nil, nil, trie.NewStackTrie(nil))

	// We did it!
	log.Info(
		"Built Bedrock transition",
		"hash", bedrockBlock.Hash(),
		"root", bedrockBlock.Root(),
		"number", bedrockBlock.NumberU64(),
		"gas-used", bedrockBlock.GasUsed(),
		"gas-limit", bedrockBlock.GasLimit(),
	)

	// Create the result of the migration.
	res := &MigrationResult{
		TransitionHeight:    bedrockBlock.NumberU64(),
		TransitionTimestamp: bedrockBlock.Time(),
		TransitionBlockHash: bedrockBlock.Hash(),
	}

	// If we're not actually writing this to disk, then we're done.
	if !commit {
		log.Info("Dry run complete")
		return res, nil
	}

	// Otherwise we need to write the changes to disk. First we commit the state changes.
	log.Info("Committing trie DB")
	if err := db.Database().TrieDB().Commit(newRoot, true); err != nil {
		return nil, err
	}

	// Next we write the Bedrock transition block to the database.
	rawdb.WriteTd(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64(), bedrockBlock.Difficulty())
	rawdb.WriteBlock(ldb, bedrockBlock)
	rawdb.WriteReceipts(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, bedrockBlock.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, bedrockBlock.Hash())
	rawdb.WriteHeadHeaderHash(ldb, bedrockBlock.Hash())

	// Make the first Bedrock block a finalized block.
	rawdb.WriteFinalizedBlockHash(ldb, bedrockBlock.Hash())

	// We need to update the chain config to set the correct hardforks.
	genesisHash := rawdb.ReadCanonicalHash(ldb, 0)
	cfg := rawdb.ReadChainConfig(ldb, genesisHash)
	if cfg == nil {
		log.Crit("chain config not found")
	}

	// Set the standard options.
	cfg.LondonBlock = bedrockBlock.Number()
	cfg.ArrowGlacierBlock = bedrockBlock.Number()
	cfg.GrayGlacierBlock = bedrockBlock.Number()
	cfg.MergeNetsplitBlock = bedrockBlock.Number()
	cfg.TerminalTotalDifficulty = big.NewInt(0)
	cfg.TerminalTotalDifficultyPassed = true

	// Set the Optimism options.
	cfg.BedrockBlock = bedrockBlock.Number()
	// Enable Regolith from the start of Bedrock
	cfg.RegolithTime = new(uint64)
	cfg.Optimism = &params.OptimismConfig{
		EIP1559Denominator: config.EIP1559Denominator,
		EIP1559Elasticity:  config.EIP1559Elasticity,
	}

	// Write the chain config to disk.
	rawdb.WriteChainConfig(ldb, genesisHash, cfg)

	// Yay!
	log.Info(
		"wrote chain config",
		"1559-denominator", config.EIP1559Denominator,
		"1559-elasticity", config.EIP1559Elasticity,
	)

	// We're done!
	log.Info(
		"wrote Bedrock transition block",
		"height", bedrockHeader.Number,
		"root", bedrockHeader.Root.String(),
		"hash", bedrockHeader.Hash().String(),
		"timestamp", bedrockHeader.Time,
	)

	// Return the result and have a nice day.
	return res, nil
}
