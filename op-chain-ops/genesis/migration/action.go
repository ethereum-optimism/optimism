package migration

import (
	"context"
	"github.com/ethereum-optimism/optimism/op-bindings/hardhat"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"path/filepath"
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

func Migrate(cfg *Config) error {
	deployConfig := cfg.DeployConfig

	ovmAddresses, err := NewAddresses(cfg.OVMAddressesPath)
	if err != nil {
		return err
	}
	evmAddresess, err := NewAddresses(cfg.EVMMessagesPath)
	if err != nil {
		return err
	}
	ovmAllowances, err := NewAllowances(cfg.OVMAllowancesPath)
	if err != nil {
		return err
	}
	ovmMessages, err := NewSentMessage(cfg.OVMMessagesPath)
	if err != nil {
		return err
	}
	evmMessages, err := NewSentMessage(cfg.EVMAddressesPath)
	if err != nil {
		return err
	}

	migrationData := MigrationData{
		OvmAddresses:  ovmAddresses,
		EvmAddresses:  evmAddresess,
		OvmAllowances: ovmAllowances,
		OvmMessages:   ovmMessages,
		EvmMessages:   evmMessages,
	}
	hh, err := hardhat.New(cfg.Network, []string{}, cfg.HardhatDeployments)
	if err != nil {
		return err
	}

	l1Client, err := ethclient.Dial(cfg.L1URL)
	if err != nil {
		return err
	}
	var blockNumber *big.Int
	bnum := cfg.StartingL1BlockNumber
	if bnum != 0 {
		blockNumber = new(big.Int).SetUint64(bnum)
	}

	block, err := l1Client.BlockByNumber(context.Background(), blockNumber)
	if err != nil {
		return err
	}

	chaindataPath := filepath.Join(cfg.L2DBPath, "geth", "chaindata")
	ancientPath := filepath.Join(cfg.L2DBPath, "ancient")
	ldb, err := rawdb.NewLevelDBDatabaseWithFreezer(chaindataPath, 1024, 60, ancientPath, "", true)
	if err != nil {
		return err
	}

	// Get the addresses from the hardhat deploy artifacts
	l1StandardBridgeProxyDeployment, err := hh.GetDeployment("Proxy__OVM_L1StandardBridge")
	if err != nil {
		return err
	}
	l1CrossDomainMessengerProxyDeployment, err := hh.GetDeployment("Proxy__OVM_L1CrossdomainMessenger")
	if err != nil {
		return err
	}
	l1ERC721BridgeProxyDeployment, err := hh.GetDeployment("L1ERC721BridgeProxy")
	if err != nil {
		return err
	}

	l2Addrs := genesis.L2Addresses{
		ProxyAdminOwner:             deployConfig.ProxyAdminOwner,
		L1StandardBridgeProxy:       l1StandardBridgeProxyDeployment.Address,
		L1CrossDomainMessengerProxy: l1CrossDomainMessengerProxyDeployment.Address,
		L1ERC721BridgeProxy:         l1ERC721BridgeProxyDeployment.Address,
	}

	if err := genesis.MigrateDB(ldb, deployConfig, block, &l2Addrs, &migrationData, !cfg.DryRun); err != nil {
		return err
	}

	return nil
}
