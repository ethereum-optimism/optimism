package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-node/rollup"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// different from inspect.GenesisAndRollup()
// - breaks the import cycle
// - doesn't really on globalState.AppliedIntent
// - Though pretty anti-DRY
func GenesisAndRollup(globalState *state.State, intermediateIntent *state.Intent, chainID common.Hash) (*core.Genesis, *rollup.Config, error) {
	chainIntent, err := intermediateIntent.Chain(chainID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get applied chain intent: %w", err)
	}

	chainState, err := globalState.Chain(chainID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain ID %s: %w", chainID.String(), err)
	}

	l2Allocs := chainState.Allocs.Data
	config, err := state.CombineDeployConfig(
		intermediateIntent,
		chainIntent,
		globalState,
		chainState,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to combine L2 init config: %w", err)
	}

	l2GenesisBuilt, err := genesis.BuildL2Genesis(&config, l2Allocs, chainState.StartBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build L2 genesis: %w", err)
	}
	l2GenesisBlock := l2GenesisBuilt.ToBlock()

	rollupConfig, err := config.RollupConfig(
		chainState.StartBlock,
		l2GenesisBlock.Hash(),
		l2GenesisBlock.Number().Uint64(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build rollup config: %w", err)
	}

	if err := rollupConfig.Check(); err != nil {
		return nil, nil, fmt.Errorf("generated rollup config does not pass validation: %w", err)
	}

	return l2GenesisBuilt, rollupConfig, nil
}

func GeneratePreState(ctx context.Context, pEnv *Env, intent *state.Intent, st *state.State, preStateBuilder *prestate.PrestateBuilderClient) error {
	lgr := pEnv.Logger.New("stage", "generate-pre-state")

	prestateBuilderOpts := []prestate.PrestateBuilderOption{}
	for _, chain := range intent.Chains {
		genesis, rollup, err := GenesisAndRollup(st, intent, chain.ID)
		if err != nil {
			return fmt.Errorf("failed to get genesis and rollup for chain %s: %w", chain.ID.Hex(), err)
		}

		rollupJSON, err := json.Marshal(rollup)
		if err != nil {
			return fmt.Errorf("failed to marshal rollup config: %w", err)
		}

		genesisJSON, err := json.Marshal(genesis)
		if err != nil {
			return fmt.Errorf("failed to marshal genesis config: %w", err)
		}

		prestateBuilderOpts = append(prestateBuilderOpts, prestate.WithChainConfig(
			chain.ID.Big().String(),
			bytes.NewReader(rollupJSON),
			bytes.NewReader(genesisJSON),
		))
	}

	if intent.UseInterop {
		lgr.Info("adding the interop deployment set option to the prestate builder")
		prestateBuilderOpts = append(prestateBuilderOpts, prestate.WithGeneratedInteropDepSet())
	}

	lgr.Info("building the prestate...")
	manifest, err := preStateBuilder.BuildPrestate(ctx, prestateBuilderOpts...)
	if err != nil {
		return fmt.Errorf("failed to build prestate: %w", err)
	}

	lgr.Info("prestate built successfully")
	st.PrestateManifest = &manifest

	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
