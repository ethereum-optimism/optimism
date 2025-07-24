package flags

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/syncnode"
)

const EnvVarPrefix = "OP_SUPERVISOR"

var (
	ErrRequiredFlagMissing = fmt.Errorf("required flag is missing")
	ErrConflictingFlags    = fmt.Errorf("conflicting flags")
)

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	L1RPCFlag = &cli.StringFlag{
		Name:    "l1-rpc",
		Usage:   "L1 RPC source.",
		EnvVars: prefixEnvVars("L1_RPC"),
	}
	L2ConsensusNodesFlag = &cli.StringSliceFlag{
		Name:    "l2-consensus.nodes",
		Usage:   "L2 Consensus rollup node RPC addresses (with auth).",
		EnvVars: prefixEnvVars("L2_CONSENSUS_NODES"),
	}
	L2ConsensusJWTSecret = &cli.StringSliceFlag{
		Name: "l2-consensus.jwt-secret",
		Usage: "Path to JWT secret key. Keys are 32 bytes, hex encoded in a file. " +
			"If multiple paths are specified, secrets are assumed to match l2-consensus-nodes order.",
		EnvVars:   prefixEnvVars("L2_CONSENSUS_JWT_SECRET"),
		Value:     cli.NewStringSlice(),
		TakesFile: true,
	}
	DataDirFlag = &cli.PathFlag{
		Name:    "datadir",
		Usage:   "Directory to store data generated as part of responding to games",
		EnvVars: prefixEnvVars("DATADIR"),
	}
	DataDirSyncEndpointFlag = &cli.PathFlag{
		Name:    "datadir.sync-endpoint",
		Usage:   "op-supervisor endpoint to sync databases from",
		EnvVars: prefixEnvVars("DATADIR_SYNC_ENDPOINT"),
	}
	NetworkFlag = &cli.StringSliceFlag{
		Name:    "networks",
		Aliases: []string{"network"},
		Usage:   fmt.Sprintf("Predefined network selection. Available networks: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars: append(prefixEnvVars("NETWORKS"), prefixEnvVars("NETWORK")...),
	}
	DependencySetFlag = &cli.PathFlag{
		Name:      "dependency-set",
		Usage:     "Dependency-set configuration, point at JSON file.",
		EnvVars:   prefixEnvVars("DEPENDENCY_SET"),
		TakesFile: true,
	}
	RollupConfigPathsFlag = &cli.StringFlag{
		Name: "rollup-config-paths",
		Usage: "Path pattern to op-node rollup.json configs to load as a rollup config set. " +
			"The pattern should use the Go filepath glob sytax, e.g. '/configs/rollup-*.json' " +
			"When using this flag, the L1 timestamps are loaded from the provided L1 RPC",
		EnvVars: prefixEnvVars("ROLLUP_CONFIG_PATHS"),
	}
	RollupConfigSetFlag = &cli.PathFlag{
		Name: "rollup-config-set",
		Usage: "Rollup config set configuration, point at JSON file. " +
			"This is an alternative to rollup-config-paths to provide the supervisor rollup config set directly.",
		EnvVars:   prefixEnvVars("ROLLUP_CONFIG_SET"),
		TakesFile: true,
	}
	MockRunFlag = &cli.BoolFlag{
		Name:    "mock-run",
		Usage:   "Mock run, no actual backend used, just presenting the service",
		EnvVars: prefixEnvVars("MOCK_RUN"),
		Hidden:  true, // this is for testing only
	}
	RPCVerificationWarningsFlag = &cli.BoolFlag{
		Name:    "rpc-verification-warnings",
		Usage:   "Enable asynchronous RPC verification of DB checkAccess call in the CheckAccessList endpoint, indicating warnings as a metric",
		EnvVars: prefixEnvVars("RPC_VERIFICATION_WARNINGS"),
		Value:   false,
	}
)

var requiredFlags = []cli.Flag{
	L1RPCFlag,
	L2ConsensusNodesFlag,
	L2ConsensusJWTSecret,
	DataDirFlag,
}

var optionalFlags = []cli.Flag{
	NetworkFlag,
	MockRunFlag,
	DataDirSyncEndpointFlag,
	RPCVerificationWarningsFlag,
	DependencySetFlag,
	RollupConfigPathsFlag,
	RollupConfigSetFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(Flags, requiredFlags...)
	Flags = append(Flags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func checkRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("%w: %s", ErrRequiredFlagMissing, f.Names()[0])
		}
	}
	if ctx.IsSet(NetworkFlag.Name) {
		if err := checkNotSet(ctx, DependencySetFlag, RollupConfigPathsFlag, RollupConfigSetFlag); err != nil {
			return fmt.Errorf("%w must not be set with %s", err, NetworkFlag.Name)
		}
	} else if !ctx.IsSet(DependencySetFlag.Name) || (!ctx.IsSet(RollupConfigSetFlag.Name) && !ctx.IsSet(RollupConfigPathsFlag.Name)) {
		return fmt.Errorf("%w: either %s or %s and one of %s must be set", ErrRequiredFlagMissing,
			NetworkFlag.Name, DependencySetFlag.Name, flagNames(RollupConfigSetFlag, RollupConfigPathsFlag))
	} else if err := checkGroupUnique(ctx, RollupConfigPathsFlag, RollupConfigSetFlag); err != nil {
		return err
	}
	return nil
}

func checkGroupUnique(ctx *cli.Context, group ...cli.Flag) error {
	var set int
	for _, f := range group {
		if ctx.IsSet(f.Names()[0]) {
			set++
		}
	}
	if set > 1 {
		return fmt.Errorf("%w: only one of %s can be set", ErrConflictingFlags, flagNames(group...))
	}
	return nil
}

func checkNotSet(ctx *cli.Context, flags ...cli.Flag) error {
	for _, flag := range flags {
		if ctx.IsSet(flag.Names()[0]) {
			return fmt.Errorf("%w: %s", ErrConflictingFlags, flagNames(flags...))
		}
	}
	return nil
}

func flagNames(flags ...cli.Flag) string {
	names := make([]string, 0, len(flags))
	for _, f := range flags {
		names = append(names, f.Names()[0])
	}
	return strings.Join(names, ", ")
}

func ConfigFromCLI(ctx *cli.Context, version string) (*config.Config, error) {
	if err := checkRequired(ctx); err != nil {
		return nil, err
	}
	c := &config.Config{
		Version:                 version,
		LogConfig:               oplog.ReadCLIConfig(ctx),
		MetricsConfig:           opmetrics.ReadCLIConfig(ctx),
		PprofConfig:             oppprof.ReadCLIConfig(ctx),
		RPC:                     oprpc.ReadCLIConfig(ctx),
		MockRun:                 ctx.Bool(MockRunFlag.Name),
		RPCVerificationWarnings: ctx.Bool(RPCVerificationWarningsFlag.Name),
		L1RPC:                   ctx.String(L1RPCFlag.Name),
		SyncSources:             syncSourceSetups(ctx),
		Datadir:                 ctx.Path(DataDirFlag.Name),
		DatadirSyncEndpoint:     ctx.Path(DataDirSyncEndpointFlag.Name),
	}
	if ctx.IsSet(RollupConfigSetFlag.Name) {
		c.FullConfigSetSource = &depset.FullConfigSetSourceMerged{
			RollupConfigSetSource: &depset.JSONRollupConfigSetLoader{Path: ctx.Path(RollupConfigSetFlag.Name)},
			DependencySetSource:   &depset.JSONDependencySetLoader{Path: ctx.Path(DependencySetFlag.Name)},
		}
	} else if ctx.IsSet(RollupConfigPathsFlag.Name) {
		c.FullConfigSetSource = &depset.FullConfigSetSourceMerged{
			RollupConfigSetSource: &depset.JSONRollupConfigsLoader{
				PathPattern: ctx.String(RollupConfigPathsFlag.Name),
				L1RPCURL:    ctx.String(L1RPCFlag.Name),
			},
			DependencySetSource: &depset.JSONDependencySetLoader{Path: ctx.Path(DependencySetFlag.Name)},
		}
	} else if ctx.IsSet(NetworkFlag.Name) {
		networks := ctx.StringSlice(NetworkFlag.Name)
		source, err := depset.NewRegistryFullConfigSetSource(ctx.String(L1RPCFlag.Name), networks)
		if err != nil {
			return nil, err
		}
		c.FullConfigSetSource = source
	} else {
		return nil, fmt.Errorf("%w: either %s or %s and one of %s must be set", ErrRequiredFlagMissing,
			NetworkFlag.Name, DependencySetFlag.Name, flagNames(RollupConfigSetFlag, RollupConfigPathsFlag))
	}
	return c, nil
}

// syncSourceSetups creates a sync source collection, from CLI arguments.
// These sources can share JWT secret configuration.
func syncSourceSetups(ctx *cli.Context) syncnode.SyncNodeCollection {
	return &syncnode.CLISyncNodes{
		Endpoints:      filterEmpty(ctx.StringSlice(L2ConsensusNodesFlag.Name)),
		JWTSecretPaths: filterEmpty(ctx.StringSlice(L2ConsensusJWTSecret.Name)),
	}
}

// filterEmpty cleans empty entries from a string-slice flag,
// which has the potential to have empty strings.
func filterEmpty(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
