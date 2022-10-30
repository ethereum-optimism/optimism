package genesis

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/crossdomain"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// MigrateDB will migrate an old l2geth database to the new bedrock style system
func MigrateDB(db vm.StateDB, config *DeployConfig, l1Block *types.Block, l2Addrs *L2Addresses, migrationData *MigrationData) error {
	if err := SetL2Proxies(db); err != nil {
		return err
	}

	storage, err := NewL2StorageConfig(config, l1Block, l2Addrs)
	if err != nil {
		return err
	}

	immutable, err := NewL2ImmutableConfig(config, l1Block, l2Addrs)
	if err != nil {
		return err
	}

	if err := SetImplementations(db, storage, immutable); err != nil {
		return err
	}

	// Convert all of the messages into legacy withdrawals
	messages := make([]*crossdomain.LegacyWithdrawal, 0)
	for _, msg := range migrationData.OvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			return err
		}
		messages = append(messages, wd)
	}
	for _, msg := range migrationData.EvmMessages {
		wd, err := msg.ToLegacyWithdrawal()
		if err != nil {
			return err
		}
		messages = append(messages, wd)
	}

	if err := crossdomain.MigrateWithdrawals(messages, db, &l2Addrs.L1CrossDomainMessengerProxy, &l2Addrs.L1StandardBridgeProxy); err != nil {
		return err
	}

	// TODO: use migration data to double check things

	return nil
}
