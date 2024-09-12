package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

const DefaultContractsBedrockRepo = "us-docker.pkg.dev/oplabs-tools-artifacts/images/contracts-bedrock"

func DeploySuperchain(ctx context.Context, env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-superchain")

	if !shouldDeploySuperchain(intent, st) {
		lgr.Info("superchain deployment not needed")
		return nil
	}

	lgr.Info("deploying superchain")

	var artifactsFS foundry.StatDirFs
	var err error
	if intent.ContractArtifactsURL.Scheme == "file" {
		fs := os.DirFS(intent.ContractArtifactsURL.Path)
		artifactsFS = fs.(foundry.StatDirFs)
	} else {
		return fmt.Errorf("only file:// artifacts URLs are supported")
	}

	dso, err := opsm.DeploySuperchainForge(
		ctx,
		opsm.DeploySuperchainOpts{
			Input: opsm.DeploySuperchainInput{
				ProxyAdminOwner:            intent.SuperchainRoles.ProxyAdminOwner,
				ProtocolVersionsOwner:      intent.SuperchainRoles.ProtocolVersionsOwner,
				Guardian:                   intent.SuperchainRoles.Guardian,
				Paused:                     false,
				RequiredProtocolVersion:    rollup.OPStackSupport,
				RecommendedProtocolVersion: rollup.OPStackSupport,
			},
			ArtifactsFS: artifactsFS,
			ChainID:     big.NewInt(int64(intent.L1ChainID)),
			Client:      env.L1Client,
			Signer:      env.Signer,
			Deployer:    env.Deployer,
			Logger:      lgr,
		},
	)
	if err != nil {
		return fmt.Errorf("error deploying superchain: %w", err)
	}

	st.SuperchainDeployment = &state.SuperchainDeployment{
		ProxyAdminAddress:            dso.SuperchainProxyAdmin,
		SuperchainConfigProxyAddress: dso.SuperchainConfigProxy,
		SuperchainConfigImplAddress:  dso.SuperchainConfigImpl,
		ProtocolVersionsProxyAddress: dso.ProtocolVersionsProxy,
		ProtocolVersionsImplAddress:  dso.ProtocolVersionsImpl,
	}

	if err := env.WriteState(st); err != nil {
		return err
	}

	return nil
}

func shouldDeploySuperchain(intent *state.Intent, st *state.State) bool {
	if st.AppliedIntent == nil {
		return true
	}

	if st.SuperchainDeployment == nil {
		return true
	}

	return false
}
