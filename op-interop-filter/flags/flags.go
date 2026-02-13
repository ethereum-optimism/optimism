package flags

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
)

const EnvVarPrefix = "OP_INTEROP_FILTER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	L2RPCsFlag = &cli.StringSliceFlag{
		Name:    "l2-rpcs",
		Usage:   "L2 RPC endpoints to connect to (chain ID is queried from each endpoint and matched to rollup configs)",
		EnvVars: prefixEnvVars("L2_RPCS"),
	}
	NetworksFlag = &cli.StringSliceFlag{
		Name:    "networks",
		Usage:   fmt.Sprintf("Predefined networks to load rollup configs from. Available: %s", strings.Join(chaincfg.AvailableNetworks(), ", ")),
		EnvVars: prefixEnvVars("NETWORKS"),
	}
	RollupConfigsFlag = &cli.StringSliceFlag{
		Name:    "rollup-configs",
		Usage:   "Paths to custom rollup config JSON files (for dev/test chains not in superchain registry)",
		EnvVars: prefixEnvVars("ROLLUP_CONFIGS"),
	}
	DataDirFlag = &cli.StringFlag{
		Name:    "data-dir",
		Usage:   "Directory for LogsDB storage. If empty, uses a temporary directory",
		EnvVars: prefixEnvVars("DATA_DIR"),
		Value:   "",
	}
	BackfillDurationFlag = &cli.StringFlag{
		Name:    "backfill-duration",
		Usage:   "Duration to backfill on startup (e.g., 24h, 30m, 1h30m)",
		EnvVars: prefixEnvVars("BACKFILL_DURATION"),
		Value:   "24h",
	}
	MessageExpiryWindowFlag = &cli.StringFlag{
		Name:    "message-expiry-window",
		Usage:   "Message expiry window duration (e.g., 168h for 7 days). Messages older than this are considered expired.",
		EnvVars: prefixEnvVars("MESSAGE_EXPIRY_WINDOW"),
		Value:   "168h", // 7 days default, matching op-supervisor
	}
	JWTSecretFlag = &cli.StringFlag{
		Name: "admin.jwt-secret",
		Usage: "Path to JWT secret key for admin RPC authentication. " +
			"Keys are 32 bytes, hex encoded in a file. " +
			"A new key will be generated if the file is missing. " +
			"Required when admin.rpc.addr is set.",
		EnvVars:   prefixEnvVars("ADMIN_JWT_SECRET"),
		Value:     "",
		TakesFile: true,
	}
	AdminRPCAddrFlag = &cli.StringFlag{
		Name:    "admin.rpc.addr",
		Usage:   "Address to bind admin RPC server. If empty, admin RPC is disabled.",
		EnvVars: prefixEnvVars("ADMIN_RPC_ADDR"),
		Value:   "",
	}
	AdminRPCPortFlag = &cli.IntFlag{
		Name:    "admin.rpc.port",
		Usage:   "Port to bind admin RPC server.",
		EnvVars: prefixEnvVars("ADMIN_RPC_PORT"),
		Value:   8546,
	}
	RPCAddrFlag = &cli.StringFlag{
		Name:    "rpc.addr",
		Usage:   "RPC listening address",
		EnvVars: prefixEnvVars("RPC_ADDR"),
		Value:   "0.0.0.0",
	}
	RPCPortFlag = &cli.IntFlag{
		Name:    "rpc.port",
		Usage:   "RPC listening port",
		EnvVars: prefixEnvVars("RPC_PORT"),
		Value:   8545,
	}
	PollIntervalFlag = &cli.StringFlag{
		Name:    "poll-interval",
		Usage:   "Interval for polling new blocks from L2 RPCs (e.g., 2s, 500ms)",
		EnvVars: prefixEnvVars("POLL_INTERVAL"),
		Value:   "2s",
	}
	ValidationIntervalFlag = &cli.StringFlag{
		Name:    "validation-interval",
		Usage:   "Interval for cross-chain validation loop (e.g., 500ms, 1s)",
		EnvVars: prefixEnvVars("VALIDATION_INTERVAL"),
		Value:   "500ms",
	}
	DangerouslyEnablePassthroughFlag = &cli.BoolFlag{
		Name:    "dangerously-enable-passthrough",
		Usage:   "Allow all transactions through without interop filtering. DANGEROUS: disables all executing message validation.",
		EnvVars: prefixEnvVars("DANGEROUSLY_ENABLE_PASSTHROUGH"),
	}
)

var requiredFlags = []cli.Flag{
	L2RPCsFlag,
}

var optionalFlags = []cli.Flag{
	NetworksFlag,
	RollupConfigsFlag,
	DataDirFlag,
	BackfillDurationFlag,
	MessageExpiryWindowFlag,
	JWTSecretFlag,
	AdminRPCAddrFlag,
	AdminRPCPortFlag,
	RPCAddrFlag,
	RPCPortFlag,
	PollIntervalFlag,
	ValidationIntervalFlag,
	DangerouslyEnablePassthroughFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oppprof.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		name := f.Names()[0]
		if !ctx.IsSet(name) {
			return fmt.Errorf("flag %s is required", name)
		}
	}

	// At least one of --networks or --rollup-configs must be provided
	if !ctx.IsSet(NetworksFlag.Name) && !ctx.IsSet(RollupConfigsFlag.Name) {
		return fmt.Errorf("at least one of --%s or --%s is required", NetworksFlag.Name, RollupConfigsFlag.Name)
	}

	return nil
}
