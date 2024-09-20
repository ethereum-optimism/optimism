package inspect

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

const (
	OutfileFlagName = "outfile"
)

var (
	FlagOutfile = &cli.StringFlag{
		Name:  OutfileFlagName,
		Usage: "output file. set to - to use stdout",
		Value: "-",
	}
)

var Flags = []cli.Flag{
	deployer.WorkdirFlag,
	FlagOutfile,
}

var Commands = []*cli.Command{
	{
		Name:      "genesis",
		Usage:     "outputs the genesis for an L2 chain",
		Args:      true,
		ArgsUsage: "<chain-id>",
		Action:    GenesisCLI,
		Flags:     Flags,
	},
	{
		Name:      "rollup",
		Usage:     "outputs the rollup config for an L2 chain",
		Args:      true,
		ArgsUsage: "<chain-id>",
		Action:    RollupCLI,
		Flags:     Flags,
	},
}

type cliConfig struct {
	Workdir string
	Outfile string
	ChainID common.Hash
}

func readConfig(cliCtx *cli.Context) (cliConfig, error) {
	var cfg cliConfig

	outfile := cliCtx.String(OutfileFlagName)
	if outfile == "" {
		return cfg, fmt.Errorf("outfile flag is required")
	}

	workdir := cliCtx.String(deployer.WorkdirFlagName)
	if workdir == "" {
		return cfg, fmt.Errorf("workdir flag is required")
	}

	chainIDStr := cliCtx.Args().First()
	if chainIDStr == "" {
		return cfg, fmt.Errorf("chain-id argument is required")
	}

	chainID, err := chainIDStrToHash(chainIDStr)
	if err != nil {
		return cfg, fmt.Errorf("failed to parse chain ID: %w", err)
	}

	return cliConfig{
		Workdir: cliCtx.String(deployer.WorkdirFlagName),
		Outfile: cliCtx.String(OutfileFlagName),
		ChainID: chainID,
	}, nil
}

type inspectState struct {
	GlobalState *state.State
	ChainIntent *state.ChainIntent
	ChainState  *state.ChainState
}

func bootstrapState(cfg cliConfig) (*inspectState, error) {
	env := &pipeline.Env{Workdir: cfg.Workdir}
	globalState, err := env.ReadState()
	if err != nil {
		return nil, fmt.Errorf("failed to read intent: %w", err)
	}

	if globalState.AppliedIntent == nil {
		return nil, fmt.Errorf("chain state is not applied - run op-deployer apply")
	}

	chainIntent, err := globalState.AppliedIntent.Chain(cfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied chain intent: %w", err)
	}

	chainState, err := globalState.Chain(cfg.ChainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID %s: %w", cfg.ChainID.String(), err)
	}

	return &inspectState{
		GlobalState: globalState,
		ChainIntent: chainIntent,
		ChainState:  chainState,
	}, nil
}
