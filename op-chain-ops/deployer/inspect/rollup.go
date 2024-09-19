package inspect

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
)

func RollupCLI(cliCtx *cli.Context) error {
	cfg, err := readConfig(cliCtx)
	if err != nil {
		return err
	}

	env := &pipeline.Env{Workdir: cfg.Workdir}
	globalState, err := env.ReadState()
	if err != nil {
		return fmt.Errorf("failed to read intent: %w", err)
	}

	rollupConfig, err := Rollup(globalState, cfg.ChainID)
	if err != nil {
		return fmt.Errorf("failed to generate rollup config: %w", err)
	}

	if err := jsonutil.WriteJSON(rollupConfig, ioutil.ToStdOutOrFileOrNoop(cfg.Outfile, 0o666)); err != nil {
		return fmt.Errorf("failed to write rollup config: %w", err)
	}

	return nil
}

func Rollup(globalState *state.State, chainID common.Hash) (*rollup.Config, error) {
	if globalState.AppliedIntent == nil {
		return nil, fmt.Errorf("chain state is not applied - run op-deployer apply")
	}

	chainIntent, err := globalState.AppliedIntent.Chain(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied chain intent: %w", err)
	}

	chainState, err := globalState.Chain(chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID %s: %w", chainID.String(), err)
	}

	l2Allocs, err := chainState.UnmarshalGenesis()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal genesis: %w", err)
	}

	config, err := state.CombineDeployConfig(
		globalState.AppliedIntent,
		chainIntent,
		globalState,
		chainState,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to combine L2 init config: %w", err)
	}

	l2GenesisBuilt, err := genesis.BuildL2Genesis(&config, l2Allocs, chainState.StartBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to build L2 genesis: %w", err)
	}
	l2GenesisBlock := l2GenesisBuilt.ToBlock()

	rollupConfig, err := config.RollupConfig(
		chainState.StartBlock,
		l2GenesisBlock.Hash(),
		l2GenesisBlock.Number().Uint64(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build rollup config: %w", err)
	}

	if err := rollupConfig.Check(); err != nil {
		return nil, fmt.Errorf("generated rollup config does not pass validation: %w", err)
	}

	return rollupConfig, nil
}
