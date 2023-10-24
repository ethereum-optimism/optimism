package predeploys

import "github.com/ethereum/go-ethereum/common"

// TODO - we should get a single toml yaml or json file source of truth in @eth-optimism/bedrock package
// This needs to be kept in sync with @eth-optimism/contracts-ts/wagmi.config.ts which also specifies this
// To improve robustness and maintainability contracts-bedrock should export all addresses
const (
	L2ToL1MessagePasser           = "0x4200000000000000000000000000000000000016"
	DeployerWhitelist             = "0x4200000000000000000000000000000000000002"
	WETH9                         = "0x4200000000000000000000000000000000000006"
	L2CrossDomainMessenger        = "0x4200000000000000000000000000000000000007"
	L2StandardBridge              = "0x4200000000000000000000000000000000000010"
	SequencerFeeVault             = "0x4200000000000000000000000000000000000011"
	OptimismMintableERC20Factory  = "0x4200000000000000000000000000000000000012"
	L1BlockNumber                 = "0x4200000000000000000000000000000000000013"
	GasPriceOracle                = "0x420000000000000000000000000000000000000F"
	L1Block                       = "0x4200000000000000000000000000000000000015"
	GovernanceToken               = "0x4200000000000000000000000000000000000042"
	LegacyMessagePasser           = "0x4200000000000000000000000000000000000000"
	L2ERC721Bridge                = "0x4200000000000000000000000000000000000014"
	OptimismMintableERC721Factory = "0x4200000000000000000000000000000000000017"
	ProxyAdmin                    = "0x4200000000000000000000000000000000000018"
	BaseFeeVault                  = "0x4200000000000000000000000000000000000019"
	L1FeeVault                    = "0x420000000000000000000000000000000000001a"
	SchemaRegistry                = "0x4200000000000000000000000000000000000020"
	EAS                           = "0x4200000000000000000000000000000000000021"
	Safe_v130                     = "0x69f4D1788e39c87893C980c06EdF4b7f686e2938"
	SafeL2                        = "0xfb1bffC9d739B8D520DaF37dF666da4C687191EA"
	MultiSend                     = "0x998739BFdAAdde7C933B942a68053933098f9EDa"
	MultiSendCallOnly             = "0xA1dabEF33b3B82c7814B6D82A79e50F4AC44102B"
	Multicall3                    = "0xcA11bde05977b3631167028862bE2a173976CA11"
	Create2Deployer               = "0x13b0D85CcB8bf860b6b79AF3029fCA081AE9beF2"
	Permit2                       = "0x000000000022D473030F116dDEE9F6B43aC78BA3"
	EntryPoint                    = "0x5FF137D4b0FDCD49DcA30c7CF57E578a026d2789"
	SafeSingletonFactory          = "0x914d7Fec6aaC8cd542e72Bca78B30650d45643d7"
	DeterministicDeploymentProxy  = "0x4e59b44847b379578588920cA78FbF26c0B4956C"
)

var (
	L2ToL1MessagePasserAddr           = common.HexToAddress(L2ToL1MessagePasser)
	DeployerWhitelistAddr             = common.HexToAddress(DeployerWhitelist)
	WETH9Addr                         = common.HexToAddress(WETH9)
	L2CrossDomainMessengerAddr        = common.HexToAddress(L2CrossDomainMessenger)
	L2StandardBridgeAddr              = common.HexToAddress(L2StandardBridge)
	SequencerFeeVaultAddr             = common.HexToAddress(SequencerFeeVault)
	OptimismMintableERC20FactoryAddr  = common.HexToAddress(OptimismMintableERC20Factory)
	L1BlockNumberAddr                 = common.HexToAddress(L1BlockNumber)
	GasPriceOracleAddr                = common.HexToAddress(GasPriceOracle)
	L1BlockAddr                       = common.HexToAddress(L1Block)
	GovernanceTokenAddr               = common.HexToAddress(GovernanceToken)
	LegacyMessagePasserAddr           = common.HexToAddress(LegacyMessagePasser)
	L2ERC721BridgeAddr                = common.HexToAddress(L2ERC721Bridge)
	OptimismMintableERC721FactoryAddr = common.HexToAddress(OptimismMintableERC721Factory)
	ProxyAdminAddr                    = common.HexToAddress(ProxyAdmin)
	BaseFeeVaultAddr                  = common.HexToAddress(BaseFeeVault)
	L1FeeVaultAddr                    = common.HexToAddress(L1FeeVault)
	SchemaRegistryAddr                = common.HexToAddress(SchemaRegistry)
	EASAddr                           = common.HexToAddress(EAS)
	Safe_v130Addr                     = common.HexToAddress(Safe_v130)
	SafeL2Addr                        = common.HexToAddress(SafeL2)
	MultiSendAddr                     = common.HexToAddress(MultiSend)
	MultiSendCallOnlyAddr             = common.HexToAddress(MultiSendCallOnly)
	Multicall3Addr                    = common.HexToAddress(Multicall3)
	Create2DeployerAddr               = common.HexToAddress(Create2Deployer)
	Permit2Addr                       = common.HexToAddress(Permit2)
	EntryPointAddr                    = common.HexToAddress(EntryPoint)
	SafeSingletonFactoryAddr          = common.HexToAddress(SafeSingletonFactory)
	DeterministicDeploymentProxyAddr  = common.HexToAddress(DeterministicDeploymentProxy)

	Predeploys = make(map[string]*common.Address)
)

// IsProxied returns true for predeploys that will sit behind a proxy contract
func IsProxied(predeployAddr common.Address) bool {
	switch predeployAddr {
	case WETH9Addr:
	case GovernanceTokenAddr:
	case Safe_v130Addr:
	case SafeL2Addr:
	case MultiSendAddr:
	case MultiSendCallOnlyAddr:
	case Multicall3Addr:
	case Create2DeployerAddr:
	case Permit2Addr:
	case EntryPointAddr:
	case SafeSingletonFactoryAddr:
	case DeterministicDeploymentProxyAddr:
	default:
		return true
	}
	return false
}

func init() {
	Predeploys["L2ToL1MessagePasser"] = &L2ToL1MessagePasserAddr
	Predeploys["DeployerWhitelist"] = &DeployerWhitelistAddr
	Predeploys["WETH9"] = &WETH9Addr
	Predeploys["L2CrossDomainMessenger"] = &L2CrossDomainMessengerAddr
	Predeploys["L2StandardBridge"] = &L2StandardBridgeAddr
	Predeploys["SequencerFeeVault"] = &SequencerFeeVaultAddr
	Predeploys["OptimismMintableERC20Factory"] = &OptimismMintableERC20FactoryAddr
	Predeploys["L1BlockNumber"] = &L1BlockNumberAddr
	Predeploys["GasPriceOracle"] = &GasPriceOracleAddr
	Predeploys["L1Block"] = &L1BlockAddr
	Predeploys["GovernanceToken"] = &GovernanceTokenAddr
	Predeploys["LegacyMessagePasser"] = &LegacyMessagePasserAddr
	Predeploys["L2ERC721Bridge"] = &L2ERC721BridgeAddr
	Predeploys["OptimismMintableERC721Factory"] = &OptimismMintableERC721FactoryAddr
	Predeploys["ProxyAdmin"] = &ProxyAdminAddr
	Predeploys["BaseFeeVault"] = &BaseFeeVaultAddr
	Predeploys["L1FeeVault"] = &L1FeeVaultAddr
	Predeploys["SchemaRegistry"] = &SchemaRegistryAddr
	Predeploys["EAS"] = &EASAddr

	Predeploys["Safe_v130"] = &Safe_v130Addr
	Predeploys["SafeL2"] = &SafeL2Addr
	Predeploys["MultiSend"] = &MultiSendAddr
	Predeploys["MultiSendCallOnly"] = &MultiSendCallOnlyAddr
	Predeploys["Multicall3"] = &Multicall3Addr
	Predeploys["Create2Deployer"] = &Create2DeployerAddr
	Predeploys["Permit2"] = &Permit2Addr
	Predeploys["EntryPoint"] = &EntryPointAddr
	Predeploys["SafeSingletonFactory"] = &SafeSingletonFactoryAddr
	Predeploys["DeterministicDeploymentProxy"] = &DeterministicDeploymentProxyAddr
}
