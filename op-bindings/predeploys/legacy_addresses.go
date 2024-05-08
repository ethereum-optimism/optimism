package predeploys

import "github.com/ethereum/go-ethereum/common"

const (
	LegacyERC20ETH = "0x4200000000000000000000000000000000000006"
)

var (
	LegacyERC20ETHAddr = common.HexToAddress(LegacyERC20ETH)
)
