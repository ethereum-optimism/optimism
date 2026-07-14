package fetch

import (
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli/v2"
)

const EnvVarPrefix = "OP_FETCHER"

var GlobalFlags = append([]cli.Flag{}, oplog.CLIFlags(EnvVarPrefix)...)

var (
	L1RPCURLFlag = &cli.StringFlag{
		Name:     "l1-rpc-url",
		Usage:    "L1 RPC URL",
		Required: true,
	}
	SystemConfigProxyFlag = &cli.StringFlag{
		Name:     "system-config",
		Usage:    "contract address for SystemConfigProxy",
		Required: true,
	}
	L1StandardBridgeProxyFlag = &cli.StringFlag{
		Name:     "l1-standard-bridge",
		Usage:    "contract address for L1StandardBridgeProxy",
		Required: true,
	}
	OutputFileFlag = &cli.StringFlag{
		Name:  "output-file",
		Usage: "(optional) file to write output json",
	}
	ScriptEngineFlag = &cli.StringFlag{
		Name:  "script-engine",
		Usage: "script engine to run FetchChainInfo: \"go\" (in-process script.Host) or \"rust\" (out-of-process op-script-engine)",
		Value: string(env.DefaultScriptEngine),
	}
)

var FetchChainInfoFlags = []cli.Flag{
	L1RPCURLFlag,
	OutputFileFlag,
	SystemConfigProxyFlag,
	L1StandardBridgeProxyFlag,
	ScriptEngineFlag,
}
