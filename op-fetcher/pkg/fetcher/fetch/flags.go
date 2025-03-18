package fetch

import (
	"github.com/urfave/cli/v2"
)

var (
	L1RPCURLFlag = &cli.StringFlag{
		Name:     "l1-rpc-url",
		Usage:    "L1 RPC URL",
		Required: true,
	}
	ChainConfigDirFlag = &cli.StringFlag{
		Name:     "chain-config-dir",
		Usage:    "Directory containing chain configuration TOML files",
		Required: true,
	}
)

var FetchChainInfoFlags = []cli.Flag{
	L1RPCURLFlag,
	ChainConfigDirFlag,
}
