package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func PreinstallL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "preinstall-l1-dev-genesis")
	lgr.Info("Adding preinstalls to L1 dev genesis")

	if err := opcm.InsertL1Preinstalls(env.Forge); err != nil {
		return fmt.Errorf("failed to add preinstalls to L1 dev state: %w", err)
	}
	// The deployer keeps its account only if the intent prefunds it. When this stage runs after the
	// deployments, as in ApplyPipeline, the kept nonce stops the deployer's next CREATE from
	// colliding with a contract that it deployed into the genesis.
	if !prefunds(intent, env.Deployer) {
		env.Forge.Genesis.Wipe(env.Deployer)
	}
	return nil
}
