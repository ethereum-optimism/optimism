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

	input := opcm.DeploySuperchainInput{
		SuperchainProxyAdminOwner:  intent.SuperchainRoles.SuperchainProxyAdminOwner,
		ProtocolVersionsOwner:      intent.SuperchainRoles.ProtocolVersionsOwner,
		Guardian:                   intent.SuperchainRoles.SuperchainGuardian,
		Paused:                     false,
		RequiredProtocolVersion:    rollup.OPStackSupport,
		RecommendedProtocolVersion: rollup.OPStackSupport,
	}

	var dso opcm.DeploySuperchainOutput
	var err error

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
		SuperchainProxyAdminImpl: dso.SuperchainProxyAdmin,
		SuperchainConfigProxy:    dso.SuperchainConfigProxy,
		SuperchainConfigImpl:     dso.SuperchainConfigImpl,
		ProtocolVersionsProxy:    dso.ProtocolVersionsProxy,
		ProtocolVersionsImpl:     dso.ProtocolVersionsImpl,
	}
	st.SuperchainRoles = intent.SuperchainRoles

	return nil
}

func shouldDeploySuperchain(intent *state.Intent, st *state.State) bool {
	return st.SuperchainDeployment == nil
}
