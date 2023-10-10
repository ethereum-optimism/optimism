package upgrades

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/safe"

	"github.com/ethereum-optimism/superchain-registry/superchain"
)

// upgradeAndCall represents the signature of the upgradeAndCall function
// on the ProxyAdmin contract.
const upgradeAndCall = "upgradeAndCall(address,address,bytes)"

// L1 will add calls for upgrading each of the L1 contracts.
func L1(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	if err := L1CrossDomainMessenger(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading L1CrossDomainMessenger: %w", err)
	}
	if err := L1ERC721Bridge(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading L1ERC721Bridge: %w", err)
	}
	if err := L1StandardBridge(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading L1StandardBridge: %w", err)
	}
	if err := L2OutputOracle(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading L2OutputOracle: %w", err)
	}
	if err := OptimismMintableERC20Factory(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading OptimismMintableERC20Factory: %w", err)
	}
	if err := OptimismPortal(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading OptimismPortal: %w", err)
	}
	if err := SystemConfig(batch, implementations, list, config, chainConfig, backend); err != nil {
		return fmt.Errorf("upgrading SystemConfig: %w", err)
	}
	return nil
}

// L1CrossDomainMessenger will add a call to the batch that upgrades the L1CrossDomainMessenger.
func L1CrossDomainMessenger(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1CrossDomainMessengerABI.Pack("initialize", common.HexToAddress(list.OptimismPortalProxy.String()))
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
		common.HexToAddress(implementations.L1CrossDomainMessenger.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L1ERC721Bridge will add a call to the batch that upgrades the L1ERC721Bridge.
func L1ERC721Bridge(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l1ERC721BridgeABI, err := bindings.L1ERC721BridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1ERC721BridgeABI.Pack("initialize", common.HexToAddress(list.L1CrossDomainMessengerProxy.String()))
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1ERC721BridgeProxy.String()),
		common.HexToAddress(implementations.L1ERC721Bridge.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L1StandardBridge will add a call to the batch that upgrades the L1StandardBridge.
func L1StandardBridge(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	// Add in OP Mainnet specific upgrade logic here
	if chainConfig.ChainID == 10 {
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}
		calldata, err := storageSetterABI.Pack("setBytes32", common.Hash{}, common.Hash{})
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.L1StandardBridgeProxy.String()),
			common.HexToAddress("0xf30CE41cA2f24D28b95Eb861553dAc2948e0157F"),
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		sig := "upgradeAndCall(address,address,bytes)"
		if err := batch.AddCall(proxyAdmin, common.Big0, sig, args, proxyAdminABI); err != nil {
			return err
		}
	}

	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1StandardBridgeABI.Pack("initialize", common.HexToAddress(list.L1CrossDomainMessengerProxy.String()))
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L1StandardBridgeProxy.String()),
		common.HexToAddress(implementations.L1StandardBridge.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// L2OutputOracle will add a call to the batch that upgrades the L2OutputOracle.
func L2OutputOracle(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	l2OutputOracleABI, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return err
	}

	var l2OutputOracleStartingBlockNumber, l2OutputOracleStartingTimestamp *big.Int
	var l2OutputOracleProposer, l2OutputOracleChallenger common.Address
	if config != nil {
		l2OutputOracleStartingBlockNumber = new(big.Int).SetUint64(config.L2OutputOracleStartingBlockNumber)
		if config.L2OutputOracleStartingTimestamp < 0 {
			return fmt.Errorf("L2OutputOracleStartingTimestamp must be concrete")
		}
		l2OutputOracleStartingTimestamp = new(big.Int).SetInt64(int64(config.L2OutputOracleStartingTimestamp))
		l2OutputOracleProposer = config.L2OutputOracleProposer
		l2OutputOracleChallenger = config.L2OutputOracleChallenger
	} else {
		l2OutputOracle, err := bindings.NewL2OutputOracleCaller(common.HexToAddress(list.L2OutputOracleProxy.String()), backend)
		if err != nil {
			return err
		}
		l2OutputOracleStartingBlockNumber, err = l2OutputOracle.StartingBlockNumber(&bind.CallOpts{})
		if err != nil {
			return err
		}

		l2OutputOracleStartingTimestamp, err = l2OutputOracle.StartingTimestamp(&bind.CallOpts{})
		if err != nil {
			return err
		}

		l2OutputOracleProposer, err = l2OutputOracle.PROPOSER(&bind.CallOpts{})
		if err != nil {
			return err
		}

		l2OutputOracleChallenger, err = l2OutputOracle.CHALLENGER(&bind.CallOpts{})
		if err != nil {
			return err
		}
	}

	calldata, err := l2OutputOracleABI.Pack("initialize", l2OutputOracleStartingBlockNumber, l2OutputOracleStartingTimestamp, l2OutputOracleProposer, l2OutputOracleChallenger)
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.L2OutputOracleProxy.String()),
		common.HexToAddress(implementations.L2OutputOracle.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// OptimismMintableERC20Factory will add a call to the batch that upgrades the OptimismMintableERC20Factory.
func OptimismMintableERC20Factory(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	optimismMintableERC20FactoryABI, err := bindings.OptimismMintableERC20FactoryMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := optimismMintableERC20FactoryABI.Pack("initialize", common.HexToAddress(list.L1StandardBridgeProxy.String()))
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
		common.HexToAddress(implementations.OptimismMintableERC20Factory.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// OptimismPortal will add a call to the batch that upgrades the OptimismPortal.
func OptimismPortal(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	optimismPortalABI, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return err
	}

	var portalGuardian common.Address
	if config != nil {
		portalGuardian = config.PortalGuardian
	} else {
		optimismPortal, err := bindings.NewOptimismPortalCaller(common.HexToAddress(list.OptimismPortalProxy.String()), backend)
		if err != nil {
			return err
		}
		guardian, err := optimismPortal.GUARDIAN(&bind.CallOpts{})
		if err != nil {
			return err
		}
		portalGuardian = guardian
	}

	calldata, err := optimismPortalABI.Pack("initialize", common.HexToAddress(list.L2OutputOracleProxy.String()), portalGuardian, common.HexToAddress(chainConfig.SystemConfigAddr.String()), false)
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(list.OptimismPortalProxy.String()),
		common.HexToAddress(implementations.OptimismPortal.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}

// SystemConfig will add a call to the batch that upgrades the SystemConfig.
func SystemConfig(batch *safe.Batch, implementations superchain.ImplementationList, list superchain.AddressList, config *genesis.DeployConfig, chainConfig *superchain.ChainConfig, backend bind.ContractBackend) error {
	proxyAdminABI, err := bindings.ProxyAdminMetaData.GetAbi()
	if err != nil {
		return err
	}

	systemConfigABI, err := bindings.SystemConfigMetaData.GetAbi()
	if err != nil {
		return err
	}

	// If we want to be able to override these based on the values in the config,
	// the logic below will need to be updated. Right now the logic prefers the
	// on chain values over the offchain values. This to maintain backwards compatibility
	// in the short term.
	startBlock := big.NewInt(0)
	batchInboxAddress := common.HexToAddress(chainConfig.BatchInboxAddr.String())

	var gasPriceOracleOverhead, gasPriceOracleScalar *big.Int
	var batcherHash common.Hash
	var p2pSequencerAddress, finalSystemOwner common.Address
	var l2GenesisBlockGasLimit uint64

	if config != nil {
		gasPriceOracleOverhead = new(big.Int).SetUint64(config.GasPriceOracleOverhead)
		gasPriceOracleScalar = new(big.Int).SetUint64(config.GasPriceOracleScalar)
		batcherHash = common.BytesToHash(config.BatchSenderAddress.Bytes())
		l2GenesisBlockGasLimit = uint64(config.L2GenesisBlockGasLimit)
		startBlock = new(big.Int).SetUint64(config.SystemConfigStartBlock)
		batchInboxAddress = config.BatchInboxAddress
		p2pSequencerAddress = config.P2PSequencerAddress
		finalSystemOwner = config.FinalSystemOwner
	} else {
		systemConfig, err := bindings.NewSystemConfigCaller(common.HexToAddress(chainConfig.SystemConfigAddr.String()), backend)
		if err != nil {
			return err
		}
		gasPriceOracleOverhead, err = systemConfig.Overhead(&bind.CallOpts{})
		if err != nil {
			return err
		}
		gasPriceOracleScalar, err = systemConfig.Scalar(&bind.CallOpts{})
		if err != nil {
			return err
		}
		batcherHash, err = systemConfig.BatcherHash(&bind.CallOpts{})
		if err != nil {
			return err
		}
		l2GenesisBlockGasLimit, err = systemConfig.GasLimit(&bind.CallOpts{})
		if err != nil {
			return err
		}
		// StartBlock is a new property, we want to explicitly set it to 0 if there is an error fetching it
		systemConfigStartBlock, err := systemConfig.StartBlock(&bind.CallOpts{})
		if err != nil && systemConfigStartBlock != nil {
			startBlock = systemConfigStartBlock
		}
		// BatchInboxAddress is a new property, we want to set it to the offchain value if there is an error fetching it
		systemConfigBatchInboxAddress, err := systemConfig.BatchInbox(&bind.CallOpts{})
		if err != nil && systemConfigBatchInboxAddress != (common.Address{}) {
			batchInboxAddress = systemConfigBatchInboxAddress
		}
		p2pSequencerAddress, err = systemConfig.UnsafeBlockSigner(&bind.CallOpts{})
		if err != nil {
			return err
		}
		finalSystemOwner, err = systemConfig.Owner(&bind.CallOpts{})
		if err != nil {
			return err
		}
	}

	addresses := bindings.SystemConfigAddresses{
		L1CrossDomainMessenger:       common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
		L1ERC721Bridge:               common.HexToAddress(list.L1ERC721BridgeProxy.String()),
		L1StandardBridge:             common.HexToAddress(list.L1StandardBridgeProxy.String()),
		L2OutputOracle:               common.HexToAddress(list.L2OutputOracleProxy.String()),
		OptimismPortal:               common.HexToAddress(list.OptimismPortalProxy.String()),
		OptimismMintableERC20Factory: common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
	}

	calldata, err := systemConfigABI.Pack(
		"initialize",
		finalSystemOwner,
		gasPriceOracleOverhead,
		gasPriceOracleScalar,
		batcherHash,
		l2GenesisBlockGasLimit,
		p2pSequencerAddress,
		genesis.DefaultResourceConfig,
		startBlock,
		batchInboxAddress,
		addresses,
	)
	if err != nil {
		return err
	}

	args := []any{
		common.HexToAddress(chainConfig.SystemConfigAddr.String()),
		common.HexToAddress(implementations.SystemConfig.Address.String()),
		calldata,
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
		return err
	}

	return nil
}
