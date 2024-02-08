package genesis

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync"

	"github.com/bobanetwork/v3-anchorage/boba-bindings/predeploys"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/chain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/crossdomain"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/ether"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/state"
	"github.com/bobanetwork/v3-anchorage/boba-chain-ops/util"
	libcommon "github.com/ledgerwatch/erigon-lib/common"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/core/rawdb"
	erigonstate "github.com/ledgerwatch/erigon/core/state"
	"github.com/ledgerwatch/erigon/core/types"
	"github.com/ledgerwatch/log/v3"
)

var (
	abiTrue  = libcommon.Hash{31: 0x01}
	abiFalse = libcommon.Hash{}
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

type StorageCheckMap = map[libcommon.Hash]libcommon.Hash

var (
	L2XDMOwnerSlot      = libcommon.Hash{31: 0x33}
	ProxyAdminOwnerSlot = libcommon.Hash{}

	LegacyETHCheckSlots = map[libcommon.Hash]libcommon.Hash{
		// Bridge
		{31: 0x06}: libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000010"),
		// L1 token address
		// This is only applied to alt l2s (bobabeam, bobaopera)
		{31: 0x05}: {},
		// Symbol
		{31: 0x04}: libcommon.HexToHash("0x4554480000000000000000000000000000000000000000000000000000000006"),
		// Name
		{31: 0x03}: libcommon.HexToHash("0x457468657200000000000000000000000000000000000000000000000000000a"),
		// Total supply
		{31: 0x02}: {},
		// Admin slot
		AdminSlot: libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
	}

	// ExpectedStorageSlots is a map of predeploy addresses to the storage slots and values that are
	// expected to be set in those predeploys after the migration. It does not include any predeploys
	// that were not wiped. It also accounts for the 2 EIP-1967 storage slots in each contract.
	// It does _not_ include L1Block. L1Block is checked separately.
	ExpectedStorageSlots = map[libcommon.Address]StorageCheckMap{
		predeploys.L2CrossDomainMessengerAddr: {
			// Slot 0x00 (0) is a combination of spacer_0_0_20, _initialized, and _initializing
			libcommon.Hash{}: libcommon.HexToHash("0x0000000000000000000000010000000000000000000000000000000000000000"),
			// Slot 0xcc (204) is xDomainMsgSender
			libcommon.Hash{31: 0xcc}: libcommon.HexToHash("0x000000000000000000000000000000000000000000000000000000000000dead"),
			// EIP-1967 storage slots
			AdminSlot:          libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: libcommon.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30007"),
		},
		predeploys.L2StandardBridgeAddr: {
			// Slot 0x00 (0) is a combination of spacer_0_0_20, _initialized, and _initializing
			libcommon.Hash{}: libcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
			// EIP-1967 storage slots
			AdminSlot:          libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: libcommon.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30010"),
		},
		predeploys.SequencerFeeVaultAddr: eip1967Slots(predeploys.SequencerFeeVaultAddr),
		predeploys.OptimismMintableERC20FactoryAddr: {
			// EIP-1967 storage slots
			AdminSlot:          libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: libcommon.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30012"),
		},
		predeploys.L1BlockNumberAddr:  eip1967Slots(predeploys.L1BlockNumberAddr),
		predeploys.GasPriceOracleAddr: eip1967Slots(predeploys.GasPriceOracleAddr),
		predeploys.L2ERC721BridgeAddr: {
			libcommon.Hash{}: libcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000002"),
			// EIP-1967 storage slots
			AdminSlot:          libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: libcommon.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30014"),
		},
		predeploys.OptimismMintableERC721FactoryAddr: eip1967Slots(predeploys.OptimismMintableERC721FactoryAddr),
		// ProxyAdmin is not a proxy, and only has the _owner slot set.
		predeploys.ProxyAdminAddr: {
			// Slot 0x00 (0) is _owner. Requires custom check, so set to a garbage value
			ProxyAdminOwnerSlot: libcommon.HexToHash("0xbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbad0"),
			// EIP-1967 storage slots
			AdminSlot:          libcommon.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000018"),
			ImplementationSlot: libcommon.HexToHash("0x000000000000000000000000c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d3c0d30018"),
		},
		predeploys.BaseFeeVaultAddr:   eip1967Slots(predeploys.BaseFeeVaultAddr),
		predeploys.L1FeeVaultAddr:     eip1967Slots(predeploys.L1FeeVaultAddr),
		predeploys.EASAddr:            eip1967Slots(predeploys.EASAddr),
		predeploys.SchemaRegistryAddr: eip1967Slots(predeploys.SchemaRegistryAddr),
	}
)

func PostCheckMigratedDB(
	chaindb kv.RwDB,
	g *types.Genesis,
	migrationData crossdomain.MigrationData,
	l1XDM *libcommon.Address,
	l1ChainID uint64,
	finalSystemOwner libcommon.Address,
	proxyAdminOwner libcommon.Address,
	transitionBlockNumber uint64,
	timestamp int,
	info *L1BlockInfo,
) error {
	log.Info("Validating database migration")

	tx, err := chaindb.BeginRo(context.Background())
	if err != nil {
		log.Error("failed to read DB", "err", err)
		return err
	}
	defer tx.Rollback()

	hash := rawdb.ReadHeadHeaderHash(tx)
	log.Info("Reading chain tip from database", "hash", hash)
	num := rawdb.ReadHeaderNumber(tx, hash)
	if num == nil {
		return fmt.Errorf("cannot find header number for %s", hash)
	}

	if *num != transitionBlockNumber {
		return fmt.Errorf("expected transition block number to be %d, but got %d", transitionBlockNumber, *num)
	}

	header := rawdb.ReadHeader(tx, hash, *num)
	log.Info("Read header from database", "number", *num)
	if header.GasLimit != g.GasLimit {
		return fmt.Errorf("expected gas limit to be %d, but got %d", g.GasLimit, header.GasLimit)
	}
	if header.Time != uint64(timestamp) {
		return fmt.Errorf("expected timestamp to be %d, but got %d", timestamp, header.Time)
	}
	parentHeader := rawdb.ReadHeader(tx, header.ParentHash, header.Number.Uint64()-1)
	if parentHeader == nil {
		return fmt.Errorf("cannot find parent header for %s", header.ParentHash)
	}

	genesisHeader := rawdb.ReadHeaderByNumber(tx, 0)
	if err != nil {
		return fmt.Errorf("failed to read genesis header from database: %w", err)
	}

	bobaGenesisHash := libcommon.HexToHash(chain.GetBobaGenesisHash(g.Config.ChainID))
	if genesisHeader.Hash() != bobaGenesisHash {
		return fmt.Errorf("expected chain tip to be %s, but got %s", bobaGenesisHash, genesisHeader.Hash())
	}

	bobaGenesisExtraData := libcommon.Hex2Bytes(chain.GetBobaGenesisExtraData(g.Config.ChainID))
	if !bytes.Equal(genesisHeader.Extra, bobaGenesisExtraData) {
		return fmt.Errorf("expected extra data to be %x, but got %x", bobaGenesisExtraData, genesisHeader.Extra)
	}

	chainConfig, err := rawdb.ReadChainConfig(tx, genesisHeader.Hash())
	if err != nil {
		return fmt.Errorf("failed to read chain config from database: %w", err)
	}
	if chainConfig.BedrockBlock.Uint64() != transitionBlockNumber {
		return fmt.Errorf("expected bedrock block to be %d, but got %d", transitionBlockNumber, chainConfig.BedrockBlock)
	}

	if err := CheckPreBedrockAllocation(tx); err != nil {
		return err
	}

	if err := PostCheckPredeployStorage(tx, finalSystemOwner, proxyAdminOwner); err != nil {
		return err
	}
	log.Info("checked predeploy storage")

	if err := PostCheckUntouchables(tx, g); err != nil {
		return err
	}
	log.Info("checked untouchables")

	if err := PostCheckPredeploys(tx, g); err != nil {
		return err
	}
	log.Info("checked predeploys")

	if err := PostCheckL1Block(tx, info); err != nil {
		return err
	}
	log.Info("checked L1Block")

	if err := PostCheckLegacyETH(tx, g, migrationData); err != nil {
		return err
	}
	log.Info("checked legacy eth")

	if err := CheckWithdrawalsAfter(tx, migrationData, l1XDM); err != nil {
		return err
	}
	log.Info("checked withdrawals")

	return nil
}

// PostCheckUntouchables will check that the untouchable contracts have
// not been modified by the migration process.
func PostCheckUntouchables(tx kv.Tx, g *types.Genesis) error {
	for addr := range UntouchablePredeploys {
		// Check that the code is the same.
		hash, err := state.GetContractCodeHash(tx, addr)
		if err != nil {
			return fmt.Errorf("failed to read code hash from database: %w", err)
		}
		expHash := UntouchableCodeHashes[addr][g.Config.ChainID.Uint64()]
		if *hash != expHash {
			return fmt.Errorf("expected code hash for %s to be %s, but got %s", addr, expHash, hash)
		}
		log.Info("checked code hash", "address", addr, "hash", hash)

		// Sample storage slots to ensure that they are not modified.
		var count int
		expSlots := make(map[libcommon.Hash]libcommon.Hash)
		for key, val := range g.Alloc[addr].Storage {
			count++
			expSlots[key] = val
			if count >= MaxPredeploySlotChecks {
				break
			}
		}

		for expKey, expValue := range expSlots {
			hash, err := state.GetStorage(tx, addr, expKey)
			if err != nil {
				return fmt.Errorf("failed to read storage from database: %w", err)
			}
			if *hash != expValue {
				return fmt.Errorf("expected slot %s on %s to be %s, but got %s", expKey, addr, expValue, hash)
			}
		}

		log.Info("checked storage", "address", addr, "count", count)
	}
	return nil
}

// PostCheckPredeploys will check that there is code at each predeploy
// address
func PostCheckPredeploys(tx kv.Tx, g *types.Genesis) error {
	for i := uint64(0); i <= 2048; i++ {
		// Compute the predeploy address
		bigAddr := new(big.Int).Or(BigL2PredeployNamespace, new(big.Int).SetUint64(i))
		addr := libcommon.BigToAddress(bigAddr)

		// Get the code for the predeploy
		code, err := state.GetContractCode(tx, addr)
		if err != nil {
			return fmt.Errorf("failed to read code from database: %w", err)
		}

		// There must be code for the predeploy
		if code == nil || len(code) == 0 {
			return fmt.Errorf("no code found at predeploy %s", addr)
		}

		if UntouchablePredeploys[addr] {
			log.Trace("skipping untouchable predeploy", "address", addr)
			continue
		}

		// There must be an admin
		hash, err := state.GetStorage(tx, addr, AdminSlot)
		if err != nil {
			return fmt.Errorf("failed to read admin from database: %w", err)
		}
		adminAddr := libcommon.BytesToAddress(hash[:])
		if addr != predeploys.ProxyAdminAddr && adminAddr != predeploys.ProxyAdminAddr {
			return fmt.Errorf("expected admin for %s to be %s but got %s", addr, predeploys.ProxyAdminAddr, adminAddr)
		}

		// Balances and nonces should match legacy
		oldNonce := g.Alloc[addr].Nonce
		oldBalance := g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[ether.CalcOVMETHStorageKey(addr)].Big()
		if oldBalance == nil {
			oldBalance = new(big.Int)
		}

		var (
			newNonce   []byte
			newBalance []byte
		)
		storageByte, err := state.GetAccount(tx, addr)
		if err != nil {
			return fmt.Errorf("failed to read account from database: %w", err)
		}
		newNonce, newBalance, err = decodeNonceBalanceFromStorage(storageByte)
		if err != nil {
			return fmt.Errorf("failed to decode nonce and balance from storage: %w", err)
		}
		if oldNonce != bytesToUint64(newNonce) {
			return fmt.Errorf("expected nonce for %s to be %d but got %d", addr, oldNonce, newNonce)
		}
		if oldBalance.Cmp(new(big.Int).SetBytes(newBalance)) != 0 {
			return fmt.Errorf("expected balance for %s to be %d but got %d", addr, oldBalance, new(big.Int).SetBytes(newBalance))
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
				implCode, err := state.GetContractCode(tx, *proxyAddr)
				if err != nil {
					return fmt.Errorf("failed to read contract code from database: %w", err)
				}
				if implCode == nil || len(implCode) == 0 {
					return errors.New("no code found at proxy admin")
				}
				continue
			}

			expImplAddr, err := AddressToCodeNamespace(*proxyAddr)
			if err != nil {
				return fmt.Errorf("error converting to code namespace: %w", err)
			}

			implCode, err := state.GetContractCode(tx, *proxyAddr)
			if err != nil {
				return fmt.Errorf("failed to read contract code from database: %w", err)
			}
			if implCode == nil || len(implCode) == 0 {
				return fmt.Errorf("no code found at predeploy impl %s", *proxyAddr)
			}

			actImplAddr, err := state.GetStorage(tx, *proxyAddr, ImplementationSlot)
			if err != nil {
				return fmt.Errorf("failed to read implementation from database: %w", err)
			}
			if expImplAddr != libcommon.HexToAddress(actImplAddr.Hex()) {
				return fmt.Errorf("expected implementation for %s to be at %s, but got %s", *proxyAddr, expImplAddr, actImplAddr)
			}
		}
	}

	return nil
}

// PostCheckPredeployStorage will ensure that the predeploys had their storage
// wiped correctly.
func PostCheckPredeployStorage(tx kv.Tx, finalSystemOwner libcommon.Address, proxyAdminOwner libcommon.Address) error {
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
		slots := make(map[libcommon.Hash]libcommon.Hash)
		cursor, err := tx.Cursor(kv.PlainState)
		if err != nil {
			return fmt.Errorf("failed to create cursor: %w", err)
		}
		defer cursor.Close()

		for k, v, err := cursor.First(); k != nil; k, v, err = cursor.Next() {
			if err != nil {
				return fmt.Errorf("failed to iterate cursor: %w", err)
			}
			// Storage is 20 bytes account address + 8 byte incarnation + 32 byte storage key
			if len(k) == 60 && libcommon.BytesToAddress(k[:20]) == *addr {
				slots[libcommon.BytesToHash(k[20:])] = libcommon.BytesToHash(v)
			}
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
				actualOwner := libcommon.BytesToAddress(slots[key].Bytes())
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
func PostCheckLegacyETH(tx kv.Tx, g *types.Genesis, migrationData crossdomain.MigrationData) error {
	allowanceSlots := make(map[libcommon.Hash]bool)
	addresses := make(map[libcommon.Hash]libcommon.Address)

	log.Info("recomputing witness data")
	for _, allowance := range migrationData.OvmAllowances {
		key := ether.CalcAllowanceStorageKey(allowance.From, allowance.To)
		allowanceSlots[key] = true
	}

	for _, addr := range migrationData.Addresses() {
		addresses[ether.CalcOVMETHStorageKey(addr)] = addr
	}

	// This is for bobabeam and bobaopera
	// We don't touch any of the old slots except the balance slots
	if crossdomain.CustomLegacyETHSlotCheck[int(g.Config.ChainID.Int64())] {
		log.Info("checking legacy eth fixed storage slots for custom chain", "chainID", g.Config.ChainID)
		defaultSlots := []libcommon.Hash{
			libcommon.BytesToHash([]byte{2}),
			libcommon.BytesToHash([]byte{3}),
			libcommon.BytesToHash([]byte{4}),
			libcommon.BytesToHash([]byte{5}),
			libcommon.BytesToHash([]byte{6}),
		}
		for _, slot := range defaultSlots {
			if g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[slot] != (libcommon.Hash{}) {
				actValue, err := state.GetStorage(tx, predeploys.LegacyERC20ETHAddr, slot)
				if err != nil {
					return fmt.Errorf("failed to get storage for %s: %w", slot, err)
				}
				// The total supply should be 0
				if slot == libcommon.BytesToHash([]byte{2}) {
					if *actValue != (libcommon.Hash{}) {
						return fmt.Errorf("expected slot %s on %s to be %s, but got %s", slot, predeploys.LegacyERC20ETHAddr, (libcommon.Hash{}), actValue)
					}
					continue
				}
				if *actValue != g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[slot] {
					return fmt.Errorf("expected slot %s on %s to be %s, but got %s", slot, predeploys.LegacyERC20ETHAddr, g.Alloc[predeploys.LegacyERC20ETHAddr].Storage[slot], actValue)
				}
			}
		}
	} else {
		log.Info("checking legacy eth fixed storage slots", "chainID", g.Config.ChainID)
		for slot, expValue := range LegacyETHCheckSlots {
			actValue, err := state.GetStorage(tx, predeploys.LegacyERC20ETHAddr, slot)
			if err != nil {
				return fmt.Errorf("failed to get storage for %s: %w", slot, err)
			}
			if *actValue != expValue {
				return fmt.Errorf("expected slot %s on %s to be %s, but got %s", slot, predeploys.LegacyERC20ETHAddr, expValue, actValue)
			}
		}
	}

	var (
		count    int
		innerErr error
		m        sync.Mutex
	)

	threshold := 100 - int(100*OVMETHSampleLikelihood)
	progress := util.ProgressLogger(100, "checking legacy eth balance slots")

	err := ether.IterateState(g, func(key, value libcommon.Hash) error {
		// Stop iterating if we've checked enough slots.
		if count >= MaxOVMETHSlotChecks {
			return nil
		}

		val := rand.Intn(100)

		// Randomly sample storage slots.
		if val > threshold {
			return nil
		}

		// Ignore fixed slots.
		if _, ok := LegacyETHCheckSlots[key]; ok {
			return nil
		}

		// Ignore allowances.
		if allowanceSlots[key] {
			return nil
		}

		// Grab the address, and bail if we can't find it.
		addr, ok := addresses[key]
		if !ok {
			innerErr = fmt.Errorf("unknown OVM_ETH storage slot %s", key)
			return innerErr
		}

		// Pull out the pre-migration OVM ETH balance, and the state balance.
		ovmETHBalance := value.Big()
		ovmETHStateBalance := g.Alloc[addr].Balance
		if ovmETHStateBalance == nil {
			ovmETHStateBalance = libcommon.Big0
		}
		// Pre-migration state balance should be zero.
		if ovmETHStateBalance.Cmp(libcommon.Big0) != 0 {
			log.Info("Found mismatched OVM_ETH pre-migration state balance", "key", ether.CalcOVMETHStorageKey(addr))
			innerErr = fmt.Errorf("expected OVM_ETH pre-migration state balance for %s to be 0, but got %s", addr, ovmETHStateBalance)
			return innerErr
		}

		// Migrated state balance should equal the OVM ETH balance.
		m.Lock()
		defer m.Unlock()
		account, err := state.GetAccount(tx, addr)
		if err != nil {
			innerErr = fmt.Errorf("failed to get account for %s: %w", addr, err)
			return innerErr
		}
		_, balance, err := decodeNonceBalanceFromStorage(account)
		if err != nil {
			innerErr = fmt.Errorf("failed to decode nonce and balance for %s: %w", addr, err)
			return innerErr
		}
		migratedStateBalance := new(big.Int).SetBytes(balance)
		if migratedStateBalance.Cmp(ovmETHBalance) != 0 {
			innerErr = fmt.Errorf("expected OVM_ETH post-migration state balance for %s to be %s, but got %s", addr, ovmETHStateBalance, migratedStateBalance)
			return innerErr
		}
		// Migrated OVM ETH balance should be zero, since we wipe the slots.
		migratedBalance, err := state.GetStorage(tx, predeploys.LegacyERC20ETHAddr, key)
		if err != nil {
			innerErr = fmt.Errorf("failed to get OVM_ETH post-migration ERC20 balance for %s: %w", addr, err)
			return innerErr
		}
		if migratedBalance.Big().Cmp(libcommon.Big0) != 0 {
			innerErr = fmt.Errorf("expected OVM_ETH post-migration ERC20 balance for %s to be 0, but got %s", addr, migratedBalance)
			return innerErr
		}

		progress()
		count++

		return nil
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
func PostCheckL1Block(tx kv.Tx, info *L1BlockInfo) error {
	// Slot 0 is the concatenation of the block number and timestamp
	hash, err := state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	data := hash.Bytes()
	// data := db.GetState(predeploys.L1BlockAddr, common.Hash{}).Bytes()
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
	hash, err = state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x01})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	baseFee := hash.Big()
	if baseFee.Cmp(info.BaseFee) != 0 {
		return fmt.Errorf("expected L1Block basefee to be %s, but got %s", info.BaseFee, baseFee)
	}
	log.Debug("validated L1Block basefee", "expected", info.BaseFee)

	// Slot 2 is the block hash
	hash, err = state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x02})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	if *hash != info.BlockHash {
		return fmt.Errorf("expected L1Block hash to be %s, but got %s", info.BlockHash, hash)
	}
	log.Debug("validated L1Block hash", "expected", info.BlockHash)

	// Slot 3 is the sequence number. It is expected to be zero.
	sequenceNumber, err := state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x03})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	expSequenceNumber := libcommon.Hash{}
	if expSequenceNumber != *sequenceNumber {
		return fmt.Errorf("expected L1Block sequence number to be %s, but got %s", expSequenceNumber, sequenceNumber)
	}
	log.Debug("validated L1Block sequence number", "expected", expSequenceNumber)

	// Slot 4 is the versioned hash to authenticate the batcher. It is expected to be the initial batch sender.
	batcherHash, err := state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x04})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	batchSender := libcommon.BytesToAddress(batcherHash.Bytes())
	if batchSender != info.BatcherAddr {
		return fmt.Errorf("expected L1Block batcherHash to be %s, but got %s", info.BatcherAddr, batchSender)
	}
	log.Debug("validated L1Block batcherHash", "expected", info.BatcherAddr)

	// Slot 5 is the L1 fee overhead.
	l1FeeOverhead, err := state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x05})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	if !bytes.Equal(l1FeeOverhead.Bytes(), info.L1FeeOverhead[:]) {
		return fmt.Errorf("expected L1Block L1FeeOverhead to be %s, but got %s", info.L1FeeOverhead, l1FeeOverhead)
	}
	log.Debug("validated L1Block L1FeeOverhead", "expected", info.L1FeeOverhead)

	// Slot 6 is the L1 fee scalar.
	l1FeeScalar, err := state.GetStorage(tx, predeploys.L1BlockAddr, libcommon.Hash{31: 0x06})
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	if !bytes.Equal(l1FeeScalar.Bytes(), info.L1FeeScalar[:]) {
		return fmt.Errorf("expected L1Block L1FeeScalar to be %s, but got %s", info.L1FeeScalar, l1FeeScalar)
	}
	log.Debug("validated L1Block L1FeeScalar", "expected", info.L1FeeScalar)

	// Check EIP-1967
	admin, err := state.GetStorage(tx, predeploys.L1BlockAddr, AdminSlot)
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	proxyAdmin := libcommon.BytesToAddress(admin.Bytes())
	if proxyAdmin != predeploys.ProxyAdminAddr {
		return fmt.Errorf("expected L1Block admin to be %s, but got %s", predeploys.ProxyAdminAddr, proxyAdmin)
	}
	log.Debug("validated L1Block admin", "expected", predeploys.ProxyAdminAddr)
	expImplementation, err := AddressToCodeNamespace(predeploys.L1BlockAddr)
	if err != nil {
		return fmt.Errorf("failed to get expected implementation for L1Block: %w", err)
	}
	impl, err := state.GetStorage(tx, predeploys.L1BlockAddr, ImplementationSlot)
	if err != nil {
		return fmt.Errorf("failed to read L1Block storage: %w", err)
	}
	actImplementation := libcommon.BytesToAddress(impl.Bytes())
	if expImplementation != actImplementation {
		return fmt.Errorf("expected L1Block implementation to be %s, but got %s", expImplementation, actImplementation)
	}
	log.Debug("validated L1Block implementation", "expected", expImplementation)

	var count int
	cursor, err := tx.Cursor(kv.PlainState)
	if err != nil {
		return fmt.Errorf("failed to create cursor: %w", err)
	}
	defer cursor.Close()
	for k, _, err := cursor.First(); k != nil; k, _, err = cursor.Next() {
		if err != nil {
			return fmt.Errorf("failed to read storage from database: %w", err)
		}
		// Storage is 20 bytes account address + 8 byte incarnation + 32 byte storage key
		if len(k) == 60 && libcommon.BytesToAddress(k[:20]) == predeploys.L1BlockAddr {
			count++
		}
	}
	if count != 8 {
		return fmt.Errorf("expected L1Block to have 8 storage slots, but got %d", count)
	}
	log.Debug("validated L1Block storage slot count", "expected", 8)

	return nil
}

func CheckWithdrawalsAfter(tx kv.Tx, data crossdomain.MigrationData, l1CrossDomainMessenger *libcommon.Address) error {
	wds, invalidMessages, err := data.ToWithdrawals()
	if err != nil {
		return err
	}

	// First, make a mapping between old withdrawal slots and new ones.
	// This list can be a superset of what was actually migrated, since
	// some witness data may references withdrawals that reverted.
	oldToNewSlots := make(map[libcommon.Hash]libcommon.Hash)
	wdsByOldSlot := make(map[libcommon.Hash]*crossdomain.LegacyWithdrawal)
	invalidMessagesByOldSlot := make(map[libcommon.Hash]crossdomain.InvalidMessage)
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
	err = state.ForEachStorage(tx, predeploys.LegacyMessagePasserAddr, func(key, value libcommon.Hash) bool {
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
			value, err := state.GetStorage(tx, predeploys.L2ToL1MessagePasserAddr, key)
			if err != nil {
				innerErr = fmt.Errorf("failed to read L2ToL1MessagePasser storage: %w", err)
				return false
			}
			if *value != abiFalse {
				innerErr = fmt.Errorf("expected invalid slot not to be migrated, but got %s", value)
				return false
			}
			return true
		}

		// Grab the migrated slot.
		migratedSlot := oldToNewSlots[key]
		if migratedSlot == (libcommon.Hash{}) {
			innerErr = fmt.Errorf("no migrated slot found for legacy slot %s", key)
			return false
		}

		// Look up the migrated slot in the DB.
		migratedValue, err := state.GetStorage(tx, predeploys.L2ToL1MessagePasserAddr, migratedSlot)
		if err != nil {
			innerErr = fmt.Errorf("failed to read L2ToL1MessagePasser storage: %w", err)
			return false
		}

		// If the sender is _not_ the L2XDM, the value should not be migrated.
		wd := wdsByOldSlot[key]
		if wd.MessageSender == predeploys.L2CrossDomainMessengerAddr {
			// Make sure the value is abiTrue if this withdrawal should be migrated.
			if *migratedValue != abiTrue {
				innerErr = fmt.Errorf("expected migrated value to be true, but got %s", migratedValue)
				return false
			}
		} else {
			// Otherwise, ensure that withdrawals from senders other than the L2XDM are _not_ migrated.
			if *migratedValue != abiFalse {
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

func CheckPreBedrockAllocation(tx kv.Tx) error {
	hash := rawdb.ReadHeadHeaderHash(tx)
	num := rawdb.ReadHeaderNumber(tx, hash)
	if num == nil {
		return fmt.Errorf("cannot find header number for %s", hash)
	}

	dumper := erigonstate.NewDumper(tx, *num-1, false)
	result := dumper.RawDump(false, false)
	if len(result.Accounts) != 0 {
		return fmt.Errorf("pre-bedrock allocation is not empty")
	}
	return nil
}

func eip1967Slots(address libcommon.Address) StorageCheckMap {
	codeAddr, err := AddressToCodeNamespace(address)
	if err != nil {
		panic(err)
	}
	return StorageCheckMap{
		AdminSlot:          predeploys.ProxyAdminAddr.Hash(),
		ImplementationSlot: codeAddr.Hash(),
	}
}

func decodeNonceBalanceFromStorage(enc []byte) ([]byte, []byte, error) {
	if len(enc) == 0 {
		return []byte{}, []byte{}, nil
	}

	var fieldSet = enc[0]
	var pos = 1

	var nonce []byte
	var balance []byte

	//looks for the position incarnation is at
	if fieldSet&1 > 0 {
		decodeLength := int(enc[pos])
		if len(enc) < pos+decodeLength+1 {
			return []byte{}, []byte{}, fmt.Errorf(
				"malformed CBOR for Account.Nonce: %s, Length %d",
				enc[pos+1:], decodeLength)
		}
		nonce = enc[pos+1 : pos+decodeLength+1]
		pos += decodeLength + 1
	}

	if fieldSet&2 > 0 {
		decodeLength := int(enc[pos])
		if len(enc) < pos+decodeLength+1 {
			return []byte{}, []byte{}, fmt.Errorf(
				"malformed CBOR for Account.Nonce: %s, Length %d",
				enc[pos+1:], decodeLength)
		}
		balance = enc[pos+1 : pos+decodeLength+1]
	}

	return nonce, balance, nil
}

func bytesToUint64(buf []byte) (x uint64) {
	for i, b := range buf {
		x = x<<8 + uint64(b)
		if i == 7 {
			return
		}
	}
	return
}
