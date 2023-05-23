package genesis

import (
	"fmt"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ledgerwatch/erigon-lib/kv"
	"github.com/ledgerwatch/erigon/boba-chain-ops/crossdomain"
	"github.com/ledgerwatch/erigon/boba-chain-ops/ether"
	"github.com/ledgerwatch/erigon/core/types"
)

func RolloverDB(chaindb kv.RwDB, genesis *types.Genesis, migrationData *crossdomain.MigrationData, commit, noCheck bool) error {
	// We migrate the balances held inside the LegacyERC20ETH contract into the state trie.
	// We also delete the balances from the LegacyERC20ETH contract. Unlike the steps above, this step
	// combines the check and mutation steps into one in order to reduce migration time.
	log.Info("Starting to migrate ERC20 ETH")
	err := ether.MigrateBalances(genesis, migrationData.Addresses(), migrationData.OvmAllowances, noCheck)
	if err != nil {
		return fmt.Errorf("failed to migrate OVM_ETH: %w", err)
	}

	if !commit {
		log.Info("Dry run complete!")
		return nil
	}

	if err = WriteGenesis(chaindb, genesis); err != nil {
		return err
	}

	return nil
}
