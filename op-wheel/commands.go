package wheel

import (
	"github.com/urfave/cli"

	opservice "github.com/ethereum-optimism/optimism/op-service"
)

var (
	GlobalGethLogLvlFlag = cli.StringFlag{
		Name:   "geth-log-level",
		Usage:  "Set the global geth logging level",
		EnvVar: opservice.PrefixEnvVar("OP_WHEEL", "GETH_LOG_LEVEL"),
		Value:  "error",
	}
)
