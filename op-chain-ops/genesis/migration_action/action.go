package migration_action

import (
	"context"
	"math/big"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis/migration"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Config struct {
	DeployConfig          *genesis.DeployConfig
	OVMAddressesPath      string
	EVMAddressesPath      string
	OVMAllowancesPath     string
	OVMMessagesPath       string
	EVMMessagesPath       string
	Network               string
	HardhatDeployments    []string
	L1URL                 string
	StartingL1BlockNumber uint64
	L2DBPath              string
	DryRun                bool
	NoCheck               bool
}

func Migrate(cfg *Config) (*genesis.MigrationResult, error) {
	deployConfig := cfg.DeployConfig

	ovmAddresses, err := migration.NewAddresses(cfg.OVMAddressesPath)
	if err != nil {
		return nil, err
	}
	evmAddresess, err := migration.NewAddresses(cfg.EVMAddressesPath)
	if err != nil {
		return nil, err
	}
	ovmAllowances, err := migration.NewAllowances(cfg.OVMAllowancesPath)
	if err != nil {
		return nil, err
	}
	ovmMessages, err := migration.NewSentMessage(cfg.OVMMessagesPath)
	if err != nil {
		return nil, err
	}
	evmMessages, err := migration.NewSentMessage(cfg.EVMMessagesPath)
	if err != nil {
		return nil, err
	}

	migrationData := migration.MigrationData{
		OvmAddresses:  ovmAddresses,
		EvmAddresses:  evmAddresess,
		OvmAllowances: ovmAllowances,
		OvmMessages:   ovmMessages,
		EvmMessages:   evmMessages,
	}

	l1Client, err := ethclient.Dial(cfg.L1URL)
	if err != nil {
		return nil, err
	}
	var blockNumber *big.Int
	bnum := cfg.StartingL1BlockNumber
	if bnum != 0 {
		blockNumber = new(big.Int).SetUint64(bnum)
	}

	block, err := l1Client.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return nil, err
	}

	chaindataPath := filepath.Join(cfg.L2DBPath, "geth", "chaindata")
	ancientPath := filepath.Join(chaindataPath, "ancient")
	ldb, err := rawdb.Open(
		rawdb.OpenOptions{
			Type:              "leveldb",
			Directory:         chaindataPath,
			Cache:             4096,
			Handles:           120,
			AncientsDirectory: ancientPath,
			Namespace:         "",
			ReadOnly:          false,
		})
	if err != nil {
		return nil, err
	}
	defer ldb.Close()

	return genesis.MigrateDB(ldb, deployConfig, block, &migrationData, !cfg.DryRun, cfg.NoCheck)
}
