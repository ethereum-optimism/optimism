package main

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/runner"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/urfave/cli/v2"
)

func RunTrace(ctx *cli.Context, _ context.CancelCauseFunc) (cliapp.Lifecycle, error) {

	logger, err := setupLogging(ctx)
	if err != nil {
		return nil, err
	}
	logger.Info("Starting trace runner", "version", VersionWithMeta)

	cfg, err := flags.NewConfigFromCLI(ctx, logger)
	if err != nil {
		return nil, err
	}
	if err := cfg.Check(); err != nil {
		return nil, err
	}
	return runner.NewRunner(logger, cfg), nil
}

func runTraceFlags() []cli.Flag {
	return flags.Flags
}

var RunTraceCommand = &cli.Command{
	Name:        "run-trace",
	Usage:       "Continuously runs the specified trace providers in a regular loop",
	Description: "Runs trace providers against real chain data to confirm compatibility",
	Action:      cliapp.LifecycleCmd(RunTrace),
	Flags:       runTraceFlags(),
}
