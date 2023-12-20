package flags

import (
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-node/flags"
	opflags "github.com/ethereum-optimism/optimism/op-service/flags"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
)

const envVarPrefix = "OP_BOOTNODE"

var Flags = []cli.Flag{
	opflags.CLINetworkFlag(envVarPrefix),
	opflags.CLIRollupConfigFlag(envVarPrefix),
}

func init() {
	Flags = append(Flags, flags.P2PFlags(envVarPrefix)...)
	Flags = append(Flags, opmetrics.CLIFlags(envVarPrefix)...)
	Flags = append(Flags, oplog.CLIFlags(envVarPrefix)...)
}
