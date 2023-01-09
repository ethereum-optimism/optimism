package genesis

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

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

type StorageCheckMap = map[common.Hash]common.Hash

var (
	L2XDMOwnerSlot      = common.Hash{31: 0x33}
	ProxyAdminOwnerSlot = common.Hash{}

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

	// ExpectedStorageSlots is a map of predeploy addresses to the storage slots and values that are
	// expected to be set in those predeploys after the migration. It does not include any predeploys
	// that were not wiped. It also accounts for the 2 EIP-1967 storage slots in each contract.
	// It does _not_ include L1Block. L1Block is checked separately.
	ExpectedStorageSlots = map[common.Address]StorageCheckMap{
		predeploys.L2CrossDomainMessengerAddr: {
			// Slot 0x00 (0) is a combination of spacer_0_0_20, _initialized, and _initializing
			common.Hash{}: common.HexToHash("0x0000000000000000000000010000000000000000000000000000000000000000"),
			// Slot 0x33 (51) is _owner. Requires custom check, so set to a garbage value
			L2XDMOwnerSlot: common.HexToHash("0xbadbadbadbad0xbadbadbadbadbadbadbadbad0xbadbadbadbad0xbadbadbad0"),
			// Slot 0x97 (151) is _status
			common.Hash{31: 0x97}: common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001"),
			// Slot 0xcc (204) is xDomainMsgSender
			common.Hash{31: 0xcc}: common.HexToHash("0x000000000000000000000000000000000000000000000000000000000000dead"),
			// EIP-1967 storage slots
			AdminSlot:          common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: common.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30007"),
		},
		predeploys.L2StandardBridgeAddr:             eip1967Slots(predeploys.L2StandardBridgeAddr),
		predeploys.SequencerFeeVaultAddr:            eip1967Slots(predeploys.SequencerFeeVaultAddr),
		predeploys.OptimismMintableERC20FactoryAddr: eip1967Slots(predeploys.OptimismMintableERC20FactoryAddr),
		predeploys.L1BlockNumberAddr:                eip1967Slots(predeploys.L1BlockNumberAddr),
		predeploys.GasPriceOracleAddr:               eip1967Slots(predeploys.GasPriceOracleAddr),
		//predeploys.L1BlockAddr:                       eip1967Slots(predeploys.L1BlockAddr),
		predeploys.L2ERC721BridgeAddr:                eip1967Slots(predeploys.L2ERC721BridgeAddr),
		predeploys.OptimismMintableERC721FactoryAddr: eip1967Slots(predeploys.OptimismMintableERC721FactoryAddr),
		predeploys.BaseFeeVaultAddr:                  eip1967Slots(predeploys.BaseFeeVaultAddr),
		predeploys.L1FeeVaultAddr:                    eip1967Slots(predeploys.L1FeeVaultAddr),
	}
)

// PostCheckMigratedDB will check that the migration was performed correctly
func PostCheckMigratedDB(
	ldb ethdb.Database,
	migrationData migration.MigrationData,
	l1XDM *common.Address,
	l1ChainID uint64,
	finalSystemOwner common.Address,
	info *derive.L1BlockInfo,
) error {
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

	if err := PostCheckPredeployStorage(db, finalSystemOwner); err != nil {
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

	if err := PostCheckL1Block(db, info); err != nil {
		return err
	}
	log.Info("checked L1Block")

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
		impl := currDB.GetState(addr, ImplementationSlot)
		implAddr := common.BytesToAddress(impl.Bytes())
		code := currDB.GetCode(implAddr)
		hash := crypto.Keccak256Hash(code)
		expHash := UntouchableCodeHashes[addr][l1ChainID]
		if hash != expHash {
			return fmt.Errorf("expected code hash for %s to be %s, but got %s", addr, expHash, hash)
		}
		log.Info("checked code hash", "address", addr, "hash", hash)

		// Iterate over all old storage slots.
		var expCount int
		expSlots := make(map[common.Hash]common.Hash)
		err := prevDB.ForEachStorage(addr, func(key, value common.Hash) bool {
			expCount++
			expSlots[key] = value
			return true
		})
		if err != nil {
			return fmt.Errorf("error iterating over old storage: %w", err)
		}

		// Iterate over all new storage slots.
		var actCount int
		actSlots := make(map[common.Hash]common.Hash)
		err = prevDB.ForEachStorage(addr, func(key, value common.Hash) bool {
			actCount++
			actSlots[key] = value
			return true
		})
		if err != nil {
			return fmt.Errorf("error iterating over old storage: %w", err)
		}

		// Assert that every old key still exists.
		for expKey, expValue := range actSlots {
			actValue := actSlots[expKey]
			if actValue != expValue {
				return fmt.Errorf("expected slot %s on %s to be %s, but got %s", expKey, addr, expValue, actValue)
			}
		}

		// Should only have two new keys.
		if actCount != expCount+2 {
			return fmt.Errorf("expected %d new storage slots, but got %d", expCount+2, actCount)
		}

		// Check that implementation slot is new.
		_, expOk1 := expSlots[ImplementationSlot]
		_, actOk1 := actSlots[ImplementationSlot]
		if expOk1 || !actOk1 {
			return fmt.Errorf("expected implementation slot to be new")
		}

		// Check that admin slot is new.
		_, expOk2 := expSlots[AdminSlot]
		_, actOk2 := actSlots[AdminSlot]
		if expOk2 || !actOk2 {
			return fmt.Errorf("expected admin slot to be new")
		}

		log.Info("checked storage", "address", addr, "expCount", expCount, "actCount", actCount)
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

		// There must be an admin
		admin := db.GetState(addr, AdminSlot)
		adminAddr := common.BytesToAddress(admin.Bytes())
		if adminAddr != predeploys.HardforkOnlyProxyOwnerAddr {
			return fmt.Errorf("expected admin for %s to be %s but got %s", addr, predeploys.HardforkOnlyProxyOwnerAddr, adminAddr)
		}
	}

	// For each predeploy, check that we've set the implementation correctly when
	// necessary and that there's code at the implementation.
	for _, proxyAddr := range predeploys.Predeploys {
		if *proxyAddr == predeploys.LegacyERC20ETHAddr {
			log.Trace("skipping legacy eth predeploy")
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
func PostCheckPredeployStorage(db vm.StateDB, finalSystemOwner common.Address) error {
	for name, addr := range predeploys.Predeploys {
		if addr == nil {
			return fmt.Errorf("nil address in predeploys mapping for %s", name)
		}

		// Skip the addresses that did not have their storage reset, also skip the
		// L2ToL1MessagePasser because it's already covered by the withdrawals check.
		if FrozenStoragePredeploys[*addr] || *addr == predeploys.L2ToL1MessagePasserAddr || *addr == predeploys.L1BlockAddr {
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
			log.Debug("storage values", "key", key.String(), "value", value.String())
		}

		expSlots := ExpectedStorageSlots[*addr]
		// Assert that the correct number of slots are present.
		if len(expSlots) != len(slots) {
			return fmt.Errorf("expected %d storage slots for %s but got %d", len(expSlots), name, len(slots))
		}

		for key, value := range expSlots {
			// The owner slots for the L2XDM and ProxyAdmin are special cases.
			// They are set to the final system owner in the config.
			if *addr == predeploys.L2CrossDomainMessengerAddr && key == L2XDMOwnerSlot {
				actualOwner := common.BytesToAddress(slots[key].Bytes())
				if actualOwner != finalSystemOwner {
					return fmt.Errorf("expected owner for %s to be %s but got %s", name, finalSystemOwner, actualOwner)
				}
				log.Debug("validated special case owner slot", "value", actualOwner, "name", name)
				continue
			}

			if slots[key] != value {
				log.Debug("validated storage value", "key", key.String(), "value", value.String())
				return fmt.Errorf("expected storage slot %s to be %s but got %s", key, value, slots[key])
			}
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

// PostCheckL1Block checks that the L1Block contract was properly set to the L1 origin.
func PostCheckL1Block(db vm.StateDB, info *derive.L1BlockInfo) error {
	// Slot 0 is the concatenation of the block number and timestamp
	data := db.GetState(predeploys.L1BlockAddr, common.Hash{}).Bytes()
	blockNumber := binary.BigEndian.Uint64(data[24:])
	timestamp := binary.BigEndian.Uint64(data[16:24])
	if blockNumber != info.Number {
		return fmt.Errorf("expected L1Block block number to be %d, but got %d", info.Number, blockNumber)
	}
	log.Debug("validated L1Block block number", "expected", info.Number)
	if timestamp != info.Time {
		return fmt.Errorf("expected L1Block timestamp to be %d, but got %d", info.Time, timestamp)
	}
	log.Debug("validated L1Block timestamp", "expected", info.Time)

	// Slot 1 is the basefee.
	baseFee := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x01}).Big()
	if baseFee.Cmp(info.BaseFee) != 0 {
		return fmt.Errorf("expected L1Block basefee to be %s, but got %s", info.BaseFee, baseFee)
	}
	log.Debug("validated L1Block basefee", "expected", info.BaseFee)

	// Slot 2 is the block hash
	hash := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x02})
	if hash != info.BlockHash {
		return fmt.Errorf("expected L1Block hash to be %s, but got %s", info.BlockHash, hash)
	}
	log.Debug("validated L1Block hash", "expected", info.BlockHash)

	// Slot 3 is the sequence number. It is expected to be zero.
	sequenceNumber := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x03})
	expSequenceNumber := common.Hash{}
	if expSequenceNumber != sequenceNumber {
		return fmt.Errorf("expected L1Block sequence number to be %s, but got %s", expSequenceNumber, sequenceNumber)
	}
	log.Debug("validated L1Block sequence number", "expected", expSequenceNumber)

	// Slot 4 is the versioned hash to authenticate the batcher. It is expected to be the initial batch sender.
	batcherHash := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x04})
	batchSender := common.BytesToAddress(batcherHash.Bytes())
	if batchSender != info.BatcherAddr {
		return fmt.Errorf("expected L1Block batcherHash to be %s, but got %s", info.BatcherAddr, batchSender)
	}
	log.Debug("validated L1Block batcherHash", "expected", info.BatcherAddr)

	// Slot 5 is the L1 fee overhead.
	l1FeeOverhead := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x05})
	if !bytes.Equal(l1FeeOverhead.Bytes(), info.L1FeeOverhead[:]) {
		return fmt.Errorf("expected L1Block L1FeeOverhead to be %s, but got %s", info.L1FeeOverhead, l1FeeOverhead)
	}
	log.Debug("validated L1Block L1FeeOverhead", "expected", info.L1FeeOverhead)

	// Slot 6 is the L1 fee scalar.
	l1FeeScalar := db.GetState(predeploys.L1BlockAddr, common.Hash{31: 0x06})
	if !bytes.Equal(l1FeeScalar.Bytes(), info.L1FeeScalar[:]) {
		return fmt.Errorf("expected L1Block L1FeeScalar to be %s, but got %s", info.L1FeeScalar, l1FeeScalar)
	}
	log.Debug("validated L1Block L1FeeScalar", "expected", info.L1FeeScalar)

	// Check EIP-1967
	proxyAdmin := common.BytesToAddress(db.GetState(predeploys.L1BlockAddr, AdminSlot).Bytes())
	if proxyAdmin != predeploys.HardforkOnlyProxyOwnerAddr {
		return fmt.Errorf("expected L1Block admin to be %s, but got %s", predeploys.HardforkOnlyProxyOwnerAddr, proxyAdmin)
	}
	log.Debug("validated L1Block admin", "expected", predeploys.HardforkOnlyProxyOwnerAddr)
	expImplementation, err := AddressToCodeNamespace(predeploys.L1BlockAddr)
	if err != nil {
		return fmt.Errorf("failed to get expected implementation for L1Block: %w", err)
	}
	actImplementation := common.BytesToAddress(db.GetState(predeploys.L1BlockAddr, ImplementationSlot).Bytes())
	if expImplementation != actImplementation {
		return fmt.Errorf("expected L1Block implementation to be %s, but got %s", expImplementation, actImplementation)
	}
	log.Debug("validated L1Block implementation", "expected", expImplementation)

	var count int
	err = db.ForEachStorage(predeploys.L1BlockAddr, func(key, value common.Hash) bool {
		count++
		return true
	})
	if err != nil {
		return fmt.Errorf("failed to iterate over L1Block storage: %w", err)
	}
	if count != 8 {
		return fmt.Errorf("expected L1Block to have 8 storage slots, but got %d", count)
	}
	log.Debug("validated L1Block storage slot count", "expected", 8)

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

func eip1967Slots(address common.Address) StorageCheckMap {
	codeAddr, err := AddressToCodeNamespace(address)
	if err != nil {
		panic(err)
	}
	return StorageCheckMap{
		AdminSlot:          predeploys.HardforkOnlyProxyOwnerAddr.Hash(),
		ImplementationSlot: codeAddr.Hash(),
	}
}
