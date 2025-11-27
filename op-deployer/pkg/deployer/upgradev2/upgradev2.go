package upgradev2

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// UpgradeV2CLI returns a CLI handler for the upgradev2 command
func UpgradeV2CLI() func(cliCtx *cli.Context) error {
	return func(cliCtx *cli.Context) error {
		fmt.Println("hello world")
		return nil
	}
}
