package flags

import (
	"os"

	"github.com/urfave/cli/v2"
)

var directory, _ = os.Getwd()

// CommonFlags are flags that are common to all commands.
var CommonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "monorepo-dir",
		Value:   directory,
		Usage:   "Directoy of the monorepo",
		EnvVars: []string{"MONOREPO_DIR"},
	},
	&cli.StringFlag{
		Name:    "l1-rpc-url",
		Value:   "127.0.0.1:8545",
		Usage:   "L1 RPC URL",
		EnvVars: []string{"L1_RPC_URL"},
	},
	&cli.StringFlag{
		Name:    "l2-rpc-url",
		Value:   "127.0.0.1:9545",
		Usage:   "L2 RPC URL",
		EnvVars: []string{"L2_RPC_URL"},
	},
}
