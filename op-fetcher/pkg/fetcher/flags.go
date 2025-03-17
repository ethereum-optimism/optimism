package fetcher

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	L1RPCURLFlag = &cli.StringFlag{
		Name:     "l1-rpc-url",
		Usage:    "L1 RPC URL",
		Required: true,
	}
	SystemConfigFlag = &cli.StringFlag{
		Name:     "system-config",
		Usage:    "System config address as hex string",
		Required: true,
	}
	L1StandardBridgeFlag = &cli.StringFlag{
		Name:     "l1-standard-bridge",
		Usage:    "L1 standard bridge address as hex string",
		Required: true,
	}
)

// FetchChainInfoFlags contains all the flags needed for fetching chain info
var FetchChainInfoFlags = []cli.Flag{
	L1RPCURLFlag,
	SystemConfigFlag,
	L1StandardBridgeFlag,
}

// Config represents the configuration for the fetcher service
type Config struct {
	L1RPCURL         string
	SystemConfig     common.Address
	L1StandardBridge common.Address
}

// NewConfig creates a new Config from CLI context
func NewConfig(ctx *cli.Context) (*Config, error) {
	systemConfig := common.HexToAddress(ctx.String(SystemConfigFlag.Name))
	l1StandardBridge := common.HexToAddress(ctx.String(L1StandardBridgeFlag.Name))

	return &Config{
		L1RPCURL:         ctx.String(L1RPCURLFlag.Name),
		SystemConfig:     systemConfig,
		L1StandardBridge: l1StandardBridge,
	}, nil
}
