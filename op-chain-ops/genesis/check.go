package genesis

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"

	"github.com/ethereum-optimism/optimism/op-chain-ops/util"

	"github.com/ethereum-optimism/optimism/op-chain-ops/ether"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

const (
	// MaxPredeploySlotChecks is the maximum number of storage slots to check
	// when validating the untouched predeploys. This limit is in place
	// to bound execution time of the migration. We can parallelize this
	// in the future.
	MaxPredeploySlotChecks = 1000

	// MaxOVMETHSlotChecks is the maximum number of OVM ETH storage slots to check
	// when validating the OVM ETH migration.
	MaxOVMETHSlotChecks = 5000

	// OVMETHSampleLikelihood is the probability that a storage slot will be checked
	// when validating the OVM ETH migration.
	OVMETHSampleLikelihood = 0.1
)

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
		// ProxyAdmin is not a proxy, and only has the _owner slot set.
		predeploys.ProxyAdminAddr: {
			// Slot 0x00 (0) is _owner. Requires custom check, so set to a garbage value
			ProxyAdminOwnerSlot: common.HexToHash("0xbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbad0"),

			// EIP-1967 storage slots
			AdminSlot:          common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: common.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30018"),
		},
		predeploys.BaseFeeVaultAddr: eip1967Slots(predeploys.BaseFeeVaultAddr),
		predeploys.L1FeeVaultAddr:   eip1967Slots(predeploys.L1FeeVaultAddr),
	}
)

// PostCheckMigratedDB will check that the migration was performed correctly
func PostCheckMigratedDB(
	ldb ethdb.Database,
	migrationData crossdomain.MigrationData,
	l1XDM *common.Address,
	l1ChainID uint64,
	finalSystemOwner common.Address,
	proxyAdminOwner common.Address,
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

	prevDB, err := state.New(prevHeader.Root, underlyingDB, nil)
	if err != nil {
		return fmt.Errorf("cannot open historical StateDB: %w", err)
	}

	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	if err := PostCheckPredeployStorage(db, finalSystemOwner, proxyAdminOwner); err != nil {
		return err
	}
	log.Info("checked predeploy storage")

	if err := PostCheckUntouchables(underlyingDB, db, prevHeader.Root, l1ChainID); err != nil {
		return err
	}
	log.Info("checked untouchables")

	if err := PostCheckPredeploys(prevDB, db); err != nil {
		return err
	}
	log.Info("checked predeploys")

	if err := PostCheckL1Block(db, info); err != nil {
		return err
	}
	log.Info("checked L1Block")

	if err := PostCheckLegacyETH(prevDB, db, migrationData); err != nil {
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
		var prevRoot, currRoot common.Hash
		prevStorage, err := prevDB.StorageTrie(addr)
		if err != nil {
			return fmt.Errorf("failed to open previous-db storage trie of %s: %w", addr, err)
		}
		if prevStorage == nil {
			prevRoot = types.EmptyRootHash
		} else {
			prevRoot = prevStorage.Hash()
		}
		currStorage, err := currDB.StorageTrie(addr)
		if err != nil {
			return fmt.Errorf("failed to open current-db storage trie of %s: %w", addr, err)
		}
		if currStorage == nil {
			currRoot = types.EmptyRootHash
		} else {
			currRoot = currStorage.Hash()
		}
		if prevRoot != currRoot {
			return fmt.Errorf("expected storage root for %s to be %s, but got %s", addr, prevRoot, currRoot)
		}
		log.Info("checked account roots", "address", addr, "curr_root", currRoot, "prev_root", prevRoot)

		// Sample storage slots to ensure that they are not modified.
		var count int
		expSlots := make(map[common.Hash]common.Hash)
		if err := prevDB.ForEachStorage(addr, func(key, value common.Hash) bool {
			count++
			expSlots[key] = value
			return count < MaxPredeploySlotChecks
		}); err != nil {
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
func PostCheckPredeploys(prevDB, currDB *state.StateDB) error {
	for i := uint64(0); i <= 2048; i++ {
		// Compute the predeploy address
		bigAddr := new(big.Int).Or(bigL2PredeployNamespace, new(big.Int).SetUint64(i))
		addr := common.BigToAddress(bigAddr)
		// Get the code for the predeploy
		code := currDB.GetCode(addr)
		// There must be code for the predeploy
		if len(code) == 0 {
			return fmt.Errorf("no code found at predeploy %s", addr)
		}

		if UntouchablePredeploys[addr] {
			log.Trace("skipping untouchable predeploy", "address", addr)
			continue
		}

		// There must be an admin
		admin := currDB.GetState(addr, AdminSlot)
		adminAddr := common.BytesToAddress(admin.Bytes())
		if addr != predeploys.ProxyAdminAddr && addr != predeploys.GovernanceTokenAddr && adminAddr != predeploys.ProxyAdminAddr {
			return fmt.Errorf("expected admin for %s to be %s but got %s", addr, predeploys.ProxyAdminAddr, adminAddr)
		}

		// Balances and nonces should match legacy
		oldNonce := prevDB.GetNonce(addr)
		oldBalance := ether.GetOVMETHBalance(prevDB, addr)
		newNonce := currDB.GetNonce(addr)
		newBalance := currDB.GetBalance(addr)
		if oldNonce != newNonce {
			return fmt.Errorf("expected nonce for %s to be %d but got %d", addr, oldNonce, newNonce)
		}
		if oldBalance.Cmp(newBalance) != 0 {
			return fmt.Errorf("expected balance for %s to be %d but got %d", addr, oldBalance, newBalance)
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
			implCode := currDB.GetCode(*proxyAddr)
			if len(implCode) == 0 {
				return errors.New("no code found at proxy admin")
			}
			continue
		}

		expImplAddr, err := AddressToCodeNamespace(*proxyAddr)
		if err != nil {
			return fmt.Errorf("error converting to code namespace: %w", err)
		}

		implCode := currDB.GetCode(expImplAddr)
		if len(implCode) == 0 {
			return fmt.Errorf("no code found at predeploy impl %s", *proxyAddr)
		}

		impl := currDB.GetState(*proxyAddr, ImplementationSlot)
		actImplAddr := common.BytesToAddress(impl.Bytes())
		if expImplAddr != actImplAddr {
			return fmt.Errorf("expected implementation for %s to be at %s, but got %s", *proxyAddr, expImplAddr, actImplAddr)
		}
	}

	return nil
}

// PostCheckPredeployStorage will ensure that the predeploys had their storage
// wiped correctly.
func PostCheckPredeployStorage(db vm.StateDB, finalSystemOwner common.Address, proxyAdminOwner common.Address) error {
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
			if *addr == predeploys.ProxyAdminAddr && key == ProxyAdminOwnerSlot {
				actualOwner := common.BytesToAddress(slots[key].Bytes())
				if actualOwner != proxyAdminOwner {
					return fmt.Errorf("expected owner for %s to be %s but got %s", name, proxyAdminOwner, actualOwner)
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
// It checks that the total supply was set to 0, and randomly samples storage
// slots pre- and post-migration to ensure that balances were correctly migrated.
func PostCheckLegacyETH(prevDB, migratedDB *state.StateDB, migrationData crossdomain.MigrationData) error {
	allowanceSlots := make(map[common.Hash]bool)
	addresses := make(map[common.Hash]common.Address)

	log.Info("recomputing witness data")
	for _, allowance := range migrationData.OvmAllowances {
		key := ether.CalcAllowanceStorageKey(allowance.From, allowance.To)
		allowanceSlots[key] = true
	}

	for _, addr := range migrationData.Addresses() {
		addresses[ether.CalcOVMETHStorageKey(addr)] = addr
	}

	log.Info("checking legacy eth fixed storage slots")
	for slot, expValue := range LegacyETHCheckSlots {
		actValue := migratedDB.GetState(predeploys.LegacyERC20ETHAddr, slot)
		if actValue != expValue {
			return fmt.Errorf("expected slot %s on %s to be %s, but got %s", slot, predeploys.LegacyERC20ETHAddr, expValue, actValue)
		}
	}

	var count int
	threshold := 100 - int(100*OVMETHSampleLikelihood)
	progress := util.ProgressLogger(100, "checking legacy eth balance slots")
	var innerErr error
	err := prevDB.ForEachStorage(predeploys.LegacyERC20ETHAddr, func(key, value common.Hash) bool {
		val := rand.Intn(100)

		// Randomly sample storage slots.
		if val > threshold {
			return true
		}

		// Ignore fixed slots.
		if _, ok := LegacyETHCheckSlots[key]; ok {
			return true
		}

		// Ignore allowances.
		if allowanceSlots[key] {
			return true
		}

		// Grab the address, and bail if we can't find it.
		addr, ok := addresses[key]
		if !ok {
			innerErr = fmt.Errorf("unknown OVM_ETH storage slot %s", key)
			return false
		}

		// Pull out the pre-migration OVM ETH balance, and the state balance.
		ovmETHBalance := value.Big()
		ovmETHStateBalance := prevDB.GetBalance(addr)
		// Pre-migration state balance should be zero.
		if ovmETHStateBalance.Cmp(common.Big0) != 0 {
			innerErr = fmt.Errorf("expected OVM_ETH pre-migration state balance for %s to be 0, but got %s", addr, ovmETHStateBalance)
			return false
		}

		// Migrated state balance should equal the OVM ETH balance.
		migratedStateBalance := migratedDB.GetBalance(addr)
		if migratedStateBalance.Cmp(ovmETHBalance) != 0 {
			innerErr = fmt.Errorf("expected OVM_ETH post-migration state balance for %s to be %s, but got %s", addr, ovmETHStateBalance, migratedStateBalance)
			return false
		}
		// Migrated OVM ETH balance should be zero, since we wipe the slots.
		migratedBalance := migratedDB.GetState(predeploys.LegacyERC20ETHAddr, key)
		if migratedBalance.Big().Cmp(common.Big0) != 0 {
			innerErr = fmt.Errorf("expected OVM_ETH post-migration ERC20 balance for %s to be 0, but got %s", addr, migratedBalance)
			return false
		}

		progress()
		count++

		// Stop iterating if we've checked enough slots.
		return count < MaxOVMETHSlotChecks
	})
	if err != nil {
		return fmt.Errorf("error iterating over OVM_ETH storage: %w", err)
	}
	if innerErr != nil {
		return innerErr
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
	if proxyAdmin != predeploys.ProxyAdminAddr {
		return fmt.Errorf("expected L1Block admin to be %s, but got %s", predeploys.ProxyAdminAddr, proxyAdmin)
	}
	log.Debug("validated L1Block admin", "expected", predeploys.ProxyAdminAddr)
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

func CheckWithdrawalsAfter(db vm.StateDB, data crossdomain.MigrationData, l1CrossDomainMessenger *common.Address) error {
	wds, invalidMessages, err := data.ToWithdrawals()
	if err != nil {
		return err
	}

	// First, make a mapping between old withdrawal slots and new ones.
	// This list can be a superset of what was actually migrated, since
	// some witness data may references withdrawals that reverted.
	oldToNewSlots := make(map[common.Hash]common.Hash)
	wdsByOldSlot := make(map[common.Hash]*crossdomain.LegacyWithdrawal)
	invalidMessagesByOldSlot := make(map[common.Hash]crossdomain.InvalidMessage)
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

		oldToNewSlots[legacySlot] = migratedSlot
		wdsByOldSlot[legacySlot] = wd
	}
	for _, im := range invalidMessages {
		invalidSlot, err := im.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute legacy storage slot: %w", err)
		}
		invalidMessagesByOldSlot[invalidSlot] = im
	}

	log.Info("computed withdrawal storage slots", "migrated", len(oldToNewSlots), "invalid", len(invalidMessagesByOldSlot))

	// Now, iterate over each legacy withdrawal and check if there is a corresponding
	// migrated withdrawal.
	var innerErr error
	progress := util.ProgressLogger(1000, "checking withdrawals")
	err = db.ForEachStorage(predeploys.LegacyMessagePasserAddr, func(key, value common.Hash) bool {
		progress()
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

		// Make sure invalid slots don't get migrated.
		_, isInvalidSlot := invalidMessagesByOldSlot[key]
		if isInvalidSlot {
			value := db.GetState(predeploys.L2ToL1MessagePasserAddr, key)
			if value != abiFalse {
				innerErr = fmt.Errorf("expected invalid slot not to be migrated, but got %s", value)
				return false
			}
			return true
		}

		// Grab the migrated slot.
		migratedSlot := oldToNewSlots[key]
		if migratedSlot == (common.Hash{}) {
			innerErr = fmt.Errorf("no migrated slot found for legacy slot %s", key)
			return false
		}

		// Look up the migrated slot in the DB.
		migratedValue := db.GetState(predeploys.L2ToL1MessagePasserAddr, migratedSlot)

		// If the sender is _not_ the L2XDM, the value should not be migrated.
		wd := wdsByOldSlot[key]
		if wd.MessageSender == predeploys.L2CrossDomainMessengerAddr {
			// Make sure the value is abiTrue if this withdrawal should be migrated.
			if migratedValue != abiTrue {
				innerErr = fmt.Errorf("expected migrated value to be true, but got %s", migratedValue)
				return false
			}
		} else {
			// Otherwise, ensure that withdrawals from senders other than the L2XDM are _not_ migrated.
			if migratedValue != abiFalse {
				innerErr = fmt.Errorf("a migration from a sender other than the L2XDM was migrated. sender: %s, migrated value: %s", wd.MessageSender, migratedValue)
				return false
			}
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
		AdminSlot:          predeploys.ProxyAdminAddr.Hash(),
		ImplementationSlot: codeAddr.Hash(),
	}
}
