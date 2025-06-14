package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
)

var (
	DataDirFlag = &cli.PathFlag{
		Name:     "datadir", // New in op-node v2
		Usage:    "Directory to store derivation-safety and log-indexing data.",
		EnvVars:  prefixEnvVars("DATADIR"),
		Value:    "data",
		Category: RollupCategory,
	}
	NetworkFlag = &cli.StringSliceFlag{
		Name:     "networks",
		Aliases:  []string{"network"},
		Usage:    fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars:  append(prefixEnvVars("NETWORKS"), prefixEnvVars("NETWORK")...),
		Category: RollupCategory,
	}
	DependencySetFlag = &cli.PathFlag{
		Name:    "dependency-set",
		Aliases: []string{"interop.dependency-set"}, // alias from op-supervisor service
		Usage: "Dependency-set configuration, point at JSON file. " +
			"Does not have to be set if a predefined network is selected.",
		EnvVars:   prefixEnvVars("DEPENDENCY_SET"),
		TakesFile: true,
		Category:  RollupCategory,
	}
	RollupConfigPathsFlag = &cli.StringFlag{
		Name:    "rollup.config-paths",
		Aliases: []string{"rollup-config-paths"},
		Usage: "Path pattern to op-node rollup.json configs to load as a rollup config set. " +
			"For an interop-set chain the pattern should use the Go filepath glob syntax, e.g. '/configs/rollup-*.json' ",
		EnvVars:  prefixEnvVars("ROLLUP_CONFIG_PATHS"),
		Category: RollupCategory,
	}
	// RollupConfigFlag is from op-node V1, originally part of the opflags package, but customized with more description.
	RollupConfigFlag = &cli.StringFlag{
		Name: "rollup.config",
		Usage: "Rollup chain configuration parameters. Alternative to predefined network selection. " +
			"Backward-compatible alternative to rollup.config-paths flag for single-chain post-interop configurations",
		EnvVars:  prefixEnvVars("ROLLUP_CONFIG"),
		Category: RollupCategory,
	}
)

var RollupFlags = append([]cli.Flag{
	DataDirFlag,
	NetworkFlag,
	DependencySetFlag,
	RollupConfigPathsFlag,
	RollupConfigFlag,
}, opflags.OverrideCLIFlags(EnvVarPrefixOpnode, RollupCategory)...)
