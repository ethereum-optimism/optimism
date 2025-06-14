package service

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/flags"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type MainFn func(ctx context.Context, cfg *config.Config, logger log.Logger) (cliapp.Lifecycle, error)

// Main is the entrypoint into the op-node-v2.
// This method returns a cliapp.LifecycleAction, to create an op-service CLI-lifecycle-managed op-node-v2 with.
func Main(version string, fn MainFn) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		cfg, err := config.FromCLI(cliCtx, version)
		if err != nil {
			return nil, err
		}
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.Handler())

		opservice.ValidateEnvVars("OP_NODE", flags.Flags, l)

		l.Info("Initializing op-node V2")
		return fn(cliCtx.Context, cfg, l)
	}
}
