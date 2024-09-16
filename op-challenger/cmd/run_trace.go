package main

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/runner"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum/go-ethereum/common"
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
	if ctx.IsSet(addMTCannonPrestate.Name) && cfg.CannonAbsolutePreStateBaseURL == nil {
		return nil, fmt.Errorf("flag %v is required when using %v", flags.CannonPreStateFlag.Name, addMTCannonPrestate.Name)
	}
	var mtPrestate common.Hash
	if ctx.IsSet(addMTCannonPrestate.Name) {
		mtPrestate = common.HexToHash(ctx.String(addMTCannonPrestate.Name))
	}
	return runner.NewRunner(logger, cfg, mtPrestate), nil
}

func runTraceFlags() []cli.Flag {
	return append(flags.Flags, addMTCannonPrestate)
}

var RunTraceCommand = &cli.Command{
	Name:        "run-trace",
	Usage:       "Continuously runs the specified trace providers in a regular loop",
	Description: "Runs trace providers against real chain data to confirm compatibility",
	Action:      cliapp.LifecycleCmd(RunTrace),
	Flags:       runTraceFlags(),
}

var addMTCannonPrestate = &cli.StringFlag{
	Name:    "add-mt-cannon-prestate",
	Usage:   "After running Cannon traces, additionally use this prestate to run MT-Cannon",
	EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ADD_MT_CANNON_PRESTATE"),
}
