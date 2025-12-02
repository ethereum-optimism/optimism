package pipeline

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

type PreStateBuilder interface {
	BuildPrestate(ctx context.Context, opts ...prestate.PrestateBuilderOption) (prestate.PrestateManifest, error)
}

func GeneratePreState(ctx context.Context, pEnv *Env, globalIntent *state.Intent, st *state.State, preStateBuilder PreStateBuilder) error {
	lgr := pEnv.Logger.New("stage", "generate-pre-state")

	if preStateBuilder == nil {
		lgr.Debug("preStateBuilder not found, skipping prestate generation")
		return nil
	}
	lgr.Info("preStateBuilder found, proceeding with prestate generation")

	prestateBuilderOpts := []prestate.PrestateBuilderOption{}
	generateDepSet := false
	for _, chain := range globalIntent.Chains {
		genesis, rollup, err := RenderGenesisAndRollup(st, chain.ID, globalIntent)
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

		if genesis.Config.InteropTime != nil {
			generateDepSet = true
		}

		prestateBuilderOpts = append(prestateBuilderOpts, prestate.WithChainConfig(
			chain.ID.Big().String(),
			bytes.NewReader(rollupJSON),
			bytes.NewReader(genesisJSON),
		))
	}

	if generateDepSet {
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
