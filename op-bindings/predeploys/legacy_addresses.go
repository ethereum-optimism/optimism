package predeploys

import "github.com/ethereum/go-ethereum/common"

const (
	LegacyERC20ETH = "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000"
)

var (
	LegacyERC20ETHAddr = common.HexToAddress(LegacyERC20ETH)
)
