package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
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
	if err := checkMTCannonFlags(ctx, cfg); err != nil {
		return nil, err
	}

	var mtPrestate common.Hash
	var mtPrestateURL *url.URL
	if ctx.IsSet(addMTCannonPrestateFlag.Name) {
		mtPrestate = common.HexToHash(ctx.String(addMTCannonPrestateFlag.Name))
		mtPrestateURL, err = url.Parse(ctx.String(addMTCannonPrestateURLFlag.Name))
		if err != nil {
			return nil, fmt.Errorf("invalid mt-cannon prestate url (%v): %w", ctx.String(addMTCannonPrestateFlag.Name), err)
		}
	}
	return runner.NewRunner(logger, cfg, mtPrestate, mtPrestateURL), nil
}

func checkMTCannonFlags(ctx *cli.Context, cfg *config.Config) error {
	if ctx.IsSet(addMTCannonPrestateFlag.Name) || ctx.IsSet(addMTCannonPrestateURLFlag.Name) {
		if ctx.IsSet(addMTCannonPrestateFlag.Name) != ctx.IsSet(addMTCannonPrestateURLFlag.Name) {
			return fmt.Errorf("both flag %v and %v must be set when running MT-Cannon traces", addMTCannonPrestateURLFlag.Name, addMTCannonPrestateFlag.Name)
		}
		if cfg.Cannon == (vm.Config{}) {
			return errors.New("required Cannon vm configuration for mt-cannon traces is missing")
		}
	}
	return nil
}

func runTraceFlags() []cli.Flag {
	return append(flags.Flags, addMTCannonPrestateFlag, addMTCannonPrestateURLFlag)
}

var RunTraceCommand = &cli.Command{
	Name:        "run-trace",
	Usage:       "Continuously runs the specified trace providers in a regular loop",
	Description: "Runs trace providers against real chain data to confirm compatibility",
	Action:      cliapp.LifecycleCmd(RunTrace),
	Flags:       runTraceFlags(),
}

var (
	addMTCannonPrestateFlag = &cli.StringFlag{
		Name:    "add-mt-cannon-prestate",
		Usage:   "Use this prestate to run MT-Cannon compatibility tests",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ADD_MT_CANNON_PRESTATE"),
	}
	addMTCannonPrestateURLFlag = &cli.StringFlag{
		Name:    "add-mt-cannon-prestate-url",
		Usage:   "Use this prestate URL to run MT-Cannon compatibility tests",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "ADD_MT_CANNON_PRESTATE_URL"),
	}
)
