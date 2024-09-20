package pipeline

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/opsm"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer/state"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

// PermissionedGameStartingAnchorRoots is a root of `0xdead` for the permissioned game at block 0,
// and no root for the permissionless game. It was generated with a console.log in DeployOpChain.s.sol
var PermissionedGameStartingAnchorRoots = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x56, 0xd3, 0x26, 0xa8, 0xef, 0x15, 0x96, 0xf3, 0xd8, 0x73, 0x97, 0xc6, 0xc7, 0xb7, 0xb9, 0xda, 0xca, 0xf8, 0x68, 0x5e, 0xd9, 0x7d, 0xdd, 0x31, 0xf0, 0xaa, 0x04, 0x29, 0x4e, 0xaf, 0x9f, 0xca, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

func DeployOPChain(ctx context.Context, env *Env, intent *state.Intent, st *state.State, chainID common.Hash) error {
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

	var dco opsm.DeployOPChainOutput
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
					opsm.DeployOPChainInput{
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
					},
				)
				return err
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error deploying OP chain: %w", err)
	}

	st.Chains = append(st.Chains, &state.ChainState{
		ID: chainID,

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
