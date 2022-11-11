package genesis

import (
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
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

var abiTrue = common.Hash{31: 0x01}

type MigrationResult struct {
	TransitionHeight    uint64
	TransitionTimestamp uint64
	TransitionBlockHash common.Hash
}

// MigrateDB will migrate an old l2geth database to the new bedrock style system
func MigrateDB(ldb ethdb.Database, config *DeployConfig, l1Block *types.Block, migrationData *migration.MigrationData, commit bool) (*MigrationResult, error) {
	hash := rawdb.ReadHeadHeaderHash(ldb)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	header := rawdb.ReadHeader(ldb, hash, *num)

	// Leaving this commented out so that it can be used to skip
	// the DB migration in development.
	//return &MigrationResult{
	//	TransitionHeight:    *num,
	//	TransitionTimestamp: header.Time,
	//	TransitionBlockHash: hash,
	//}, nil

	underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
		Preimages: true,
	})

	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot open StateDB: %w", err)
	}

	// Convert all of the messages into legacy withdrawals
	withdrawals, err := migrationData.ToWithdrawals()
	if err != nil {
		return nil, fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	if err := CheckWithdrawals(db, withdrawals); err != nil {
		return nil, fmt.Errorf("withdrawals mismatch: %w", err)
	}

	// Now start the migration
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

	log.Info("Starting to migrate withdrawals")
	err = crossdomain.MigrateWithdrawals(withdrawals, db, &config.L1CrossDomainMessengerProxy, &config.L1StandardBridgeProxy)
	if err != nil {
		return nil, fmt.Errorf("cannot migrate withdrawals: %w", err)
	}
	log.Info("Completed withdrawal migration")

	log.Info("Starting to migrate ERC20 ETH")
	addrs := migrationData.Addresses()
	newRoot, err := ether.MigrateLegacyETH(ldb, addrs, migrationData.OvmAllowances, int(config.L1ChainID), commit)
	log.Info("Completed ERC20 ETH migration")

	if err != nil {
		return nil, fmt.Errorf("cannot migrate legacy eth: %w", err)
	}

	// Create the bedrock transition block
	bedrockHeader := &types.Header{
		ParentHash:  header.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    config.L2GenesisBlockCoinbase,
		Root:        newRoot,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  common.Big0,
		Number:      new(big.Int).Add(header.Number, common.Big1),
		GasLimit:    (uint64)(config.L2GenesisBlockGasLimit),
		GasUsed:     0,
		Time:        uint64(config.L2OutputOracleStartingTimestamp),
		Extra:       []byte("BEDROCK"),
		MixDigest:   common.Hash{},
		Nonce:       types.BlockNonce{},
		BaseFee:     (*big.Int)(config.L2GenesisBlockBaseFeePerGas),
	}

	bedrockBlock := types.NewBlock(bedrockHeader, nil, nil, nil, trie.NewStackTrie(nil))

	res := &MigrationResult{
		TransitionHeight:    bedrockBlock.NumberU64(),
		TransitionTimestamp: bedrockBlock.Time(),
		TransitionBlockHash: bedrockBlock.Hash(),
	}

	if !commit {
		log.Info("Dry run complete")
		return res, nil
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
	rawdb.WriteChainConfig(ldb, genesisHash, cfg)

	log.Info(
		"wrote Bedrock transition block",
		"height", bedrockHeader.Number,
		"root", bedrockHeader.Root.String(),
		"hash", bedrockHeader.Hash().String(),
	)

	return res, nil
}

// CheckWithdrawals will ensure that the entire list of withdrawals is being
// operated on during the database migration.
func CheckWithdrawals(db vm.StateDB, withdrawals []*crossdomain.LegacyWithdrawal) error {
	// Create a mapping of all of their storage slots
	knownSlots := make(map[common.Hash]bool)
	for _, wd := range withdrawals {
		slot, err := wd.StorageSlot()
		if err != nil {
			return err
		}
		knownSlots[slot] = true
	}
	// Build a map of all the slots in the LegacyMessagePasser
	slots := make(map[common.Hash]bool)
	err := db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		if value != abiTrue {
			return false
		}
		slots[key] = true
		return true
	})
	if err != nil {
		return err
	}

	// Check that all of the slots from storage correspond to a known message
	for slot := range slots {
		_, ok := knownSlots[slot]
		if !ok {
			return fmt.Errorf("Unknown storage slot in state: %s", slot)
		}
	}
	// Check that all of the input messages are legit
	for slot := range knownSlots {
		//nolint:staticcheck
		_, ok := slots[slot]
		//nolint:staticcheck
		if !ok {
			//return nil, fmt.Errorf("Unknown input message: %s", slot)
		}
	}

	return nil
}
