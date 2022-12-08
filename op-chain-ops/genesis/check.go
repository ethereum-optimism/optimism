package genesis

import (
	"errors"
	"fmt"
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
func CheckMigratedDB(ldb ethdb.Database) error {
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
		return errors.New("total supply not set to 0")
	}
	return nil
}
