package pipeline

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

func DeploySuperchain(env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-superchain")

	if !shouldDeploySuperchain(intent, st) {
		lgr.Info("superchain deployment not needed")
		return nil
	}

	lgr.Info("deploying superchain")
	isOPCMv2 := false
	requiredProtocolVersion := rollup.OPStackSupport
	recommendedProtocolVersion := rollup.OPStackSupport
	if devFeatureBitmap, ok := intent.GlobalDeployOverrides["devFeatureBitmap"].(common.Hash); ok {
		opcmV2Flag := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000010000")
		if isDevFeatureEnabled(devFeatureBitmap, opcmV2Flag) {
			isOPCMv2 = true
			intent.SuperchainRoles.ProtocolVersionsOwner = common.Address{}
			requiredProtocolVersion = params.ProtocolVersion{}
			recommendedProtocolVersion = params.ProtocolVersion{}
		}
	}
	input := opcm.DeploySuperchainInput{
		SuperchainProxyAdminOwner:  intent.SuperchainRoles.SuperchainProxyAdminOwner,
		ProtocolVersionsOwner:      intent.SuperchainRoles.ProtocolVersionsOwner,
		Guardian:                   intent.SuperchainRoles.SuperchainGuardian,
		Paused:                     false,
		RequiredProtocolVersion:    requiredProtocolVersion,
		RecommendedProtocolVersion: recommendedProtocolVersion,
		IsOPCMv2:                   isOPCMv2,
	}

	var dso opcm.DeploySuperchainOutput
	var err error

	if env.UseForge {
		if env.ForgeClient == nil {
			return fmt.Errorf("Forge client is nil but UseForge is enabled")
		}
		if env.Context == nil {
			env.Context = context.Background()
		}
		if env.PrivateKey == "" {
			return fmt.Errorf("private key is required when UseForge is enabled")
		}
		lgr.Info("using Forge for DeploySuperchain")
		forgeCaller := opcm.NewDeploySuperchainForgeCaller(env.ForgeClient)
		forgeOpts := []string{
			"--rpc-url", env.L1RPCUrl,
			"--broadcast",
			"--private-key", env.PrivateKey,
		}
		dso, _, err = forgeCaller(env.Context, input, forgeOpts...)
		if err != nil {
			return fmt.Errorf("failed to deploy superchain with Forge: %w", err)
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
