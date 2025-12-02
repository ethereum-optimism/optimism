package deployer

import "github.com/ethereum/go-ethereum/common"

// Constants for Sepolia chain.
var (
	SepoliaChainID                  uint64         = 11155111
	DefaultL1ProxyAdminOwnerSepolia common.Address = common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2")
	DefaultSystemConfigProxySepolia common.Address = common.HexToAddress("0x034edD2A225f7f429A63E0f1D2084B9E0A93b538")
)
