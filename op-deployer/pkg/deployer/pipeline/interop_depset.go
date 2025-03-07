package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/devnet-sdk/proofs/prestate"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func GenerateInteropDepset(ctx context.Context, pEnv *Env, globalIntent *state.Intent, st *state.State) error {
	lgr := pEnv.Logger.New("stage", "generate-interop-depset")

	if !globalIntent.UseInterop {
		lgr.Warn("interop not enabled - skipping interop depset generation")
		return nil
	}

	var chains []string
	for _, chain := range globalIntent.Chains {
		chains = append(chains, chain.ID.Big().String())
	}

	lgr.Info("rendering the interop dependency set...")
	depSet := prestate.RenderInteropDepSet(chains)
	st.InteropDepSet = &depSet

	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
