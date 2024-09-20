package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

func DeployImplementations(ctx context.Context, env *Env, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")

	if !shouldDeployImplementations(intent, st) {
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var artifactsFS foundry.StatDirFs
	var err error
	if intent.ContractArtifactsURL.Scheme == "file" {
		fs := os.DirFS(intent.ContractArtifactsURL.Path)
		artifactsFS = fs.(foundry.StatDirFs)
	} else {
		return fmt.Errorf("only file:// artifacts URLs are supported")
	}

	var dump *foundry.ForgeAllocs
	var dio opsm.DeployImplementationsOutput
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
				host.SetEnvVar("IMPL_SALT", st.Create2Salt.Hex()[2:])
				host.ImportState(st.SuperchainDeployment.StateDump)
				dio, err = opsm.DeployImplementations(
					host,
					opsm.DeployImplementationsInput{
						WithdrawalDelaySeconds:          big.NewInt(604800),
						MinProposalSizeBytes:            big.NewInt(126000),
						ChallengePeriodSeconds:          big.NewInt(86400),
						ProofMaturityDelaySeconds:       big.NewInt(604800),
						DisputeGameFinalityDelaySeconds: big.NewInt(302400),
						Release:                         "op-contracts/v1.6.0",
						SuperchainConfigProxy:           st.SuperchainDeployment.SuperchainConfigProxyAddress,
						ProtocolVersionsProxy:           st.SuperchainDeployment.ProtocolVersionsProxyAddress,
						SuperchainProxyAdmin:            st.SuperchainDeployment.ProxyAdminAddress,
						UseInterop:                      false,
					},
				)
				if err != nil {
					return fmt.Errorf("error deploying implementations: %w", err)
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
		return fmt.Errorf("error deploying implementations: %w", err)
	}

	st.ImplementationsDeployment = &state.ImplementationsDeployment{
		OpsmProxyAddress:                        dio.OpsmProxy,
		DelayedWETHImplAddress:                  dio.DelayedWETHImpl,
		OptimismPortalImplAddress:               dio.OptimismPortalImpl,
		PreimageOracleSingletonAddress:          dio.PreimageOracleSingleton,
		MipsSingletonAddress:                    dio.MipsSingleton,
		SystemConfigImplAddress:                 dio.SystemConfigImpl,
		L1CrossDomainMessengerImplAddress:       dio.L1CrossDomainMessengerImpl,
		L1ERC721BridgeImplAddress:               dio.L1ERC721BridgeImpl,
		L1StandardBridgeImplAddress:             dio.L1StandardBridgeImpl,
		OptimismMintableERC20FactoryImplAddress: dio.OptimismMintableERC20FactoryImpl,
		DisputeGameFactoryImplAddress:           dio.DisputeGameFactoryImpl,
		StateDump:                               dump,
	}
	if err := env.WriteState(st); err != nil {
		return err
	}

	return nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
