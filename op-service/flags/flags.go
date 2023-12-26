package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RollupConfigFlagName   = "rollup.config"
	NetworkFlagName        = "network"
	CanyonOverrideFlagName = "override.canyon"
	DeltaOverrideFlagName  = "override.delta"
)

func CLIFlags(envPrefix string) []cli.Flag {
	return []cli.Flag{
		&cli.Uint64Flag{
			Name:    CanyonOverrideFlagName,
			Usage:   "Manually specify the Canyon fork timestamp, overriding the bundled setting",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "OVERRIDE_CANYON"),
			Hidden:  false,
		},
		&cli.Uint64Flag{
			Name:    DeltaOverrideFlagName,
			Usage:   "Manually specify the Delta fork timestamp, overriding the bundled setting",
			EnvVars: opservice.PrefixEnvVar(envPrefix, "OVERRIDE_DELTA"),
			Hidden:  false,
		},
		CLINetworkFlag(envPrefix),
		CLIRollupConfigFlag(envPrefix),
	}
}

func CLINetworkFlag(envPrefix string) cli.Flag {
	return &cli.StringFlag{
		Name:    NetworkFlagName,
		Usage:   fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars: opservice.PrefixEnvVar(envPrefix, "NETWORK"),
	}
}

func CLIRollupConfigFlag(envPrefix string) cli.Flag {
	return &cli.StringFlag{
		Name:    RollupConfigFlagName,
		Usage:   "Rollup chain parameters",
		EnvVars: opservice.PrefixEnvVar(envPrefix, "ROLLUP_CONFIG"),
	}
}

// This checks flags that are exclusive & required. Specifically for each
// set of flags, exactly one flag must be set.
var requiredXorFlags = [][]string{
	// TODO(client-pod#391): Re-enable this check at a later point
	// {
	// 	RollupConfigFlagName,
	// 	NetworkFlagName,
	// },
}

func CheckRequiredXor(ctx *cli.Context) error {
	for _, flagSet := range requiredXorFlags {
		var setCount int
		for _, flagName := range flagSet {
			if ctx.IsSet(flagName) {
				setCount++
			}
		}
		if setCount != 1 {
			return fmt.Errorf("exactly one of the following flags must be set: %s", strings.Join(flagSet, ", "))
		}
	}
	return nil
}
