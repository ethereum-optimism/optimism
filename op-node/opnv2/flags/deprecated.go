package flags

import (
	"github.com/urfave/cli/v2"
)

// op-node legacy flags, ignored, here for compatibility with old CLI configurations.
var (
	L2EngineSyncEnabledFlag = &cli.BoolFlag{
		Name:    "l2.engine-sync",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: prefixEnvVars("L2_ENGINE_SYNC_ENABLED"),
		Hidden:  true,
	}
	SkipSyncStartCheckFlag = &cli.BoolFlag{
		Name:    "l2.skip-sync-start-check",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: prefixEnvVars("L2_SKIP_SYNC_START_CHECK"),
		Hidden:  true,
	}
	BetaExtraNetworksFlag = &cli.BoolFlag{
		Name:    "beta.extra-networks",
		Usage:   "WARNING: Legacy flag. No longer in use.", // networks are always available
		EnvVars: prefixEnvVars("BETA_EXTRA_NETWORKS"),
		Hidden:  true,
	}
	BackupL2UnsafeSyncRPCFlag = &cli.StringFlag{
		Name:    "l2.backup-unsafe-sync-rpc",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: prefixEnvVars("L2_BACKUP_UNSAFE_SYNC_RPC"),
		Hidden:  true,
	}
	BackupL2UnsafeSyncRPCTrustRPCFlag = &cli.StringFlag{
		Name:    "l2.backup-unsafe-sync-rpc.trustrpc",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: prefixEnvVars("L2_BACKUP_UNSAFE_SYNC_RPC_TRUST_RPC"),
		Hidden:  true,
	}
	SnapshotLogFlag = &cli.StringFlag{
		Name:     "snapshotlog.file",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("SNAPSHOT_LOG"),
		Category: OperationsCategory,
		Hidden:   true, // non-critical function, removed, flag is no-op to avoid breaking setups.
	}
	HeartbeatEnabledFlag = &cli.BoolFlag{
		Name:     "heartbeat.enabled",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("HEARTBEAT_ENABLED"),
		Category: OperationsCategory,
		Hidden:   true,
	}
	HeartbeatMonikerFlag = &cli.StringFlag{
		Name:     "heartbeat.moniker",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("HEARTBEAT_MONIKER"),
		Category: OperationsCategory,
		Hidden:   true,
	}
	HeartbeatURLFlag = &cli.StringFlag{
		Name:     "heartbeat.url",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("HEARTBEAT_URL"),
		Category: OperationsCategory,
		Hidden:   true,
	}
	PeerScoringNameFlag = &cli.StringFlag{
		Name:     "p2p.scoring.peers", // Use p2p.scoring instead
		Usage:    "WARNING: Legacy flag. No longer in use.",
		Hidden:   true,
		Category: P2PCategory,
	}
	PeerScoringBandsNameFlag = &cli.StringFlag{
		Name:     "p2p.score.bands",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		Hidden:   true,
		Category: P2PCategory,
	}
	TopicScoringNameFlag = &cli.StringFlag{
		Name:     "p2p.scoring.topics",
		Usage:    "WARNING: Legacy flag. No longer in use.", // Use p2p.scoring instead
		Hidden:   true,
		Category: P2PCategory,
	}
	SyncModeFlag = &cli.StringFlag{
		Name:     "syncmode",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("SYNCMODE"),
		Category: RollupCategory,
		Hidden:   true,
	}
	L2EngineKindFlag = &cli.StringFlag{
		Name: "l2.enginekind",
		// used to inform syncmode differences
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("L2_ENGINE_KIND"),
		Category: RollupCategory,
		Hidden:   true,
	}
	RollupHaltFlag = &cli.StringFlag{
		Name:     "rollup.halt",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("ROLLUP_HALT"),
		Category: RollupCategory,
		Hidden:   true,
	}
	RollupLoadProtocolVersionsFlag = &cli.BoolFlag{
		Name:     "rollup.load-protocol-versions",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("ROLLUP_LOAD_PROTOCOL_VERSIONS"),
		Category: RollupCategory,
		Hidden:   true,
	}
	IgnoreMissingPectraBlobScheduleFlag = &cli.BoolFlag{
		Name:     "ignore-missing-pectra-blob-schedule",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("IGNORE_MISSING_PECTRA_BLOB_SCHEDULE"),
		Category: RollupCategory,
		Hidden:   true,
	}
	SafeDBPathFlag = &cli.StringFlag{
		Name:     "safedb.path",
		Usage:    "WARNING: Legacy flag. No longer in use.",
		EnvVars:  prefixEnvVars("SAFEDB_PATH"),
		Category: OperationsCategory,
		Hidden:   true,
	}
)

// op-supervisor flags
var (
	L2ConsensusNodesFlag = &cli.StringSliceFlag{
		Name:    "l2-consensus.nodes",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: []string{"OP_SUPERVISOR_L2_CONSENSUS_NODES"},
		Hidden:  true,
	}
	L2ConsensusJWTSecretFlag = &cli.StringSliceFlag{
		Name:    "l2-consensus.jwt-secret",
		Usage:   "WARNING: Legacy flag. No longer in use.",
		EnvVars: []string{"OP_SUPERVISOR_L2_CONSENSUS_JWT_SECRET"},
		Value:   cli.NewStringSlice(),
		Hidden:  true,
	}
)

var DeprecatedFlags = []cli.Flag{
	L2EngineSyncEnabledFlag,
	SkipSyncStartCheckFlag,
	BetaExtraNetworksFlag,
	BackupL2UnsafeSyncRPCFlag,
	BackupL2UnsafeSyncRPCTrustRPCFlag,
	SnapshotLogFlag,
	HeartbeatEnabledFlag,
	HeartbeatMonikerFlag,
	HeartbeatURLFlag,
	PeerScoringNameFlag,
	PeerScoringBandsNameFlag,
	TopicScoringNameFlag,
	SyncModeFlag,
	L2EngineKindFlag,
	RollupHaltFlag,
	RollupLoadProtocolVersionsFlag,
	IgnoreMissingPectraBlobScheduleFlag,
	SafeDBPathFlag,
	L2ConsensusNodesFlag,
	L2ConsensusJWTSecretFlag,
}
