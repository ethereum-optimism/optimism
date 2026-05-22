package pipeline

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
)

func DeploySuperchain(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-superchain")

	if !shouldDeploySuperchain(intent, st) {
		lgr.Info("superchain deployment not needed")
		return nil
	}

	lgr.Info("deploying superchain")

	input, err := evaluateLegacyInputMapping(env, "deploy-superchain.input.yaml", opcm.StaticInputSources{
		"intent": intent,
		"state":  st,
	})
	if err != nil {
		return fmt.Errorf("failed to build DeploySuperchain input: %w", err)
	}

	var dso opcm.DeploySuperchainOutput

	if env.UseForge {
		lgr.Info("using Forge for DeploySuperchain")
		forgeEnv := &opcm.ForgeEnv{
			Client:     env.ForgeClient,
			Context:    env.Context,
			L1RPCUrl:   env.L1RPCUrl,
			PrivateKey: env.PrivateKey,
		}
		dso, err = opcm.DeploySuperchainViaForge(forgeEnv, input)
		if err != nil {
			return err
		}
	} else {
		dso, err = env.Scripts.DeploySuperchain.Run(input)
		if err != nil {
			return fmt.Errorf("failed to deploy superchain: %w", err)
		}
	}

	st.SuperchainDeployment = &addresses.SuperchainContracts{
		SuperchainProxyAdminImpl: dso.Address("superchainProxyAdmin"),
		SuperchainConfigProxy:    dso.Address("superchainConfigProxy"),
		SuperchainConfigImpl:     dso.Address("superchainConfigImpl"),
	}
	st.SuperchainRoles = intent.SuperchainRoles

	return nil
}

func shouldDeploySuperchain(intent *state.Intent, st *state.State) bool {
	return st.SuperchainDeployment == nil
}
