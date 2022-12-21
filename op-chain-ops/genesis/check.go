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
const MaxSlotChecks = 5000

var LegacyETHCheckSlots = map[common.Hash]common.Hash{
	// Bridge
	common.Hash{31: 0x06}: common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000010"),
	// Symbol
	common.Hash{31: 0x04}: common.HexToHash("0x4554480000000000000000000000000000000000000000000000000000000006"),
	// Name
	common.Hash{31: 0x03}: common.HexToHash("0x457468657200000000000000000000000000000000000000000000000000000a"),
	// Total supply
	common.Hash{31: 0x02}: {},
}

// PostCheckMigratedDB will check that the migration was performed correctly
func PostCheckMigratedDB(ldb ethdb.Database, migrationData migration.MigrationData, l1XDM *common.Address, l1ChainID int) error {
	log.Info("Validating database migration")

	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return fmt.Errorf("cannot find header number for %s", hash)
	}

	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "number", *num)

	if !bytes.Equal(header.Extra, bedrockTransitionBlockExtraData) {
		return fmt.Errorf("expected extra data to be %x, but got %x", bedrockTransitionBlockExtraData, header.Extra)
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

	if err := PostCheckUntouchables(underlyingDB, db, prevHeader.Root); err != nil {
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
func PostCheckUntouchables(udb state.Database, currDB *state.StateDB, prevRoot common.Hash) error {
	prevDB, err := state.New(prevRoot, udb, nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	for addr := range UntouchablePredeploys {
		// Check that the code is the same.
		code := currDB.GetCode(addr)
		hash := crypto.Keccak256Hash(code)
		if hash != UntouchableCodeHashes[addr] {
			return fmt.Errorf("expected code hash for %s to be %s, but got %s", addr, UntouchableCodeHashes[addr], hash)
		}
		log.Info("checked code hash", "address", addr, "hash", hash)

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
	for _, wd := range wds {
		legacySlot, err := wd.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute legacy storage slot: %w", err)
		}

		legacyValue := db.GetState(predeploys.LegacyMessagePasserAddr, legacySlot)
		if legacyValue != abiTrue {
			return fmt.Errorf("legacy value is not ABI true: %s", legacyValue)
		}

		withdrawal, err := crossdomain.MigrateWithdrawal(wd, l1CrossDomainMessenger)
		if err != nil {
			return err
		}

		migratedSlot, err := withdrawal.StorageSlot()
		if err != nil {
			return fmt.Errorf("cannot compute withdrawal storage slot: %w", err)
		}

		value := db.GetState(predeploys.L2ToL1MessagePasserAddr, migratedSlot)
		if value != abiTrue {
			return fmt.Errorf("withdrawal %s not set to ABI true", withdrawal.Nonce)
		}
	}
	return nil
}
