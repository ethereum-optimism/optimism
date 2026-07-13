package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func PreinstallL1DevGenesis(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "preinstall-l1-dev-genesis")
	lgr.Info("Adding preinstalls to L1 dev genesis")

	if err := env.InsertPreinstallsL1(); err != nil {
		return fmt.Errorf("failed to add preinstalls to L1 dev state: %w", err)
	}
	if err := env.WipeL1(env.Deployer); err != nil {
		return fmt.Errorf("failed to wipe deployer from L1 dev state: %w", err)
	}

	return nil
}
