package pipeline

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

func DeployImplementations(ctx context.Context, env *Env, bundle ArtifactsBundle, intent *state.Intent, st *state.State) error {
	lgr := env.Logger.New("stage", "deploy-implementations")

	if !shouldDeployImplementations(intent, st) {
		lgr.Info("implementations deployment not needed")
		return nil
	}

	lgr.Info("deploying implementations")

	var standardVersionsTOML string
	var contractsRelease string
	var err error
	if intent.L1ContractsLocator.IsTag() {
		standardVersionsTOML, err = opcm.StandardL1VersionsDataFor(intent.L1ChainID)
		if err != nil {
			return fmt.Errorf("error getting standard versions TOML: %w", err)
		}
		contractsRelease = intent.L1ContractsLocator.Tag
	} else {
		contractsRelease = "dev"
	}

	var dump *foundry.ForgeAllocs
	var dio opcm.DeployImplementationsOutput
	err = CallScriptBroadcast(
		ctx,
		CallScriptBroadcastOpts{
			L1ChainID:   big.NewInt(int64(intent.L1ChainID)),
			Logger:      lgr,
			ArtifactsFS: bundle.L1,
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
						MipsVersion:                     big.NewInt(1),
						Release:                         contractsRelease,
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

	return nil
}

func shouldDeployImplementations(intent *state.Intent, st *state.State) bool {
	return st.ImplementationsDeployment == nil
}
