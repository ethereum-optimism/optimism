package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func GenerateInteropDepset(_ context.Context, pEnv *Env, globalIntent *state.Intent, st *state.State) error {
	lgr := pEnv.Logger.New("stage", "generate-interop-depset")

	lgr.Info("creating interop dependency set...")
	deps := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for i, chain := range globalIntent.Chains {
		id := eth.ChainIDFromBytes32(chain.ID)
		_, config, err := calculateL2GenesisOverrides(globalIntent, chain)
		if err != nil {
			return fmt.Errorf("failed to calculate L2 genesis overrides for chain %v: %w", chain.ID, err)
		}
		if config.L2GenesisInteropTimeOffset == nil {
			// Don't add chains to the dep set if they don't have interop scheduled.
			continue
		}

		deps[id] = &depset.StaticConfigDependency{ChainIndex: types.ChainIndex(i)}
	}

	if len(deps) == 0 {
		lgr.Info("No interop chains found, skipping dependency set generation")
		return nil
	}
	interopDepSet, err := depset.NewStaticConfigDependencySet(deps)
	if err != nil {
		return fmt.Errorf("failed to create interop dependency set: %w", err)
	}
	st.InteropDepSet = interopDepSet

	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
