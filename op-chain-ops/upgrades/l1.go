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

const (
	// upgradeAndCall represents the signature of the upgradeAndCall function
	// on the ProxyAdmin contract.
	upgradeAndCall = "upgradeAndCall(address,address,bytes)"
	// upgrade represents the signature of the upgrade function on the ProxyAdmin contract.
	upgrade = "upgrade(address,address)"

	method = "setBytes32"
)

var (
	// storageSetterAddr represents the address of the StorageSetter contract.
	storageSetterAddr = common.HexToAddress("0xd81f43eDBCAcb4c29a9bA38a13Ee5d79278270cC")

	// superchainConfigProxy refers to the address of the Sepolia superchain config proxy.
	// NOTE: this is currently hardcoded and we will need to move this to the superchain-registry
	// and have 1 deployed for each superchain target.
	superchainConfigProxy = common.HexToAddress("0xC2Be75506d5724086DEB7245bd260Cc9753911Be")
)

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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L11-L13
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L28
			{
				Key:   common.Hash{31: 249},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.L1CrossDomainMessengerProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	l1CrossDomainMessengerABI, err := bindings.L1CrossDomainMessengerMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1CrossDomainMessengerABI.Pack("initialize", superchainConfigProxy)
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L100-L102
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return fmt.Errorf("setBytes32: %w", err)
		}
		args := []any{
			common.HexToAddress(list.L1ERC721BridgeProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	l1ERC721BridgeABI, err := bindings.L1ERC721BridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1ERC721BridgeABI.Pack("initialize", superchainConfigProxy)
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L36-L37
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L41
			{
				Key:   common.Hash{31: 0x03},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.L1StandardBridgeProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	l1StandardBridgeABI, err := bindings.L1StandardBridgeMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := l1StandardBridgeABI.Pack("initialize", superchainConfigProxy)
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L50-L51
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L55
			{
				Key:   common.Hash{31: 0x04},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L56
			{
				Key:   common.Hash{31: 0x05},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.L2OutputOracleProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	l2OutputOracleABI, err := bindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		return err
	}

	var l2OutputOracleStartingBlockNumber, l2OutputOracleStartingTimestamp *big.Int
	if config != nil {
		l2OutputOracleStartingBlockNumber = new(big.Int).SetUint64(config.L2OutputOracleStartingBlockNumber)
		if config.L2OutputOracleStartingTimestamp < 0 {
			return fmt.Errorf("L2OutputOracleStartingTimestamp must be concrete")
		}
		l2OutputOracleStartingTimestamp = new(big.Int).SetInt64(int64(config.L2OutputOracleStartingTimestamp))
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
	}

	calldata, err := l2OutputOracleABI.Pack("initialize", l2OutputOracleStartingBlockNumber, l2OutputOracleStartingTimestamp)
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L287-L289
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	args := []any{
		common.HexToAddress(list.OptimismMintableERC20FactoryProxy.String()),
		common.HexToAddress(implementations.OptimismMintableERC20Factory.Address.String()),
	}

	proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
	if err := batch.AddCall(proxyAdmin, common.Big0, upgrade, args, proxyAdminABI); err != nil {
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L64-L65
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L72
			{
				Key:   common.Hash{31: 53},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L73
			{
				Key:   common.Hash{31: 54},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L74
			{
				Key:   common.Hash{31: 55},
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(list.OptimismPortalProxy.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	optimismPortalABI, err := bindings.OptimismPortalMetaData.GetAbi()
	if err != nil {
		return err
	}

	calldata, err := optimismPortalABI.Pack("initialize", superchainConfigProxy)
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

	// 2 Step Upgrade
	{
		storageSetterABI, err := bindings.StorageSetterMetaData.GetAbi()
		if err != nil {
			return err
		}

		input := []bindings.StorageSetterSlot{
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L82-L83
			{
				Key:   common.Hash{},
				Value: common.Hash{},
			},
			// https://github.com/ethereum-optimism/optimism/blob/86a96023ffd04d119296dff095d02fff79fa15de/packages/contracts-bedrock/.storage-layout#L92
			{
				Key:   common.Hash{31: 106},
				Value: common.Hash{},
			},
			// bytes32 public constant L1_CROSS_DOMAIN_MESSENGER_SLOT = bytes32(uint256(keccak256("systemconfig.l1crossdomainmessenger")) - 1);
			{
				Key:   common.HexToHash("0x383f291819e6d54073bc9a648251d97421076bdd101933c0c022219ce9580636"),
				Value: common.Hash{},
			},
			// bytes32 public constant L1_ERC_721_BRIDGE_SLOT = bytes32(uint256(keccak256("systemconfig.l1erc721bridge")) - 1);
			{
				Key:   common.HexToHash("0x46adcbebc6be8ce551740c29c47c8798210f23f7f4086c41752944352568d5a7"),
				Value: common.Hash{},
			},
			// bytes32 public constant L1_STANDARD_BRIDGE_SLOT = bytes32(uint256(keccak256("systemconfig.l1standardbridge")) - 1);
			{
				Key:   common.HexToHash("0x9904ba90dde5696cda05c9e0dab5cbaa0fea005ace4d11218a02ac668dad6376"),
				Value: common.Hash{},
			},
			// bytes32 public constant L2_OUTPUT_ORACLE_SLOT = bytes32(uint256(keccak256("systemconfig.l2outputoracle")) - 1);
			{
				Key:   common.HexToHash("0xe52a667f71ec761b9b381c7b76ca9b852adf7e8905da0e0ad49986a0a6871815"),
				Value: common.Hash{},
			},
			// bytes32 public constant OPTIMISM_PORTAL_SLOT = bytes32(uint256(keccak256("systemconfig.optimismportal")) - 1);
			{
				Key:   common.HexToHash("0x4b6c74f9e688cb39801f2112c14a8c57232a3fc5202e1444126d4bce86eb19ac"),
				Value: common.Hash{},
			},
			// bytes32 public constant OPTIMISM_MINTABLE_ERC20_FACTORY_SLOT = bytes32(uint256(keccak256("systemconfig.optimismmintableerc20factory")) - 1);
			{
				Key:   common.HexToHash("0xa04c5bb938ca6fc46d95553abf0a76345ce3e722a30bf4f74928b8e7d852320c"),
				Value: common.Hash{},
			},
			// bytes32 public constant BATCH_INBOX_SLOT = bytes32(uint256(keccak256("systemconfig.batchinbox")) - 1);
			{
				Key:   common.HexToHash("0x71ac12829d66ee73d8d95bff50b3589745ce57edae70a3fb111a2342464dc597"),
				Value: common.Hash{},
			},
		}

		calldata, err := storageSetterABI.Pack(method, input)
		if err != nil {
			return err
		}
		args := []any{
			common.HexToAddress(chainConfig.SystemConfigAddr.String()),
			storageSetterAddr,
			calldata,
		}
		proxyAdmin := common.HexToAddress(list.ProxyAdmin.String())
		if err := batch.AddCall(proxyAdmin, common.Big0, upgradeAndCall, args, proxyAdminABI); err != nil {
			return err
		}
	}

	systemConfigABI, err := bindings.SystemConfigMetaData.GetAbi()
	if err != nil {
		return err
	}

	var gasPriceOracleOverhead, gasPriceOracleScalar *big.Int
	var batcherHash common.Hash
	var p2pSequencerAddress, finalSystemOwner common.Address
	var l2GenesisBlockGasLimit uint64

	if config != nil {
		gasPriceOracleOverhead = new(big.Int).SetUint64(config.GasPriceOracleOverhead)
		gasPriceOracleScalar = new(big.Int).SetUint64(config.GasPriceOracleScalar)
		batcherHash = common.BytesToHash(config.BatchSenderAddress.Bytes())
		l2GenesisBlockGasLimit = uint64(config.L2GenesisBlockGasLimit)
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
		p2pSequencerAddress, err = systemConfig.UnsafeBlockSigner(&bind.CallOpts{})
		if err != nil {
			return err
		}
		finalSystemOwner, err = systemConfig.Owner(&bind.CallOpts{})
		if err != nil {
			return err
		}
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
