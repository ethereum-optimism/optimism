package flags

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

var deprecatedFlags = func() []cli.Flag {
	const usage = "Deprecated and ignored. Configure the paired op-node instead."
	env := func(name string) []string {
		return []string{opservice.FlagNameToEnvVarName(name, EnvVarPrefix)}
	}
	const network = "network"
	const rollupConfig = "rollup.config"
	const keepKarstUpgradeGas = "override.keep-karst-upgrade-gas"
	flags := []cli.Flag{
		&cli.StringFlag{Name: network, Usage: usage, EnvVars: env(network), Hidden: true},
		&cli.StringFlag{Name: rollupConfig, Usage: usage, EnvVars: env(rollupConfig), Hidden: true},
		&cli.BoolFlag{Name: keepKarstUpgradeGas, Usage: usage, EnvVars: env(keepKarstUpgradeGas), Hidden: true},
	}
	// Keep the timestamp overrides accepted by conductor during the deprecation window.
	for _, fork := range []forks.Name{
		forks.Canyon, forks.Delta, forks.Ecotone, forks.Fjord, forks.Granite,
		forks.Holocene, forks.Isthmus, forks.Jovian, forks.Karst, forks.Lagoon, forks.PectraBlobSchedule,
	} {
		name := "override." + string(fork)
		flags = append(flags, &cli.Uint64Flag{Name: name, Usage: usage, EnvVars: env(name), Hidden: true})
	}
	return flags
}()

// WarnDeprecated reports legacy settings provided by the operator.
func WarnDeprecated(ctx *cli.Context, log log.Logger) {
	for _, flag := range deprecatedFlags {
		name := flag.Names()[0]
		if ctx.IsSet(name) {
			log.Warn("Ignoring deprecated op-conductor flag. It will be removed in an upcoming release after 2027-03-01. Configure the paired op-node instead.", "flag", name)
		}
	}
}
