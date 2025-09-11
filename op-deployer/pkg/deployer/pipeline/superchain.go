package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

func DeploySuperchain(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-superchain")

	if !shouldDeploySuperchain(intent, st) {
		lgr.Info("superchain deployment not needed")
		return nil
	}

	lgr.Info("deploying superchain")

	dso, err := env.Scripts.DeploySuperchain.Run(
		opcm.DeploySuperchainInput{
			SuperchainProxyAdminOwner:  intent.SuperchainRoles.SuperchainProxyAdminOwner,
			Guardian:                   intent.SuperchainRoles.SuperchainGuardian,
			Paused:                     false,
			RequiredProtocolVersion:    rollup.OPStackSupport,
			RecommendedProtocolVersion: rollup.OPStackSupport,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to deploy superchain: %w", err)
	}

	st.SuperchainDeployment = &addresses.SuperchainContracts{
		SuperchainProxyAdminImpl: dso.SuperchainProxyAdmin,
		SuperchainConfigProxy:    dso.SuperchainConfigProxy,
		SuperchainConfigImpl:     dso.SuperchainConfigImpl,
	}
	st.SuperchainRoles = intent.SuperchainRoles

	return nil
}

func shouldDeploySuperchain(intent *state.Intent, st *state.State) bool {
	return st.SuperchainDeployment == nil
}
