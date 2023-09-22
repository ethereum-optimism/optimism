package upgrades

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/safe"

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

// L1 will add calls for upgrading each of the L1 contracts.
func L1(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	if err := L1CrossDomainMessenger(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := L1ERC721Bridge(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := L1StandardBridge(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := L2OutputOracle(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := OptimismMintableERC20Factory(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := OptimismPortal(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	if err := SystemConfig(batch, implementations, list, config, chainConfig); err != nil {
		return err
	}
	return nil
}

// L1CrossDomainMessenger will add a call to the batch that upgrades the L1CrossDomainMessenger.
func L1CrossDomainMessenger(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := l1CrossDomainMessengerABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	calldata, err := initialize.Inputs.PackValues([]any{
		common.HexToAddress(list.OptimismPortalProxy.String()),
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
		common.HexToAddress(implementations.L1CrossDomainMessenger.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L1ERC721Bridge will add a call to the batch that upgrades the L1ERC721Bridge.
func L1ERC721Bridge(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1ERC721BridgeABI, err := bindings.L1ERC721BridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := l1ERC721BridgeABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	calldata, err := initialize.Inputs.PackValues([]any{
		common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1ERC721BridgeProxy.String()),
		common.HexToAddress(implementations.L1ERC721Bridge.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L1StandardBridge will add a call to the batch that upgrades the L1StandardBridge.
func L1StandardBridge(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := l1StandardBridgeABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	calldata, err := initialize.Inputs.PackValues([]any{
		common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1StandardBridgeProxy.String()),
		common.HexToAddress(implementations.L1StandardBridge.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L2OutputOracle will add a call to the batch that upgrades the L2OutputOracle.
func L2OutputOracle(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l2OutputOracleABI, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := l2OutputOracleABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	l2OutputOracleStartingBlockNumber := new(big.Int).SetUint64(config.L2OutputOracleStartingBlockNumber)
	if config.L2OutputOracleStartingTimestamp < 0 {
		return fmt.Errorf("L2OutputOracleStartingBlockNumber must be concrete")
	}
	l2OutputOraclesStartingTimestamp := new(big.Int).SetInt64(int64(config.L2OutputOracleStartingTimestamp))

	calldata, err := initialize.Inputs.PackValues([]any{
		l2OutputOracleStartingBlockNumber,
		l2OutputOraclesStartingTimestamp,
		config.L2OutputOracleProposer,
		config.L2OutputOracleChallenger,
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L2OutputOracleProxy.String()),
		common.HexToAddress(implementations.L2OutputOracle.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// OptimismMintableERC20Factory will add a call to the batch that upgrades the OptimismMintableERC20Factory.
func OptimismMintableERC20Factory(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	optimismMintableERC20FactoryABI, err := bindings.OptimismMintableERC20FactoryMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := optimismMintableERC20FactoryABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	calldata, err := initialize.Inputs.PackValues([]any{
		common.HexToAddress(list.L1StandardBridgeProxy.String()),
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
		common.HexToAddress(implementations.OptimismMintableERC20Factory.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// OptimismPortal will add a call to the batch that upgrades the OptimismPortal.
func OptimismPortal(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	optimismPortalABI, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := optimismPortalABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	calldata, err := initialize.Inputs.PackValues([]any{
		common.HexToAddress(list.L2OutputOracleProxy.String()),
		config.PortalGuardian,
		common.HexToAddress(chainConfig.SystemConfigAddr.String()),
		false,
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.OptimismPortalProxy.String()),
		common.HexToAddress(implementations.OptimismPortal.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// SystemConfig will add a call to the batch that upgrades the SystemConfig.
func SystemConfig(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	systemConfigABI, err := bindings.SystemConfigMetaData.GetAbi()
	if err != nil {
		return err
	}

	initialize, ok := systemConfigABI.Methods["initialize"]
	if !ok {
		return fmt.Errorf("no initialize method")
	}

	gasPriceOracleOverhead := new(big.Int).SetUint64(config.GasPriceOracleOverhead)
	gasPriceOracleScalar := new(big.Int).SetUint64(config.GasPriceOracleScalar)
	batcherHash := common.BytesToHash(config.BatchSenderAddress.Bytes())
	l2GenesisBlockGasLimit := uint64(config.L2GenesisBlockGasLimit)
	startBlock := new(big.Int).SetUint64(config.SystemConfigStartBlock)

	addresses := bindings.SystemConfigAddresses{
		L1CrossDomainMessenger:       common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
		L1ERC721Bridge:               common.HexToAddress(list.L1ERC721BridgeProxy.String()),
		L1StandardBridge:             common.HexToAddress(list.L1StandardBridgeProxy.String()),
		L2OutputOracle:               common.HexToAddress(list.L2OutputOracleProxy.String()),
		OptimismPortal:               common.HexToAddress(list.OptimismPortalProxy.String()),
		OptimismMintableERC20Factory: common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
	}

	// This is more complex
	calldata, err := initialize.Inputs.PackValues([]any{
		config.FinalSystemOwner,
		gasPriceOracleOverhead,
		gasPriceOracleScalar,
		batcherHash,
		l2GenesisBlockGasLimit,
		config.P2PSequencerAddress,
		genesis.DefaultResourceConfig,
		startBlock,
		config.BatchInboxAddress,
		addresses,
	})
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(chainConfig.SystemConfigAddr.String()),
		common.HexToAddress(implementations.SystemConfig.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	sig := "upgradeAndCall(address,address,bytes)"
	if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, *proxyAdminABI); err != nil {
		return err
	}

	return nil
}
