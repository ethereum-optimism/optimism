package predeploys

import "github.com/ethereum/go-ethereum/common"

const (
	L2ToL1MessagePasser          = "0x4200000000000000000000000000000000000000"
	OVM_DeployerWhitelist        = "0x4200000000000000000000000000000000000002"
	OVM_ETH                      = "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000"
	WETH9                        = "0x4200000000000000000000000000000000000006"
	L2CrossDomainMessenger       = "0x4200000000000000000000000000000000000007"
	L2StandardBridge             = "0x4200000000000000000000000000000000000010"
	SequencerFeeVault            = "0x4200000000000000000000000000000000000011"
	OptimismMintableTokenFactory = "0x4200000000000000000000000000000000000012"
	L1BlockNumber                = "0x4200000000000000000000000000000000000013"
	OVM_GasPriceOracle           = "0x420000000000000000000000000000000000000F"
	L1Block                      = "0x4200000000000000000000000000000000000015"
	GovernanceToken              = "0x4200000000000000000000000000000000000042"
)

var (
	L2ToL1MessagePasserAddr          = common.HexToAddress(L2ToL1MessagePasser)
	OVM_DeployerWhitelistAddr        = common.HexToAddress(OVM_DeployerWhitelist)
	OVM_ETHAddr                      = common.HexToAddress(OVM_ETH)
	WETH9Addr                        = common.HexToAddress(WETH9)
	L2CrossDomainMessengerAddr       = common.HexToAddress(L2CrossDomainMessenger)
	L2StandardBridgeAddr             = common.HexToAddress(L2StandardBridge)
	SequencerFeeVaultAddr            = common.HexToAddress(SequencerFeeVault)
	OptimismMintableTokenFactoryAddr = common.HexToAddress(OptimismMintableTokenFactory)
	L1BlockNumberAddr                = common.HexToAddress(L1BlockNumber)
	OVM_GasPriceOracleAddr           = common.HexToAddress(OVM_GasPriceOracle)
	L1BlockAddr                      = common.HexToAddress(L1Block)
	GovernanceTokenAddr              = common.HexToAddress(GovernanceToken)
)
