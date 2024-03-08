package server

import (
	"context"

	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
)

func Main(version string) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		cfg := ReadCLIConfig(cliCtx, version)
		return FromCLIConfig(cfg)
	}
}
