package migration_action

import (
	"context"
	"math/big"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
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
	hh, err := hardhat.New(cfg.Network, []string{}, cfg.HardhatDeployments)
	if err != nil {
		return nil, err
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
	ldb, err := rawdb.NewLevelDBDatabaseWithFreezer(chaindataPath, 4096, 120, ancientPath, "", false)
	if err != nil {
		return nil, err
	}
	defer ldb.Close()

	// Get the addresses from the hardhat deploy artifacts
	l1StandardBridgeProxyDeployment, err := hh.GetDeployment("Proxy__OVM_L1StandardBridge")
	if err != nil {
		return nil, err
	}
	l1CrossDomainMessengerProxyDeployment, err := hh.GetDeployment("Proxy__OVM_L1CrossDomainMessenger")
	if err != nil {
		return nil, err
	}
	l1ERC721BridgeProxyDeployment, err := hh.GetDeployment("L1ERC721BridgeProxy")
	if err != nil {
		return nil, err
	}

	l2Addrs := genesis.L2Addresses{
		ProxyAdminOwner:             deployConfig.ProxyAdminOwner,
		L1StandardBridgeProxy:       l1StandardBridgeProxyDeployment.Address,
		L1CrossDomainMessengerProxy: l1CrossDomainMessengerProxyDeployment.Address,
		L1ERC721BridgeProxy:         l1ERC721BridgeProxyDeployment.Address,
	}
	return genesis.MigrateDB(ldb, deployConfig, block, &l2Addrs, &migrationData, !cfg.DryRun)
}
