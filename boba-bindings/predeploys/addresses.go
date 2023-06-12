package predeploys

import (
	"github.com/ledgerwatch/erigon-lib/common"
)

const (
	L2ToL1MessagePasser = "0x4200000000000000000000000000000000000016"
	DeployerWhitelist   = "0x4200000000000000000000000000000000000002"
	// We are different here
	LegacyERC20ETH                = "0x4200000000000000000000000000000000000006"
	WETH9                         = "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000"
	L2CrossDomainMessenger        = "0x4200000000000000000000000000000000000007"
	L2StandardBridge              = "0x4200000000000000000000000000000000000010"
	SequencerFeeVault             = "0x4200000000000000000000000000000000000011"
	OptimismMintableERC20Factory  = "0x4200000000000000000000000000000000000012"
	L1BlockNumber                 = "0x4200000000000000000000000000000000000013"
	GasPriceOracle                = "0x420000000000000000000000000000000000000F"
	L1Block                       = "0x4200000000000000000000000000000000000015"
	LegacyMessagePasser           = "0x4200000000000000000000000000000000000000"
	L2ERC721Bridge                = "0x4200000000000000000000000000000000000014"
	OptimismMintableERC721Factory = "0x4200000000000000000000000000000000000017"
	ProxyAdmin                    = "0x4200000000000000000000000000000000000018"
	BaseFeeVault                  = "0x4200000000000000000000000000000000000019"
	L1FeeVault                    = "0x420000000000000000000000000000000000001a"
	// BOBA Specific
	BobaTuringCredit   = "0x4200000000000000000000000000000000000020"
	BobaL2             = "0x4200000000000000000000000000000000000023"
	BobaGasPriceOracle = "0x4200000000000000000000000000000000000024"
	// These contracts need to be destroyed
	BobaTuringCreditImplementation  = "0x4200000000000000000000000000000000000021"
	BobaTuringHelperImplementation  = "0x4200000000000000000000000000000000000022"
	BobaGasPriceOracleImplmentation = "0x4200000000000000000000000000000000000025"
)

var (
	L2ToL1MessagePasserAddr           = common.HexToAddress(L2ToL1MessagePasser)
	DeployerWhitelistAddr             = common.HexToAddress(DeployerWhitelist)
	LegacyERC20ETHAddr                = common.HexToAddress(LegacyERC20ETH)
	WETH9Addr                         = common.HexToAddress(WETH9)
	L2CrossDomainMessengerAddr        = common.HexToAddress(L2CrossDomainMessenger)
	L2StandardBridgeAddr              = common.HexToAddress(L2StandardBridge)
	SequencerFeeVaultAddr             = common.HexToAddress(SequencerFeeVault)
	OptimismMintableERC20FactoryAddr  = common.HexToAddress(OptimismMintableERC20Factory)
	L1BlockNumberAddr                 = common.HexToAddress(L1BlockNumber)
	GasPriceOracleAddr                = common.HexToAddress(GasPriceOracle)
	L1BlockAddr                       = common.HexToAddress(L1Block)
	LegacyMessagePasserAddr           = common.HexToAddress(LegacyMessagePasser)
	L2ERC721BridgeAddr                = common.HexToAddress(L2ERC721Bridge)
	OptimismMintableERC721FactoryAddr = common.HexToAddress(OptimismMintableERC721Factory)
	ProxyAdminAddr                    = common.HexToAddress(ProxyAdmin)
	BaseFeeVaultAddr                  = common.HexToAddress(BaseFeeVault)
	L1FeeVaultAddr                    = common.HexToAddress(L1FeeVault)
	// BOBA Specific
	BobaTuringCreditAddr   = common.HexToAddress(BobaTuringCredit)
	BobaL2Addr             = common.HexToAddress(BobaL2)
	BobaGasPriceOracleAddr = common.HexToAddress(BobaGasPriceOracle)
	// Boba Legacy
	BobaTuringCreditImplementationAddr  = common.HexToAddress(BobaTuringCreditImplementation)
	BobaTuringHelperImplementationAddr  = common.HexToAddress(BobaTuringHelperImplementation)
	BobaGasPriceOracleImplmentationAddr = common.HexToAddress(BobaGasPriceOracleImplmentation)

	Predeploys                    = make(map[string]*common.Address)
	LegacyBobaProxyImplementation = make(map[string]*common.Address)
)

func init() {
	Predeploys["L2ToL1MessagePasser"] = &L2ToL1MessagePasserAddr
	Predeploys["DeployerWhitelist"] = &DeployerWhitelistAddr
	Predeploys["LegacyERC20ETH"] = &LegacyERC20ETHAddr
	Predeploys["WETH9"] = &WETH9Addr
	Predeploys["L2CrossDomainMessenger"] = &L2CrossDomainMessengerAddr
	Predeploys["L2StandardBridge"] = &L2StandardBridgeAddr
	Predeploys["SequencerFeeVault"] = &SequencerFeeVaultAddr
	Predeploys["OptimismMintableERC20Factory"] = &OptimismMintableERC20FactoryAddr
	Predeploys["L1BlockNumber"] = &L1BlockNumberAddr
	Predeploys["GasPriceOracle"] = &GasPriceOracleAddr
	Predeploys["L1Block"] = &L1BlockAddr
	Predeploys["LegacyMessagePasser"] = &LegacyMessagePasserAddr
	Predeploys["L2ERC721Bridge"] = &L2ERC721BridgeAddr
	Predeploys["OptimismMintableERC721Factory"] = &OptimismMintableERC721FactoryAddr
	Predeploys["ProxyAdmin"] = &ProxyAdminAddr
	Predeploys["BaseFeeVault"] = &BaseFeeVaultAddr
	Predeploys["L1FeeVault"] = &L1FeeVaultAddr
	// BOBA Specific
	Predeploys["BobaTuringCredit"] = &BobaTuringCreditAddr
	Predeploys["BobaL2"] = &BobaL2Addr
	Predeploys["BobaGasPriceOracle"] = &BobaGasPriceOracleAddr
	// Legacy
	LegacyBobaProxyImplementation["BobaTuringCreditImplementation"] = &BobaTuringCreditImplementationAddr
	LegacyBobaProxyImplementation["BobaTuringHelperImplementation"] = &BobaTuringHelperImplementationAddr
	LegacyBobaProxyImplementation["BobaGasPriceOracleImplmentation"] = &BobaGasPriceOracleImplmentationAddr
}
