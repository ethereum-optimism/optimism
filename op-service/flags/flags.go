package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliiface"
)

const (
	RollupConfigFlagName = "rollup.config"
	NetworkFlagName      = "network"
)

// OverridableForks lists all forks that can be overridden via CLI flags or env vars.
// It's all mainline forks from Canyon onwards, plus all optional forks.
var OverridableForks = append(forks.From(forks.Canyon), forks.AllOpt...)

func OverrideName(f forks.Name) string { return "override." + string(f) }

// KeepKarstUpgradeGasOverrideName overrides the keep_karst_upgrade_gas rollup-config flag. It is a
// behavioral bool rather than a fork activation time, so it is registered separately from the
// per-fork timestamp overrides above.
const KeepKarstUpgradeGasOverrideName = "override.keep-karst-upgrade-gas"

func OverrideEnvVar(envPrefix string, fork forks.Name) []string {
	return opservice.PrefixEnvVar(envPrefix, "OVERRIDE_"+strings.ToUpper(string(fork)))
}

func CLIFlags(envPrefix string, category string) []cli.Flag {
	return append(CLIOverrideFlags(envPrefix, category),
		CLINetworkFlag(envPrefix, category),
		CLIRollupConfigFlag(envPrefix, category),
	)
}

func CLIOverrideFlags(envPrefix string, category string) []cli.Flag {
	var flags []cli.Flag
	for _, fork := range OverridableForks {
		flags = append(flags,
			&cli.Uint64Flag{
				Name:     OverrideName(fork),
				Usage:    fmt.Sprintf("Manually specify the %s fork timestamp, overriding the bundled setting", fork),
				EnvVars:  OverrideEnvVar(envPrefix, fork),
				Category: category,
			})
	}
	flags = append(flags,
		&cli.BoolFlag{
			Name:     KeepKarstUpgradeGasOverrideName,
			Usage:    "Manually set keep_karst_upgrade_gas, overriding the bundled setting. When true, the Karst activation block's one-time upgrade gas is kept on every later block (for chains that activated Karst with the leak baked into their history).",
			EnvVars:  opservice.PrefixEnvVar(envPrefix, "OVERRIDE_KEEP_KARST_UPGRADE_GAS"),
			Category: category,
		})
	return flags
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

func CheckRequiredXor(ctx cliiface.Context) error {
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
