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
	"github.com/ethereum/go-ethereum/trie"
)

var abiTrue = common.Hash{31: 0x01}

// MigrateDB will migrate an old l2geth database to the new bedrock style system
func MigrateDB(ldb ethdb.Database, config *DeployConfig, l1Block *types.Block, l2Addrs *L2Addresses, migrationData *migration.MigrationData, commit bool) error {
	hash := rawdb.ReadHeadHeaderHash(ldb)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	header := rawdb.ReadHeader(ldb, hash, *num)

	db, err := state.New(header.Root, state.NewDatabase(ldb), nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	// Convert all of the messages into legacy withdrawals
	withdrawals, err := migrationData.ToWithdrawals()
	if err != nil {
		return fmt.Errorf("cannot serialize withdrawals: %w", err)
	}

	if err := CheckWithdrawals(db, withdrawals); err != nil {
		return fmt.Errorf("withdrawals mismatch: %w", err)
	}

	// Now start the migration
	if err := SetL2Proxies(db); err != nil {
		return fmt.Errorf("cannot set L2Proxies: %w", err)
	}

	storage, err := NewL2StorageConfig(config, l1Block, l2Addrs)
	if err != nil {
		return fmt.Errorf("cannot create storage config: %w", err)
	}

	immutable, err := NewL2ImmutableConfig(config, l1Block, l2Addrs)
	if err != nil {
		return fmt.Errorf("cannot create immutable config: %w", err)
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return fmt.Errorf("cannot set implementations: %w", err)
	}

	err = crossdomain.MigrateWithdrawals(withdrawals, db, &l2Addrs.L1CrossDomainMessengerProxy, &l2Addrs.L1StandardBridgeProxy)
	if err != nil {
		return fmt.Errorf("cannot migrate withdrawals: %w", err)
	}

	addrs := migrationData.Addresses()
	if err := ether.MigrateLegacyETH(ldb, addrs, migrationData.OvmAllowances, int(config.L1ChainID)); err != nil {
		return fmt.Errorf("cannot migrate legacy eth: %w", err)
	}

	if !commit {
		return nil
	}

	root, err := db.Commit(true)
	if err != nil {
		return fmt.Errorf("cannot commit state db: %w", err)
	}

	// Create the bedrock transition block
	bedrockHeader := &types.Header{
		ParentHash:  header.Hash(),
		UncleHash:   types.EmptyUncleHash,
		Coinbase:    config.L2GenesisBlockCoinbase,
		Root:        root,
		TxHash:      types.EmptyRootHash,
		ReceiptHash: types.EmptyRootHash,
		Bloom:       types.Bloom{},
		Difficulty:  (*big.Int)(config.L2GenesisBlockDifficulty),
		Number:      new(big.Int).Add(header.Number, common.Big1),
		GasLimit:    (uint64)(config.L2GenesisBlockGasLimit),
		GasUsed:     (uint64)(config.L2GenesisBlockGasUsed),
		Time:        uint64(config.L2OutputOracleStartingTimestamp),
		Extra:       config.L2GenesisBlockExtraData,
		MixDigest:   config.L2GenesisBlockMixHash,
		Nonce:       types.EncodeNonce((uint64)(config.L1GenesisBlockNonce)),
		BaseFee:     (*big.Int)(config.L2GenesisBlockBaseFeePerGas),
	}

	block := types.NewBlock(bedrockHeader, nil, nil, nil, trie.NewStackTrie(nil))

	rawdb.WriteTd(ldb, block.Hash(), block.NumberU64(), block.Difficulty())
	rawdb.WriteBlock(ldb, block)
	rawdb.WriteReceipts(ldb, block.Hash(), block.NumberU64(), nil)
	rawdb.WriteCanonicalHash(ldb, block.Hash(), block.NumberU64())
	rawdb.WriteHeadBlockHash(ldb, block.Hash())
	rawdb.WriteHeadFastBlockHash(ldb, block.Hash())
	rawdb.WriteHeadHeaderHash(ldb, block.Hash())

	return nil
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
		_, ok := slots[slot]
		if !ok {
			return fmt.Errorf("Unknown input message: %s", slot)
		}
	}

	return nil
}
