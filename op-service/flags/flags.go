package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
)

const (
	RollupConfigFlagName     = "rollup.config"
	NetworkFlagName          = "network"
	CanyonOverrideFlagName   = "override.canyon"
	DeltaOverrideFlagName    = "override.delta"
	EcotoneOverrideFlagName  = "override.ecotone"
	FjordOverrideFlagName    = "override.fjord"
	GraniteOverrideFlagName  = "override.granite"
	HoloceneOverrideFlagName = "override.holocene"
)

func CLIFlags(envPrefix string, category string) []cli.Flag {
	return []cli.Flag{
		&cli.Uint64Flag{
			Name:     CanyonOverrideFlagName,
			Usage:    "Manually specify the Canyon fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_CANYON"),
			Hidden:   false,
			Category: category,
		},
		&cli.Uint64Flag{
			Name:     DeltaOverrideFlagName,
			Usage:    "Manually specify the Delta fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_DELTA"),
			Hidden:   false,
			Category: category,
		},
		&cli.Uint64Flag{
			Name:     EcotoneOverrideFlagName,
			Usage:    "Manually specify the Ecotone fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_ECOTONE"),
			Hidden:   false,
			Category: category,
		},
		&cli.Uint64Flag{
			Name:     FjordOverrideFlagName,
			Usage:    "Manually specify the Fjord fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_FJORD"),
			Hidden:   false,
			Category: category,
		},
		&cli.Uint64Flag{
			Name:     GraniteOverrideFlagName,
			Usage:    "Manually specify the Granite fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_GRANITE"),
			Hidden:   false,
			Category: category,
		},
		&cli.Uint64Flag{
			Name:     HoloceneOverrideFlagName,
			Usage:    "Manually specify the Holocene fork timestamp, overriding the bundled setting",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_HOLOCENE"),
			Hidden:   false,
			Category: category,
		},
		CLINetworkFlag(envPrefix, category),
		CLIRollupConfigFlag(envPrefix, category),
	}
}

func CLINetworkFlag(envPrefix string, category string) cli.Flag {
	return &cli.StringFlag{
		Name:     NetworkFlagName,
		Usage:    fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars:  opservice.PrefixEnvVar(envPrefix, "NETWORK"),
		Category: category,
	}
}

func CLIRollupConfigFlag(envPrefix string, category string) cli.Flag {
	return &cli.StringFlag{
		Name:     RollupConfigFlagName,
		Usage:    "Rollup chain parameters",
		EnvVars:  opservice.PrefixEnvVar(envPrefix, "ROLLUP_CONFIG"),
		Category: category,
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
