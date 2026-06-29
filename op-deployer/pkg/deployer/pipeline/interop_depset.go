package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// BuildInteropDepSet builds the interop dependency set for the given chains. It is
// shared between the legacy apply pipeline and the prepare flow so both produce the
// same depset.
func BuildInteropDepSet(chains []*state.ChainIntent) (*depset.StaticConfigDependencySet, error) {
	deps := make(map[eth.ChainID]*depset.StaticConfigDependency)
	for _, chain := range chains {
		id := eth.ChainIDFromBytes32(chain.ID)
		deps[id] = &depset.StaticConfigDependency{}
	}
	return depset.NewStaticConfigDependencySet(deps)
}

func GenerateInteropDepset(_ context.Context, pEnv *Env, globalIntent *state.Intent, st *state.State) error {
	lgr := pEnv.Logger.New("stage", "generate-interop-depset")

	lgr.Info("Creating interop dependency set...")
	interopDepSet, err := BuildInteropDepSet(globalIntent.Chains)
	if err != nil {
		return fmt.Errorf("failed to create interop dependency set: %w", err)
	}
	st.InteropDepSet = interopDepSet

	if err := pEnv.StateWriter.WriteState(st); err != nil {
		return fmt.Errorf("failed to write state: %w", err)
	}

	return nil
}
