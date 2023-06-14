package flags

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli"
)

const envVarPrefix = "OP_BOOTNODE"

var (
	RollupConfig = cli.StringFlag{
		Name:   flags.RollupConfig.Name,
		Usage:  "Rollup chain parameters",
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "ROLLUP_CONFIG"),
	}
	Network = cli.StringFlag{
		Name:   flags.Network.Name,
		Usage:  fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVar: opservice.PrefixEnvVar(envVarPrefix, "NETWORK"),
	}
)

var Flags = []cli.Flag{
	RollupConfig,
	Network,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(envVarPrefix)...)
}
