package genesis

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/ether"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
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
	abiTrue = common.Hash{31: 0x01}
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

// MigrateDB will migrate an old l2geth database to the new bedrock style system
func MigrateDB(ldb ethdb.Database, config *DeployConfig, l1Block *types.Block, migrationData *migration.MigrationData, commit, noCheck bool) (*MigrationResult, error) {
	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return nil, fmt.Errorf("cannot find header number for %s", hash)
	}

	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "number", *num)

	if bytes.Equal(header.Extra, BedrockTransitionBlockExtraData) {
		log.Info("Detected migration already happened", "root", header.Root, "blockhash", header.Hash())

		return &MigrationResult{
			TransitionHeight:    *num,
			TransitionTimestamp: header.Time,
			TransitionBlockHash: hash,
		}, nil
	}

	// Ensure monotonic timestamps
	if uint64(config.L2OutputOracleStartingTimestamp) <= header.Time {
		return nil, fmt.Errorf(
			"L2 output oracle starting timestamp (%d) is less than the header timestamp (%d)", config.L2OutputOracleStartingTimestamp, header.Time,
		)
	}

	// Ensure that the starting timestamp is safe
	if config.L2OutputOracleStartingTimestamp <= 0 {
		return nil, fmt.Errorf(
			"L2 output oracle starting timestamp (%d) cannot be <= 0", config.L2OutputOracleStartingTimestamp,
		)
	}

	underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
		Preimages: true,
		Cache:     1024,
	})

	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open StateDB: %w", err)
	}

	// Convert all of the messages into legacy withdrawals
	unfilteredWithdrawals, err := migrationData.ToWithdrawals()
	if err != nil {
		return nil, fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	var filteredWithdrawals []*crossdomain.LegacyWithdrawal
	if !noCheck {
		log.Info("Checking withdrawals...")
		filteredWithdrawals, err = PreCheckWithdrawals(db, unfilteredWithdrawals)
		if err != nil {
			return nil, fmt.Errorf("withdrawals mismatch: %w", err)
		}
		log.Info("Withdrawals accounted for!")
	} else {
		log.Info("Skipping checking withdrawals")
		filteredWithdrawals = unfilteredWithdrawals
	}

	// Now start the migration
	log.Info("Setting the Proxies")
	if err := SetL2Proxies(db); err != nil {
		return nil, fmt.Errorf("cannot set L2Proxies: %w", err)
	}

	storage, err := NewL2StorageConfig(config, l1Block)
	if err != nil {
		return nil, fmt.Errorf("cannot create storage config: %w", err)
	}

	immutable, err := NewL2ImmutableConfig(config, l1Block)
	if err != nil {
		return nil, fmt.Errorf("cannot create immutable config: %w", err)
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set implementations: %w", err)
	}

	if err := SetLegacyETH(db, storage, immutable); err != nil {
		return nil, fmt.Errorf("cannot set legacy ETH: %w", err)
	}

	log.Info("Starting to migrate withdrawals", "no-check", noCheck)
	err = crossdomain.MigrateWithdrawals(filteredWithdrawals, db, &config.L1CrossDomainMessengerProxy, noCheck)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate withdrawals: %w", err)
	}
	log.Info("Completed withdrawal migration")

	log.Info("Starting to migrate ERC20 ETH")
	addrs := migrationData.Addresses()
	err = ether.MigrateLegacyETH(ldb, db, addrs, migrationData.OvmAllowances, int(config.L1ChainID), noCheck)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate legacy eth: %w", err)
	}
	log.Info("Completed ERC20 ETH migration")

	newRoot, err := db.Commit(true)
	if err != nil {
		return nil, err
	}
	log.Info("committed state DB", "root", newRoot)

	// Set the amount of gas used so that EIP 1559 starts off stable
	gasUsed := (uint64)(config.L2GenesisBlockGasLimit) * config.EIP1559Elasticity

	// Ensure that the extradata is valid
	if size := len(BedrockTransitionBlockExtraData); size > 32 {
		return nil, fmt.Errorf("transition block extradata too long: %d", size)
	}

	// Create the bedrock transition block
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
		GasUsed:     gasUsed,
		Time:        uint64(config.L2OutputOracleStartingTimestamp),
		Extra:       BedrockTransitionBlockExtraData,
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     big.NewInt(params.InitialBaseFee),
	}

	bedrockBlock := types.NewBlock(bedrockHeader, nil, nil, nil, trie.NewStackTrie(nil))

	log.Info(
		"Built Bedrock transition",
		"hash", bedrockBlock.Hash(),
		"root", bedrockBlock.Root(),
		"number", bedrockBlock.NumberU64(),
		"gas-used", bedrockBlock.GasUsed(),
		"gas-limit", bedrockBlock.GasLimit(),
	)

	res := &MigrationResult{
		TransitionHeight:    bedrockBlock.NumberU64(),
		TransitionTimestamp: bedrockBlock.Time(),
		TransitionBlockHash: bedrockBlock.Hash(),
	}

	if !commit {
		log.Info("Dry run complete")
		return res, nil
	}

	log.Info("committing trie DB")
	if err := db.Database().TrieDB().Commit(newRoot, true, nil); err != nil {
		return nil, err
	}

	rawdb.WriteTd(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64(), bedrockBlock.Difficulty())
	rawdb.WriteBlock(ldb, bedrockBlock)
	rawdb.WriteReceipts(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, bedrockBlock.Hash(), bedrockBlock.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, bedrockBlock.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, bedrockBlock.Hash())
	rawdb.WriteHeadHeaderHash(ldb, bedrockBlock.Hash())

	// Make the first Bedrock block a finalized block.
	rawdb.WriteFinalizedBlockHash(ldb, bedrockBlock.Hash())

	// We need to pull the chain config out of the DB, and update
	// it so that the latest hardforks are enabled.
	genesisHash := rawdb.ReadCanonicalHash(ldb, 0)
	cfg := rawdb.ReadChainConfig(ldb, genesisHash)
	if cfg == nil {
		log.Crit("chain config not found")
	}
	cfg.LondonBlock = bedrockBlock.Number()
	cfg.ArrowGlacierBlock = bedrockBlock.Number()
	cfg.GrayGlacierBlock = bedrockBlock.Number()
	cfg.MergeNetsplitBlock = bedrockBlock.Number()
	cfg.TerminalTotalDifficulty = big.NewInt(0)
	cfg.TerminalTotalDifficultyPassed = true
	cfg.Optimism = &params.OptimismConfig{
		EIP1559Denominator: config.EIP1559Denominator,
		EIP1559Elasticity:  config.EIP1559Elasticity,
	}
	cfg.BedrockBlock = bedrockBlock.Number()
	rawdb.WriteChainConfig(ldb, genesisHash, cfg)

	log.Info(
		"wrote chain config",
		"1559-denominator", config.EIP1559Denominator,
		"1559-elasticity", config.EIP1559Elasticity,
	)

	log.Info(
		"wrote Bedrock transition block",
		"height", bedrockHeader.Number,
		"root", bedrockHeader.Root.String(),
		"hash", bedrockHeader.Hash().String(),
		"timestamp", bedrockHeader.Time,
	)

	return res, nil
}

// PreCheckWithdrawals will ensure that the entire list of withdrawals is being
// operated on during the database migration.
func PreCheckWithdrawals(db *state.StateDB, withdrawals []*crossdomain.LegacyWithdrawal) ([]*crossdomain.LegacyWithdrawal, error) {
	// Create a mapping of all of their storage slots
	slotsWds := make(map[common.Hash]*crossdomain.LegacyWithdrawal)
	for _, wd := range withdrawals {
		slot, err := wd.StorageSlot()
		if err != nil {
			return nil, fmt.Errorf("cannot check withdrawals: %w", err)
		}

		slotsWds[slot] = wd
	}

	// Build a map of all the slots in the LegacyMessagePasser
	var count int
	slots := make(map[common.Hash]bool)
	err := db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		if value != abiTrue {
			return false
		}
		slots[key] = true
		count++
		return true
	})

	if err != nil {
		return nil, fmt.Errorf("cannot iterate over LegacyMessagePasser: %w", err)
	}

	log.Info("iterated legacy messages", "count", count)

	// Check that all of the slots from storage correspond to a known message
	for slot := range slots {
		_, ok := slotsWds[slot]
		if !ok {
			return nil, fmt.Errorf("Unknown storage slot in state: %s", slot)
		}
	}

	filtered := make([]*crossdomain.LegacyWithdrawal, 0)

	// Check that all of the input messages are legit
	for slot := range slotsWds {
		//nolint:staticcheck
		_, ok := slots[slot]
		//nolint:staticcheck
		if !ok {
			log.Info("filtering out unknown input message", "slot", slot.String())
			continue
		}

		filtered = append(filtered, slotsWds[slot])
	}

	return filtered, nil
}
