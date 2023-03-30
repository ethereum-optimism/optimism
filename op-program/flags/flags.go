package flags

import (
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/urfave/cli"
)

const envVarPrefix = "OP_PROGRAM"

// Flags contains the list of configuration options available to the binary.
var Flags []cli.Flag

func init() {
	Flags = oplog.CLIFlags(envVarPrefix)
}
