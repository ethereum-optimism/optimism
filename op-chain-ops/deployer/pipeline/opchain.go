package pipeline

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/broadcaster"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

func DeployOPChain(ctx context.Context, env *Env, artifactsFS foundry.StatDirFs, intent *state.Intent, st *state.State, chainID common.Hash) error {
	lgr := env.Logger.New("stage", "deploy-opchain")

	if !shouldDeployOPChain(intent, st, chainID) {
		lgr.Info("opchain deployment not needed")
		return nil
	}

	lgr.Info("deploying OP chain", "id", chainID.Hex())

	thisIntent, err := intent.Chain(chainID)
	if err != nil {
		return fmt.Errorf("failed to get chain intent: %w", err)
	}

	input := opsm.DeployOPChainInput{
		OpChainProxyAdminOwner: thisIntent.Roles.ProxyAdminOwner,
		SystemConfigOwner:      thisIntent.Roles.SystemConfigOwner,
		Batcher:                thisIntent.Roles.Batcher,
		UnsafeBlockSigner:      thisIntent.Roles.UnsafeBlockSigner,
		Proposer:               thisIntent.Roles.Proposer,
		Challenger:             thisIntent.Roles.Challenger,
		BasefeeScalar:          1368,
		BlobBaseFeeScalar:      801949,
		L2ChainId:              chainID.Big(),
		OpsmProxy:              st.ImplementationsDeployment.OpsmProxyAddress,
	}

	var dco opsm.DeployOPChainOutput
	if intent.OPSMAddress == (common.Address{}) {
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
					host.ImportState(st.ImplementationsDeployment.StateDump)

					dco, err = opsm.DeployOPChain(
						host,
						input,
					)
					return err
				},
			},
		)
		if err != nil {
			return fmt.Errorf("error deploying OP chain: %w", err)
		}
	} else {
		lgr.Info("deploying using existing OPSM", "address", intent.OPSMAddress.Hex())

		bcaster, err := broadcaster.NewKeyedBroadcaster(broadcaster.KeyedBroadcasterOpts{
			Logger:  lgr,
			ChainID: big.NewInt(int64(intent.L1ChainID)),
			Client:  env.L1Client,
			Signer:  env.Signer,
			From:    env.Deployer,
		})
		if err != nil {
			return fmt.Errorf("failed to create broadcaster: %w", err)
		}
		dco, err = opsm.DeployOPChainRaw(
			ctx,
			env.L1Client,
			bcaster,
			artifactsFS,
			input,
		)
		if err != nil {
			return fmt.Errorf("error deploying OP chain: %w", err)
		}
	}

	st.Chains = append(st.Chains, &state.ChainState{
		ID:                                        chainID,
		ProxyAdminAddress:                         dco.OpChainProxyAdmin,
		AddressManagerAddress:                     dco.AddressManager,
		L1ERC721BridgeProxyAddress:                dco.L1ERC721BridgeProxy,
		SystemConfigProxyAddress:                  dco.SystemConfigProxy,
		OptimismMintableERC20FactoryProxyAddress:  dco.OptimismMintableERC20FactoryProxy,
		L1StandardBridgeProxyAddress:              dco.L1StandardBridgeProxy,
		L1CrossDomainMessengerProxyAddress:        dco.L1CrossDomainMessengerProxy,
		OptimismPortalProxyAddress:                dco.OptimismPortalProxy,
		DisputeGameFactoryProxyAddress:            dco.DisputeGameFactoryProxy,
		AnchorStateRegistryProxyAddress:           dco.AnchorStateRegistryProxy,
		AnchorStateRegistryImplAddress:            dco.AnchorStateRegistryImpl,
		FaultDisputeGameAddress:                   dco.FaultDisputeGame,
		PermissionedDisputeGameAddress:            dco.PermissionedDisputeGame,
		DelayedWETHPermissionedGameProxyAddress:   dco.DelayedWETHPermissionedGameProxy,
		DelayedWETHPermissionlessGameProxyAddress: dco.DelayedWETHPermissionlessGameProxy,
	})
	if err := env.WriteState(st); err != nil {
		return err
	}

	return nil
}

func shouldDeployOPChain(intent *state.Intent, st *state.State, chainID common.Hash) bool {
	for _, chain := range st.Chains {
		if chain.ID == chainID {
			return false
		}
	}

	return true
}
