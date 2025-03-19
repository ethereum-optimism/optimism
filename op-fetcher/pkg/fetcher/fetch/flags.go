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
		Usage:    "directory containing input chain config toml files",
		Required: true,
	}
	OutputDirFlag = &cli.StringFlag{
		Name:     "output-dir",
		Usage:    "directory to write output json files (one per input chain config toml file)",
		Required: true,
	}
)

var FetchChainInfoFlags = []cli.Flag{
	L1RPCURLFlag,
	ChainConfigDirFlag,
	OutputDirFlag,
}
