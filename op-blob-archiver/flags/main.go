package flags

import (
	"fmt"
	"time"

	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

const EnvVarPrefix = "OP_ARCHIVER"

func prefixEnvVars(name string) []string {
	return opservice.PrefixEnvVar(EnvVarPrefix, name)
}

var (
	// Required flags
	L1EthRpcFlag = &cli.StringFlag{
		Name:    "l1-eth-rpc",
		Usage:   "HTTP provider URL for L1",
		EnvVars: prefixEnvVars("L1_ETH_RPC"),
	}
	L2EthRpcFlag = &cli.StringFlag{
		Name:    "l2-eth-rpc",
		Usage:   "HTTP provider URL for L2 execution engine",
		EnvVars: prefixEnvVars("L2_ETH_RPC"),
	}
	RollupRpcFlag = &cli.StringFlag{
		Name:    "rollup-rpc",
		Usage:   "HTTP provider URL for Rollup node",
		EnvVars: prefixEnvVars("ROLLUP_RPC"),
	}
	PollIntervalFlag = &cli.DurationFlag{
		Name:    "poll-interval",
		Usage:   "How frequently to poll L1 for new blocks to archive blobs from",
		Value:   6 * time.Second,
		EnvVars: prefixEnvVars("POLL_INTERVAL"),
	}
	// Optional flags
	S3BucketNameFlag = &cli.StringFlag{
		Name:    "s3-bucket-name",
		Usage:   "Name of S3 bucket to store blobs",
		EnvVars: prefixEnvVars("S3_BUCKET_NAME"),
	}
	S3RegionFlag = &cli.StringFlag{
		Name:    "s3-region",
		Usage:   "Region of S3 bucket to store blobs",
		EnvVars: prefixEnvVars("S3_REGION"),
	}
	BatchInboxAddressFlag = &cli.StringFlag{
		Name:    "batch-inbox-address",
		Usage:   "Address of the BatchInbox",
		EnvVars: prefixEnvVars("BATCH_INBOX_ADDRESS"),
	}
	// StoppedFlag = &cli.BoolFlag{
	// 	Name:    "stopped",
	// 	Usage:   "Initialize the archiver in a stopped state. The batcher can be started using the admin_startBatcher RPC",
	// 	EnvVars: prefixEnvVars("STOPPED"),
	// }
)

var requiredFlags = []cli.Flag{
	L1EthRpcFlag,
	L2EthRpcFlag,
	RollupRpcFlag,
	PollIntervalFlag,
}

var optionalFlags = []cli.Flag{
	// StoppedFlag,
	S3BucketNameFlag,
	S3RegionFlag,
	BatchInboxAddressFlag,
}

func init() {
	optionalFlags = append(optionalFlags, oprpc.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, oplog.CLIFlags(EnvVarPrefix)...)
	optionalFlags = append(optionalFlags, opmetrics.CLIFlags(EnvVarPrefix)...)

	Flags = append(requiredFlags, optionalFlags...)
}

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func CheckRequired(ctx *cli.Context) error {
	for _, f := range requiredFlags {
		if !ctx.IsSet(f.Names()[0]) {
			return fmt.Errorf("flag %s is required", f.Names()[0])
		}
	}
	return nil
}
