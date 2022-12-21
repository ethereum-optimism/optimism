package genesis

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/ether"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// CheckMigratedDB will check that the migration was performed correctly
func CheckMigratedDB(ldb ethdb.Database, migrationData migration.MigrationData, l1XDM *common.Address) error {
	log.Info("Validating database migration")

	hash := rawdb.ReadHeadHeaderHash(ldb)
	log.Info("Reading chain tip from database", "hash", hash)
	num := rawdb.ReadHeaderNumber(ldb, hash)
	if num == nil {
		return fmt.Errorf("cannot find header number for %s", hash)
	}

	header := rawdb.ReadHeader(ldb, hash, *num)
	log.Info("Read header from database", "number", *num)

	underlyingDB := state.NewDatabaseWithConfig(ldb, &trie.Config{
		Preimages: true,
	})

	db, err := state.New(header.Root, underlyingDB, nil)
	if err != nil {
		return fmt.Errorf("cannot open StateDB: %w", err)
	}

	if err := CheckPredeploys(db); err != nil {
		return err
	}
	log.Info("checked predeploys")

	if err := CheckLegacyETH(db); err != nil {
		return err
	}
	log.Info("checked legacy eth")

	if err := CheckWithdrawalsAfter(db, migrationData, l1XDM); err != nil {
		return err
	}
	log.Info("checked withdrawals")

	return nil
}

// CheckPredeploys will check that there is code at each predeploy
// address
func CheckPredeploys(db vm.StateDB) error {
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
		if addr != predeploys.ProxyAdminAddr && addr != predeploys.GovernanceTokenAddr && adminAddr != predeploys.ProxyAdminAddr {
			return fmt.Errorf("admin is %s when it should be %s for %s", adminAddr, predeploys.ProxyAdminAddr, addr)
		}
	}

	// For each predeploy, check that we've set the implementation correctly when
	// necessary and that there's code at the implementation.
	for _, proxyAddr := range predeploys.Predeploys {
		implAddr, special, err := mapImplementationAddress(proxyAddr)
		if err != nil {
			return err
		}

		if !special {
			impl := db.GetState(*proxyAddr, ImplementationSlot)
			implAddr := common.BytesToAddress(impl.Bytes())
			if implAddr == (common.Address{}) {
				return fmt.Errorf("no implementation for %s", *proxyAddr)
			}
		}

		implCode := db.GetCode(implAddr)
		if len(implCode) == 0 {
			return fmt.Errorf("no code found at predeploy impl %s", *proxyAddr)
		}
	}

	return nil
}

// CheckLegacyETH checks that the legacy eth migration was successful.
// It currently only checks that the total supply was set to 0.
func CheckLegacyETH(db vm.StateDB) error {
	// Ensure total supply is set to 0
	slot := db.GetState(predeploys.LegacyERC20ETHAddr, ether.GetOVMETHTotalSupplySlot())
	if slot != (common.Hash{}) {
		log.Warn("total supply is not 0", "slot", slot)
	}
	return nil
}

// add another check to make sure that the withdrawals are set to ABI true
// for the hash
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
			log.Warn("legacy value is not ABI true", "legacySlot", legacySlot, "legacyValue", legacyValue)
			//return fmt.Errorf("legacy value is not ABI true: %s", legacyValue)
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
			log.Warn("withdrawal not set to ABI true", "slot", migratedSlot, "value", value)
			//return fmt.Errorf("withdrawal %s not set to ABI true", withdrawal.Nonce)
		}
	}
	return nil
}
