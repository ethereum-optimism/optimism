package genesis

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// MaxSlotChecks is the maximum number of storage slots to check
// when validating the untouched predeploys. This limit is in place
// to bound execution time of the migration. We can parallelize this
// in the future.
const MaxSlotChecks = 1000

var (
	LegacyETHCheckSlots = map[common.Hash]common.Hash{
		// Bridge
		common.Hash{31: 0x06}: common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000010"),
		// Symbol
		common.Hash{31: 0x04}: common.HexToHash("0x4554480000000000000000000000000000000000000000000000000000000006"),
		// Name
		common.Hash{31: 0x03}: common.HexToHash("0x457468657200000000000000000000000000000000000000000000000000000a"),
		// Total supply
		common.Hash{31: 0x02}: {},
	}

	// ContractStorageCount is a map of predeploy addresses to the number of storage slots expected
	// to be set in those predeploys after the migration. It does not include any predeploys that
	// were not wiped. It also accounts for the 2 EIP-1967 storage slots in each contract.
	ContractStorageCount = map[common.Address]int{
		predeploys.L2CrossDomainMessengerAddr:        3,
		predeploys.L2StandardBridgeAddr:              2,
		predeploys.SequencerFeeVaultAddr:             2,
		predeploys.OptimismMintableERC20FactoryAddr:  2,
		predeploys.L1BlockNumberAddr:                 2,
		predeploys.GasPriceOracleAddr:                2,
		predeploys.L1BlockAddr:                       2,
		predeploys.L2ERC721BridgeAddr:                2,
		predeploys.OptimismMintableERC721FactoryAddr: 2,
		predeploys.ProxyAdminAddr:                    1,
		predeploys.BaseFeeVaultAddr:                  2,
		predeploys.L1FeeVaultAddr:                    2,
	}
)

// PostCheckMigratedDB will check that the migration was performed correctly
func PostCheckMigratedDB(ldb ethdb.Database, migrationData migration.MigrationData, l1XDM *common.Address, l1ChainID uint64) error {
	log.Info("Validating database migration")

	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return fmt.Errorf("cannot find header number for %s", hash)
	}

	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "number", *num)

	if !bytes.Equal(header.Extra, BedrockTransitionBlockExtraData) {
		return fmt.Errorf("expected extra data to be %x, but got %x", BedrockTransitionBlockExtraData, header.Extra)
	}

	prevHeader := rawdb.ReadHeader(ldb, header.ParentHash, *num-1)
	log.Info("Read previous header from database", "number", *num-1)

	underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
		Preimages: true,
	})

	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	if err := PostCheckPredeployStorage(db); err != nil {
		return err
	}
	log.Info("checked predeploy storage")

	if err := PostCheckUntouchables(underlyingDB, db, prevHeader.Root, l1ChainID); err != nil {
		return err
	}
	log.Info("checked untouchables")

	if err := PostCheckPredeploys(db); err != nil {
		return err
	}
	log.Info("checked predeploys")

	if err := PostCheckLegacyETH(db); err != nil {
		return err
	}
	log.Info("checked legacy eth")

	if err := CheckWithdrawalsAfter(db, migrationData, l1XDM); err != nil {
		return err
	}
	log.Info("checked withdrawals")

	return nil
}

// PostCheckUntouchables will check that the untouchable contracts have
// not been modified by the migration process.
func PostCheckUntouchables(udb state.Database, currDB *state.StateDB, prevRoot common.Hash, l1ChainID uint64) error {
	prevDB, err := state.New(prevRoot, udb, nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	for addr := range UntouchablePredeploys {
		// Check that the code is the same.
		code := currDB.GetCode(addr)
		hash := crypto.Keccak256Hash(code)
		expHash := UntouchableCodeHashes[addr][l1ChainID]
		if hash != expHash {
			return fmt.Errorf("expected code hash for %s to be %s, but got %s", addr, expHash, hash)
		}
		log.Info("checked code hash", "address", addr, "hash", hash)

		// Ensure that the current/previous roots match
		prevRoot := prevDB.StorageTrie(addr).Hash()
		currRoot := currDB.StorageTrie(addr).Hash()
		if prevRoot != currRoot {
			return fmt.Errorf("expected storage root for %s to be %s, but got %s", addr, prevRoot, currRoot)
		}
		log.Info("checked account roots", "address", addr, "curr_root", currRoot, "prev_root", prevRoot)

		// Sample storage slots to ensure that they are not modified.
		var count int
		expSlots := make(map[common.Hash]common.Hash)
		err := prevDB.ForEachStorage(addr, func(key, value common.Hash) bool {
			count++
			expSlots[key] = value
			return count < MaxSlotChecks
		})
		if err != nil {
			return fmt.Errorf("error iterating over storage: %w", err)
		}

		for expKey, expValue := range expSlots {
			actValue := currDB.GetState(addr, expKey)
			if actValue != expValue {
				return fmt.Errorf("expected slot %s on %s to be %s, but got %s", expKey, addr, expValue, actValue)
			}
		}

		log.Info("checked storage", "address", addr, "count", count)
	}
	return nil
}

// PostCheckPredeploys will check that there is code at each predeploy
// address
func PostCheckPredeploys(db *state.StateDB) error {
	for i := uint64(0); i <= 2048; i++ {
		// Compute the predeploy address
		bigAddr := new(big.Int).Or(bigL2PredeployNamespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)
		// Get the code for the predeploy
		code := db.GetCode(addr)
		// There must be code for the predeploy
		if len(code) == 0 {
			return fmt.Errorf("no code found at predeploy %s", addr)
		}

		if UntouchablePredeploys[addr] {
			log.Trace("skipping untouchable predeploy", "address", addr)
			continue
		}

		// There must be an admin
		admin := db.GetState(addr, AdminSlot)
		adminAddr := common.BytesToAddress(admin.Bytes())
		if addr != predeploys.ProxyAdminAddr && addr != predeploys.GovernanceTokenAddr && adminAddr != predeploys.ProxyAdminAddr {
			return fmt.Errorf("expected admin for %s to be %s but got %s", addr, predeploys.ProxyAdminAddr, adminAddr)
		}
	}

	// For each predeploy, check that we've set the implementation correctly when
	// necessary and that there's code at the implementation.
	for _, proxyAddr := range predeploys.Predeploys {
		if UntouchablePredeploys[*proxyAddr] {
			log.Trace("skipping untouchable predeploy", "address", proxyAddr)
			continue
		}

		if *proxyAddr == predeploys.LegacyERC20ETHAddr {
			log.Trace("skipping legacy eth predeploy")
			continue
		}

		if *proxyAddr == predeploys.ProxyAdminAddr {
			implCode := db.GetCode(*proxyAddr)
			if len(implCode) == 0 {
				return errors.New("no code found at proxy admin")
			}
			continue
		}

		expImplAddr, err := AddressToCodeNamespace(*proxyAddr)
		if err != nil {
			return fmt.Errorf("error converting to code namespace: %w", err)
		}

		implCode := db.GetCode(expImplAddr)
		if len(implCode) == 0 {
			return fmt.Errorf("no code found at predeploy impl %s", *proxyAddr)
		}

		impl := db.GetState(*proxyAddr, ImplementationSlot)
		actImplAddr := common.BytesToAddress(impl.Bytes())
		if expImplAddr != actImplAddr {
			return fmt.Errorf("expected implementation for %s to be at %s, but got %s", *proxyAddr, expImplAddr, actImplAddr)
		}
	}

	return nil
}

// PostCheckPredeployStorage will ensure that the predeploys had their storage
// wiped correctly.
func PostCheckPredeployStorage(db vm.StateDB) error {
	for name, addr := range predeploys.Predeploys {
		if addr == nil {
			return fmt.Errorf("nil address in predeploys mapping for %s", name)
		}

		// Skip the addresses that did not have their storage reset, also skip the
		// L2ToL1MessagePasser because it's already covered by the withdrawals check.
		if FrozenStoragePredeploys[*addr] || *addr == predeploys.L2ToL1MessagePasserAddr {
			continue
		}

		// Create a mapping of all storage slots. These values were wiped
		// so it should not take long to iterate through all of them.
		slots := make(map[common.Hash]common.Hash)
		err := db.ForEachStorage(*addr, func(key, value common.Hash) bool {
			slots[key] = value
			return true
		})
		if err != nil {
			return err
		}

		log.Info("predeploy storage", "name", name, "address", *addr, "count", len(slots))
		for key, value := range slots {
			log.Debug("storage values", "key", key, "value", value)
		}

		// Assert that the correct number of slots are present.
		if ContractStorageCount[*addr] != len(slots) {
			return fmt.Errorf("expected %d storage slots for %s but got %d", ContractStorageCount[*addr], name, len(slots))
		}
	}
	return nil
}

// PostCheckLegacyETH checks that the legacy eth migration was successful.
// It currently only checks that the total supply was set to 0.
func PostCheckLegacyETH(db vm.StateDB) error {
	for slot, expValue := range LegacyETHCheckSlots {
		actValue := db.GetState(predeploys.LegacyERC20ETHAddr, slot)
		if actValue != expValue {
			return fmt.Errorf("expected slot %s on %s to be %s, but got %s", slot, predeploys.LegacyERC20ETHAddr, expValue, actValue)
		}
	}
	return nil
}

func CheckWithdrawalsAfter(db vm.StateDB, data migration.MigrationData, l1CrossDomainMessenger *common.Address) error {
	wds, err := data.ToWithdrawals()
	if err != nil {
		return err
	}

	// First, make a mapping between old withdrawal slots and new ones.
	// This list can be a superset of what was actually migrated, since
	// some witness data may references withdrawals that reverted.
	oldToNew := make(map[common.Hash]common.Hash)
	for _, wd := range wds {
		migrated, err := crossdomain.MigrateWithdrawal(wd, l1CrossDomainMessenger)
		if err != nil {
			return err
		}

		legacySlot, err := wd.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute legacy storage slot: %w", err)
		}
		migratedSlot, err := migrated.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute migrated storage slot: %w", err)
		}

		oldToNew[legacySlot] = migratedSlot
	}

	// Now, iterate over each legacy withdrawal and check if there is a corresponding
	// migrated withdrawal.
	var innerErr error
	err = db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		// The legacy message passer becomes a proxy during the migration,
		// so we need to ignore the implementation/admin slots.
		if key == ImplementationSlot || key == AdminSlot {
			return true
		}

		// All other values should be abiTrue, since the only other state
		// in the message passer is the mapping of messages to boolean true.
		if value != abiTrue {
			innerErr = fmt.Errorf("non-true value found in legacy message passer. key: %s, value: %s", key, value)
			return false
		}

		// Grab the migrated slot.
		migratedSlot := oldToNew[key]
		if migratedSlot == (common.Hash{}) {
			innerErr = fmt.Errorf("no migrated slot found for legacy slot %s", key)
			return false
		}

		// Look up the migrated slot in the DB, and make sure it is abiTrue.
		migratedValue := db.GetState(predeploys.L2ToL1MessagePasserAddr, migratedSlot)
		if migratedValue != abiTrue {
			innerErr = fmt.Errorf("expected migrated value to be true, but got %s", migratedValue)
			return false
		}

		return true
	})
	if err != nil {
		return fmt.Errorf("error iterating storage slots: %w", err)
	}
	if innerErr != nil {
		return fmt.Errorf("error checking storage slots: %w", innerErr)
	}
	return nil
}
