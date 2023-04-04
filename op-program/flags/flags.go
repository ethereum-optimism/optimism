package flags

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	service "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli"
)

const envVarPrefix = "OP_PROGRAM"

var (
	RollupConfig = cli.StringFlag{
		Name:   "rollup.config",
		Usage:  "Rollup chain parameters",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "ROLLUP_CONFIG"),
	}
	Network = cli.StringFlag{
		Name:   "network",
		Usage:  fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVar: service.PrefixEnvVar(envVarPrefix, "NETWORK"),
	}
	L2NodeAddr = cli.StringFlag{
		Name:   "l2",
		Usage:  "Address of L2 JSON-RPC endpoint to use (eth and debug namespace required)",
		EnvVar: service.PrefixEnvVar(envVarPrefix, "L2_RPC"),
	}
)

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

var programFlags = []cli.Flag{
	RollupConfig,
	Network,
	L2NodeAddr,
}

func init() {
	Flags = append(Flags, oplog.CLIFlags(envVarPrefix)...)
	Flags = append(Flags, programFlags...)
}

func CheckRequired(ctx *cli.Context) error {
	rollupConfig := ctx.GlobalString(RollupConfig.Name)
	network := ctx.GlobalString(Network.Name)
	if rollupConfig == "" && network == "" {
		return fmt.Errorf("flag %s or %s is required", RollupConfig.Name, Network.Name)
	}
	if rollupConfig != "" && network != "" {
		return fmt.Errorf("cannot specify both %s and %s", RollupConfig.Name, Network.Name)
	}
	return nil
}
