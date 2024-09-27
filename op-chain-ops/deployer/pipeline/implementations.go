package pipeline

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

func DeployImplementations(ctx context.Context, env *Env, artifactsFS foundry.StatDirFs, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")

	if !shouldDeployImplementations(intent, st) {
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var standardVersionsTOML string
	var err error
	if strings.HasPrefix(intent.ContractsRelease, "op-contracts") {
		standardVersionsTOML, err = opcm.StandardVersionsFor(intent.L1ChainID)
		if err != nil {
			return fmt.Errorf("error getting standard versions TOML: %w", err)
		}
	}

	var dump *foundry.ForgeAllocs
	var dio opcm.DeployImplementationsOutput
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
				host.ImportState(st.SuperchainDeployment.StateDump)

				dio, err = opcm.DeployImplementations(
					host,
					opcm.DeployImplementationsInput{
						Salt:                            st.Create2Salt,
						WithdrawalDelaySeconds:          big.NewInt(604800),
						MinProposalSizeBytes:            big.NewInt(126000),
						ChallengePeriodSeconds:          big.NewInt(86400),
						ProofMaturityDelaySeconds:       big.NewInt(604800),
						DisputeGameFinalityDelaySeconds: big.NewInt(302400),
						Release:                         intent.ContractsRelease,
						SuperchainConfigProxy:           st.SuperchainDeployment.SuperchainConfigProxyAddress,
						ProtocolVersionsProxy:           st.SuperchainDeployment.ProtocolVersionsProxyAddress,
						OpcmProxyOwner:                  st.SuperchainDeployment.ProxyAdminAddress,
						StandardVersionsToml:            standardVersionsTOML,
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
		OpcmProxyAddress:                        dio.OpcmProxy,
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
