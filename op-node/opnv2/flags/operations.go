package flags

import (
	"github.com/urfave/cli/v2"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
)

var (
	RPCListenAddr = &cli.StringFlag{
		Name:     "rpc.addr",
		Usage:    "RPC listening address",
		EnvVars:  prefixEnvVars("RPC_ADDR"),
		Value:    "127.0.0.1",
		Category: OperationsCategory,
	}
	RPCListenPort = &cli.IntFlag{
		Name:     "rpc.port",
		Usage:    "RPC listening port",
		EnvVars:  prefixEnvVars("RPC_PORT"),
		Value:    9545, // Note: op-service/rpc/cli.go uses 8545 as the default.
		Category: OperationsCategory,
	}
	RPCEnableAdmin = &cli.BoolFlag{
		Name:     "rpc.enable-admin",
		Usage:    "Enable the admin API (experimental)",
		EnvVars:  prefixEnvVars("RPC_ENABLE_ADMIN"),
		Category: OperationsCategory,
	}
	RPCAdminPersistence = &cli.StringFlag{
		Name:     "rpc.admin-state",
		Usage:    "File path used to persist state changes made via the admin API so they persist across restarts. Disabled if not set.",
		EnvVars:  prefixEnvVars("RPC_ADMIN_STATE"),
		Category: OperationsCategory,
	}
	MetricsEnabledFlag = &cli.BoolFlag{
		Name:     "metrics.enabled",
		Usage:    "Enable the metrics server",
		EnvVars:  prefixEnvVars("METRICS_ENABLED"),
		Category: OperationsCategory,
	}
	MetricsAddrFlag = &cli.StringFlag{
		Name:     "metrics.addr",
		Usage:    "Metrics listening address",
		Value:    "127.0.0.1", // Warning: this used to default to 0.0.0.0
		EnvVars:  prefixEnvVars("METRICS_ADDR"),
		Category: OperationsCategory,
	}
	MetricsPortFlag = &cli.IntFlag{
		Name:     "metrics.port",
		Usage:    "Metrics listening port",
		Value:    7300,
		EnvVars:  prefixEnvVars("METRICS_PORT"),
		Category: OperationsCategory,
	}
	FetchWithdrawalRootFromState = &cli.BoolFlag{
		Name:     "fetch-withdrawal-root-from-state",
		Usage:    "Read withdrawal_storage_root (aka message passer storage root) from state trie (via execution layer) instead of the block header. Restores pre-Isthmus behavior, requires an archive EL client.",
		Required: false,
		EnvVars:  prefixEnvVars("FETCH_WITHDRAWAL_ROOT_FROM_STATE"),
		Category: OperationsCategory,
	}
)

var OperationsFlags = append(append([]cli.Flag{
	RPCListenAddr,
	RPCListenPort,
	RPCEnableAdmin,
	RPCAdminPersistence,
	MetricsEnabledFlag,
	MetricsAddrFlag,
	MetricsPortFlag,
	FetchWithdrawalRootFromState,
},
	// TODO: why does op-node redeclare these flags? We can import from the package?
	//oprpc.CLIFlags(EnvVarPrefix)...)
	//opmetrics.CLIFlags(EnvVarPrefix)...)
	oplog.CLIFlagsWithCategory(EnvVarPrefixOpnode, OperationsCategory)...),
	oppprof.CLIFlagsWithCategory(EnvVarPrefixOpnode, OperationsCategory)...)
