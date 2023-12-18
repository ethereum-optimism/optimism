package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const EnvVarPrefix = "SUPERCHAIN"

var (
	RollupConfig = &cli.StringFlag{
		Name:    "superchain.rollup.config",
		Usage:   "Rollup chain parameters",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "ROLLUP_CONFIG"),
	}
	Network = &cli.StringFlag{
		Name:    "superchain.network",
		Usage:   fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "NETWORK"),
	}
	CanyonOverrideFlag = &cli.Uint64Flag{
		Name:    "superchain.override.canyon",
		Usage:   "Manually specify the Canyon fork timestamp, overriding the bundled setting",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "OVERRIDE_CANYON"),
		Hidden:  false,
	}
	DeltaOverrideFlag = &cli.Uint64Flag{
		Name:    "superchain.override.delta",
		Usage:   "Manually specify the Delta fork timestamp, overriding the bundled setting",
		EnvVars: opservice.PrefixEnvVar(EnvVarPrefix, "OVERRIDE_DELTA"),
		Hidden:  false,
	}
)

var requiredFlags = []cli.Flag{}

var optionalFlags = []cli.Flag{
	RollupConfig,
	Network,
	CanyonOverrideFlag,
	DeltaOverrideFlag,
}

// Flags contains the list of configuration options available to the service.
var Flags []cli.Flag

func init() {
	Flags = append(requiredFlags, optionalFlags...)
}

// This checks flags that are exclusive & required. Specifically for each
// set of flags, exactly one flag must be set.
var requiredXorFlags = [][]string{
	{
		RollupConfig.Name,
		Network.Name,
	},
}

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	for _, flags := range requiredXorFlags {
		var count int
		for _, flag := range flags {
			if ctx.IsSet(flag) {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("exactly one of the flags in the set [%s] must be set", strings.Join(flags, ", "))
		}
	}
	return nil
}
