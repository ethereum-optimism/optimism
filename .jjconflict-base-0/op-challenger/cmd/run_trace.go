package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/flags"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/runner"
	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

var (
	ErrInvalidPrestateHash = errors.New("invalid prestate hash")
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
	runConfigs, err := parseRunArgs(ctx.StringSlice(RunTraceRunFlag.Name))
	if err != nil {
		return nil, err
	}
	if len(runConfigs) == 0 {
		// Default to running on-chain version of each enabled game type
		for _, gameType := range cfg.GameTypes {
			runConfigs = append(runConfigs, runner.RunConfig{GameType: gameType})
		}
	}
	vmTimeout := ctx.Duration(VMTimeoutFlag.Name)
	return runner.NewRunner(logger, cfg, runConfigs, vmTimeout), nil
}

func runTraceFlags() []cli.Flag {
	return append(flags.Flags, RunTraceRunFlag, VMTimeoutFlag)
}

var RunTraceCommand = &cli.Command{
	Name:        "run-trace",
	Usage:       "Continuously runs the specified trace providers in a regular loop",
	Description: "Runs trace providers against real chain data to confirm compatibility",
	Action:      cliapp.LifecycleCmd(RunTrace),
	Flags:       runTraceFlags(),
}

const DefaultVMTimeout = 3 * time.Hour

var (
	RunTraceRunFlag = &cli.StringSliceFlag{
		Name: "run",
		Usage: "Specify a trace to run. Format is gameType/name/prestateHash where " +
			"gameType is the game type to use with the prestate (e.g cannon or cannon-kona), " +
			"name is an arbitrary name for the prestate to use when reporting metrics and" +
			"prestateHash is the hex encoded absolute prestate commitment to use. " +
			"If name is omitted the game type name is used." +
			"If the prestateHash is omitted, the absolute prestate hash used for new games on-chain.",
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "RUN"),
	}
	VMTimeoutFlag = &cli.DurationFlag{
		Name:    "vm-timeout",
		Usage:   fmt.Sprintf("Maximum duration for VM execution per run. Default is %s. Set to 0 to disable timeout.", DefaultVMTimeout),
		EnvVars: opservice.PrefixEnvVar(flags.EnvVarPrefix, "VM_TIMEOUT"),
		Value:   DefaultVMTimeout,
	}
)

func parseRunArgs(args []string) ([]runner.RunConfig, error) {
	cfgs := make([]runner.RunConfig, len(args))
	for i, arg := range args {
		cfg, err := parseRunArg(arg)
		if err != nil {
			return nil, err
		}
		cfgs[i] = cfg
	}
	return cfgs, nil
}

func parseRunArg(arg string) (runner.RunConfig, error) {
	cfg := runner.RunConfig{}
	opts := strings.SplitN(arg, "/", 3)
	if len(opts) == 0 {
		return runner.RunConfig{}, fmt.Errorf("invalid run config %q", arg)
	}
	gameType, err := gameTypes.SupportedGameTypeFromString(opts[0])
	if err != nil {
		return runner.RunConfig{}, fmt.Errorf("%w %q for run config %q", err, opts[0], arg)
	}
	cfg.GameType = gameType
	if len(opts) > 1 {
		cfg.Name = opts[1]
	} else {
		cfg.Name = cfg.GameType.String()
	}
	if len(opts) > 2 {
		if strings.HasPrefix(opts[2], "0x") {
			cfg.Prestate = common.HexToHash(opts[2])
			if cfg.Prestate == (common.Hash{}) {
				return runner.RunConfig{}, fmt.Errorf("%w %q for run config %q", ErrInvalidPrestateHash, opts[2], arg)
			}
		} else {
			cfg.PrestateFilename = opts[2]
		}
	}
	return cfg, nil
}
