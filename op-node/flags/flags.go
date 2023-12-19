package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	openum "github.com/ethereum-optimism/optimism/op-service/enum"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// Flags

const EnvVarPrefix = "OP_NODE"

func prefixEnvVars(name string) []string {
	return []string{EnvVarPrefix + "_" + name}
}

var (
	/* Required Flags */
	L1NodeAddr = &cli.StringFlag{
		Name:    "l1",
		Usage:   "Address of L1 User JSON-RPC endpoint to use (eth namespace required)",
		Value:   "http://127.0.0.1:8545",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	L2EngineAddr = &cli.StringFlag{
		Name:    "l2",
		Usage:   "Address of L2 Engine JSON-RPC endpoints to use (engine and eth namespace required)",
		EnvVars: prefixEnvVars("L2_ENGINE_RPC"),
	}
	L2EngineJWTSecret = &cli.StringFlag{
		Name:        "l2.jwt-secret",
		Usage:       "Path to JWT secret key. Keys are 32 bytes, hex encoded in a file. A new key will be generated if the file is empty.",
		EnvVars:     prefixEnvVars("L2_ENGINE_AUTH"),
		Value:       "",
		Destination: new(string),
	}
	/* Optional Flags */
	SyncModeFlag = &cli.GenericFlag{
		Name:    "syncmode",
		Usage:   fmt.Sprintf("IN DEVELOPMENT: Options are: %s", openum.EnumString(sync.ModeStrings)),
		EnvVars: prefixEnvVars("SYNCMODE"),
		Value: func() *sync.Mode {
			out := sync.CLSync
			return &out
		}(),
		Hidden: true,
	}
	RPCListenAddr = &cli.StringFlag{
		Name:    "rpc.addr",
		Usage:   "RPC listening address",
		EnvVars: prefixEnvVars("RPC_ADDR"),
		Value:   "127.0.0.1",
	}
	RPCListenPort = &cli.IntFlag{
		Name:    "rpc.port",
		Usage:   "RPC listening port",
		EnvVars: prefixEnvVars("RPC_PORT"),
		Value:   9545, // Note: op-service/rpc/cli.go uses 8545 as the default.
	}
	RPCEnableAdmin = &cli.BoolFlag{
		Name:    "rpc.enable-admin",
		Usage:   "Enable the admin API (experimental)",
		EnvVars: prefixEnvVars("RPC_ENABLE_ADMIN"),
	}
	RPCAdminPersistence = &cli.StringFlag{
		Name:    "rpc.admin-state",
		Usage:   "File path used to persist state changes made via the admin API so they persist across restarts. Disabled if not set.",
		EnvVars: prefixEnvVars("RPC_ADMIN_STATE"),
	}
	L1TrustRPC = &cli.BoolFlag{
		Name:    "l1.trustrpc",
		Usage:   "Trust the L1 RPC, sync faster at risk of malicious/buggy RPC providing bad or inconsistent L1 data",
		EnvVars: prefixEnvVars("L1_TRUST_RPC"),
	}
	L1RPCProviderKind = &cli.GenericFlag{
		Name: "l1.rpckind",
		Usage: "The kind of RPC provider, used to inform optimal transactions receipts fetching, and thus reduce costs. Valid options: " +
			openum.EnumString(sources.RPCProviderKinds),
		EnvVars: prefixEnvVars("L1_RPC_KIND"),
		Value: func() *sources.RPCProviderKind {
			out := sources.RPCKindStandard
			return &out
		}(),
	}
	L1RethDBPath = &cli.StringFlag{
		Name:    "l1.rethdb",
		Usage:   "The L1 RethDB path, used to fetch receipts for L1 blocks. Only applicable when using the `reth_db` RPC kind with `l1.rpckind`.",
		EnvVars: prefixEnvVars("L1_RETHDB"),
		Hidden:  true,
	}
	L1RPCMaxConcurrency = &cli.IntFlag{
		Name:    "l1.max-concurrency",
		Usage:   "Maximum number of concurrent RPC requests to make to the L1 RPC provider.",
		EnvVars: prefixEnvVars("L1_MAX_CONCURRENCY"),
		Value:   10,
	}
	L1RPCRateLimit = &cli.Float64Flag{
		Name:    "l1.rpc-rate-limit",
		Usage:   "Optional self-imposed global rate-limit on L1 RPC requests, specified in requests / second. Disabled if set to 0.",
		EnvVars: prefixEnvVars("L1_RPC_RATE_LIMIT"),
		Value:   0,
	}
	L1RPCMaxBatchSize = &cli.IntFlag{
		Name:    "l1.rpc-max-batch-size",
		Usage:   "Maximum number of RPC requests to bundle, e.g. during L1 blocks receipt fetching. The L1 RPC rate limit counts this as N items, but allows it to burst at once.",
		EnvVars: prefixEnvVars("L1_RPC_MAX_BATCH_SIZE"),
		Value:   20,
	}
	L1HTTPPollInterval = &cli.DurationFlag{
		Name:    "l1.http-poll-interval",
		Usage:   "Polling interval for latest-block subscription when using an HTTP RPC provider. Ignored for other types of RPC endpoints.",
		EnvVars: prefixEnvVars("L1_HTTP_POLL_INTERVAL"),
		Value:   time.Second * 12,
	}
	VerifierL1Confs = &cli.Uint64Flag{
		Name:    "verifier.l1-confs",
		Usage:   "Number of L1 blocks to keep distance from the L1 head before deriving L2 data from. Reorgs are supported, but may be slow to perform.",
		EnvVars: prefixEnvVars("VERIFIER_L1_CONFS"),
		Value:   0,
	}
	SequencerEnabledFlag = &cli.BoolFlag{
		Name:    "sequencer.enabled",
		Usage:   "Enable sequencing of new L2 blocks. A separate batch submitter has to be deployed to publish the data for verifiers.",
		EnvVars: prefixEnvVars("SEQUENCER_ENABLED"),
	}
	SequencerStoppedFlag = &cli.BoolFlag{
		Name:    "sequencer.stopped",
		Usage:   "Initialize the sequencer in a stopped state. The sequencer can be started using the admin_startSequencer RPC",
		EnvVars: prefixEnvVars("SEQUENCER_STOPPED"),
	}
	SequencerMaxSafeLagFlag = &cli.Uint64Flag{
		Name:    "sequencer.max-safe-lag",
		Usage:   "Maximum number of L2 blocks for restricting the distance between L2 safe and unsafe. Disabled if 0.",
		EnvVars: prefixEnvVars("SEQUENCER_MAX_SAFE_LAG"),
		Value:   0,
	}
	SequencerL1Confs = &cli.Uint64Flag{
		Name:    "sequencer.l1-confs",
		Usage:   "Number of L1 blocks to keep distance from the L1 head as a sequencer for picking an L1 origin.",
		EnvVars: prefixEnvVars("SEQUENCER_L1_CONFS"),
		Value:   4,
	}
	L1EpochPollIntervalFlag = &cli.DurationFlag{
		Name:    "l1.epoch-poll-interval",
		Usage:   "Poll interval for retrieving new L1 epoch updates such as safe and finalized block changes. Disabled if 0 or negative.",
		EnvVars: prefixEnvVars("L1_EPOCH_POLL_INTERVAL"),
		Value:   time.Second * 12 * 32,
	}
	RuntimeConfigReloadIntervalFlag = &cli.DurationFlag{
		Name:    "l1.runtime-config-reload-interval",
		Usage:   "Poll interval for reloading the runtime config, useful when config events are not being picked up. Disabled if 0 or negative.",
		EnvVars: prefixEnvVars("L1_RUNTIME_CONFIG_RELOAD_INTERVAL"),
		Value:   time.Minute * 10,
	}
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:    "metrics.enabled",
		Usage:   "Enable the metrics server",
		EnvVars: prefixEnvVars("METRICS_ENABLED"),
	}
	MetricsAddrFlag = &cli.StringFlag{
		Name:    "metrics.addr",
		Usage:   "Metrics listening address",
		Value:   "0.0.0.0", // TODO(CLI-4159): Switch to 127.0.0.1
		EnvVars: prefixEnvVars("METRICS_ADDR"),
	}
	MetricsPortFlag = &cli.IntFlag{
		Name:    "metrics.port",
		Usage:   "Metrics listening port",
		Value:   7300,
		EnvVars: prefixEnvVars("METRICS_PORT"),
	}
	PprofEnabledFlag = &cli.BoolFlag{
		Name:    "pprof.enabled",
		Usage:   "Enable the pprof server",
		EnvVars: prefixEnvVars("PPROF_ENABLED"),
	}
	PprofAddrFlag = &cli.StringFlag{
		Name:    "pprof.addr",
		Usage:   "pprof listening address",
		Value:   "0.0.0.0", // TODO(CLI-4159): Switch to 127.0.0.1
		EnvVars: prefixEnvVars("PPROF_ADDR"),
	}
	PprofPortFlag = &cli.IntFlag{
		Name:    "pprof.port",
		Usage:   "pprof listening port",
		Value:   6060,
		EnvVars: prefixEnvVars("PPROF_PORT"),
	}
	SnapshotLog = &cli.StringFlag{
		Name:    "snapshotlog.file",
		Usage:   "Path to the snapshot log file",
		EnvVars: prefixEnvVars("SNAPSHOT_LOG"),
	}
	HeartbeatEnabledFlag = &cli.BoolFlag{
		Name:    "heartbeat.enabled",
		Usage:   "Enables or disables heartbeating",
		EnvVars: prefixEnvVars("HEARTBEAT_ENABLED"),
	}
	HeartbeatMonikerFlag = &cli.StringFlag{
		Name:    "heartbeat.moniker",
		Usage:   "Sets a moniker for this node",
		EnvVars: prefixEnvVars("HEARTBEAT_MONIKER"),
	}
	HeartbeatURLFlag = &cli.StringFlag{
		Name:    "heartbeat.url",
		Usage:   "Sets the URL to heartbeat to",
		EnvVars: prefixEnvVars("HEARTBEAT_URL"),
		Value:   "https://heartbeat.optimism.io",
	}
	RollupHalt = &cli.StringFlag{
		Name:    "rollup.halt",
		Usage:   "Opt-in option to halt on incompatible protocol version requirements of the given level (major/minor/patch/none), as signaled onchain in L1",
		EnvVars: prefixEnvVars("ROLLUP_HALT"),
	}
	RollupLoadProtocolVersions = &cli.BoolFlag{
		Name:    "rollup.load-protocol-versions",
		Usage:   "Load protocol versions from the superchain L1 ProtocolVersions contract (if available), and report in logs and metrics",
		EnvVars: prefixEnvVars("ROLLUP_LOAD_PROTOCOL_VERSIONS"),
	}
	/* Deprecated Flags */
	L2EngineSyncEnabled = &cli.BoolFlag{
		Name:    "l2.engine-sync",
		Usage:   "WARNING: Deprecated. Use --syncmode=execution-layer instead",
		EnvVars: prefixEnvVars("L2_ENGINE_SYNC_ENABLED"),
		Value:   false,
		Hidden:  true,
	}
	SkipSyncStartCheck = &cli.BoolFlag{
		Name: "l2.skip-sync-start-check",
		Usage: "Skip sanity check of consistency of L1 origins of the unsafe L2 blocks when determining the sync-starting point. " +
			"This defers the L1-origin verification, and is recommended to use in when utilizing l2.engine-sync",
		EnvVars: prefixEnvVars("L2_SKIP_SYNC_START_CHECK"),
		Value:   false,
		Hidden:  true,
	}
	BetaExtraNetworks = &cli.BoolFlag{
		Name:    "beta.extra-networks",
		Usage:   "Legacy flag, ignored, all superchain-registry networks are enabled by default.",
		EnvVars: prefixEnvVars("BETA_EXTRA_NETWORKS"),
		Hidden:  true, // hidden, this is deprecated, the flag is not used anymore.
	}
	BackupL2UnsafeSyncRPC = &cli.StringFlag{
		Name:    "l2.backup-unsafe-sync-rpc",
		Usage:   "Set the backup L2 unsafe sync RPC endpoint.",
		EnvVars: prefixEnvVars("L2_BACKUP_UNSAFE_SYNC_RPC"),
		Hidden:  true,
	}
	BackupL2UnsafeSyncRPCTrustRPC = &cli.StringFlag{
		Name: "l2.backup-unsafe-sync-rpc.trustrpc",
		Usage: "Like l1.trustrpc, configure if response data from the RPC needs to be verified, e.g. blockhash computation." +
			"This does not include checks if the blockhash is part of the canonical chain.",
		EnvVars: prefixEnvVars("L2_BACKUP_UNSAFE_SYNC_RPC_TRUST_RPC"),
		Hidden:  true,
	}
)

var requiredFlags = []cli.Flag{
	L1NodeAddr,
	L2EngineAddr,
	L2EngineJWTSecret,
}

var optionalFlags = []cli.Flag{
	SyncModeFlag,
	RPCListenAddr,
	RPCListenPort,
	L1TrustRPC,
	L1RPCProviderKind,
	L1RPCRateLimit,
	L1RPCMaxBatchSize,
	L1RPCMaxConcurrency,
	L1HTTPPollInterval,
	VerifierL1Confs,
	SequencerEnabledFlag,
	SequencerStoppedFlag,
	SequencerMaxSafeLagFlag,
	SequencerL1Confs,
	L1EpochPollIntervalFlag,
	RuntimeConfigReloadIntervalFlag,
	RPCEnableAdmin,
	RPCAdminPersistence,
	MetricsEnabledFlag,
	MetricsAddrFlag,
	MetricsPortFlag,
	PprofEnabledFlag,
	PprofAddrFlag,
	PprofPortFlag,
	SnapshotLog,
	HeartbeatEnabledFlag,
	HeartbeatMonikerFlag,
	HeartbeatURLFlag,
	RollupHalt,
	RollupLoadProtocolVersions,
	L1RethDBPath,
}

var DeprecatedFlags = []cli.Flag{
	L2EngineSyncEnabled,
	SkipSyncStartCheck,
	BetaExtraNetworks,
	BackupL2UnsafeSyncRPC,
	BackupL2UnsafeSyncRPCTrustRPC,
	// Deprecated P2P Flags are added at the init step
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func init() {
	DeprecatedFlags = append(DeprecatedFlags, deprecatedP2PFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, P2PFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, DeprecatedFlags...)
	optionalFlags = append(optionalFlags, opflags.CLIFlags(EnvVarPrefix)...)
	Flags = append(requiredFlags, optionalFlags...)
}

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return opflags.CheckRequiredXor(ctx)
}
