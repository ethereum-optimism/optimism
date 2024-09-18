package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

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

	var dump *foundry.ForgeAllocs
	var dso opsm.DeploySuperchainOutput
	err = CallScriptBroadcast(
		ctx,
		CallScriptBroadcastOpts{
			L1ChainID:   big.NewInt(int64(intent.L1ChainID)),
			Logger:      lgr,
			ArtifactsFS: artifactsFS,
			Deployer:    env.Deployer,
			Signer:      env.Signer,
			Client:      env.L1Client,
			Broadcaster: KeyedBroadcaster,
			Handler: func(host *script.Host) error {
				dso, err = opsm.DeploySuperchain(
					host,
					opsm.DeploySuperchainInput{
						ProxyAdminOwner:            intent.SuperchainRoles.ProxyAdminOwner,
						ProtocolVersionsOwner:      intent.SuperchainRoles.ProtocolVersionsOwner,
						Guardian:                   intent.SuperchainRoles.Guardian,
						Paused:                     false,
						RequiredProtocolVersion:    rollup.OPStackSupport,
						RecommendedProtocolVersion: rollup.OPStackSupport,
					},
				)
				if err != nil {
					return fmt.Errorf("failed to deploy superchain: %w", err)
				}
				dump, err = host.StateDump()
				if err != nil {
					return fmt.Errorf("error dumping state: %w", err)
				}
				return nil
			},
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
		StateDump:                    dump,
	}
	if err := env.WriteState(st); err != nil {
		return err
	}

	return nil
}

func shouldDeploySuperchain(intent *state.Intent, st *state.State) bool {
	return st.SuperchainDeployment == nil
}
