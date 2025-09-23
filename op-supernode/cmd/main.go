package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/ctxinterrupt"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/urfave/cli/v2"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/flags"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
	"github.com/ethereum/go-ethereum/log"
)

var (
	Version   = "v0.0.0"
	GitCommit = ""
	GitDate   = ""
)

func main() {
	oplog.SetupDefaults()

	// Extract and store vn.* flags before urfave/cli processes args
	// This allows us to handle dynamic chain-specific flags
	vnFlags, filteredArgs := flags.ExtractVNFlags(os.Args)

	app := cli.NewApp()
	app.Flags = cliapp.ProtectFlags(flags.Flags)
	app.Version = opservice.FormatVersion(Version, GitCommit, GitDate, "")
	app.Name = "op-supernode"
	app.Usage = "Supernode"
	app.Description = "Supernode service that starts up, prints, and exits"
	app.Action = cliapp.LifecycleCmd(func(cliCtx *cli.Context, close context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		if err := flags.CheckRequired(cliCtx); err != nil {
			return nil, err
		}
		cfg := config.NewConfig(cliCtx, vnFlags)
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid CLI flags: %w", err)
		}
		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		vnCfgs, err := config.VirtualNodeConfigs(cliCtx, cfg.VNFlags, l)
		if err != nil {
			return nil, fmt.Errorf("failed to create virtual node configs: %w", err)
		}

		oplog.SetGlobalLogHandler(l.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)

		l.Info("configured sample", "sample", cfg.Sample)
		return supernode.New(l, Version, close, cfg, vnCfgs), nil
	})

	ctx := ctxinterrupt.WithSignalWaiterMain(context.Background())
	if err := app.RunContext(ctx, filteredArgs); err != nil {
		log.Crit("Application failed", "message", err)
	}
}
