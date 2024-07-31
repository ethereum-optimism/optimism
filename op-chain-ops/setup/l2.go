package setup

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-chain-ops/setup/deploy"
	"github.com/ethereum-optimism/optimism/op-chain-ops/setup/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type L2 interface {
	Chain

	SetupL2(l2Init genesis.L2InitializationConfig)

	DeployFaultProofs(cfg genesis.FaultProofDeployConfig)
}

var _ L2 = (*chain)(nil)

type l2Props struct {
	l1Chain *uint256.Int

	l1Deployments genesis.L1Deployments
}

func (ch *chain) DeployFaultProofs(cfg genesis.FaultProofDeployConfig) {
	// TODO
}

func (ch *chain) SetupL2(l2Init genesis.L2InitializationConfig) {
	ch.w.req.Nil(ch.l2, "cannot setup L2 twice")
	ch.l2 = &l2Props{
		l1Chain: new(uint256.Int).SetUint64(l2Init.L1ChainID),
	}

	// Deploy the contracts to L1
	l1Deps := func() genesis.L1Deployments {
		deployL1 := &deploy.L1ContractsOfL2{
			Args: struct{ genesis.L2InitializationConfig }{
				L2InitializationConfig: l2Init,
				// TODO need to add FP / OO deployment
			},
			Addresses: struct{}{},
		}
		l1Chain := ch.w.chains[*ch.l2.l1Chain]
		res, err := script.Run(ch.w.ctx, ch.log, ch.w.cachePath, l1Chain.state, deployL1)
		ch.w.req.NoError(err)
		l1Chain.state = res.Post
		l1Chain.labels.AddLabels(res.LabelsDiff)
		l1Chain.labels.LabelDeployments(res.DeploymentsDiff)

		return genesis.L1Deployments{
			AddressManager:                    res.DeploymentsDiff["AddressManager"],
			DisputeGameFactory:                res.DeploymentsDiff["DisputeGameFactory"],
			DisputeGameFactoryProxy:           res.DeploymentsDiff["DisputeGameFactoryProxy"],
			L1CrossDomainMessenger:            res.DeploymentsDiff["L1CrossDomainMessenger"],
			L1CrossDomainMessengerProxy:       res.DeploymentsDiff["L1CrossDomainMessengerProxy"],
			L1ERC721Bridge:                    res.DeploymentsDiff["L1ERC721Bridge"],
			L1ERC721BridgeProxy:               res.DeploymentsDiff["L1ERC721BridgeProxy"],
			L1StandardBridge:                  res.DeploymentsDiff["L1StandardBridge"],
			L1StandardBridgeProxy:             res.DeploymentsDiff["L1StandardBridgeProxy"],
			L2OutputOracle:                    res.DeploymentsDiff["L2OutputOracle"],
			L2OutputOracleProxy:               res.DeploymentsDiff["L2OutputOracleProxy"],
			OptimismMintableERC20Factory:      res.DeploymentsDiff["OptimismMintableERC20Factory"],
			OptimismMintableERC20FactoryProxy: res.DeploymentsDiff["OptimismMintableERC20FactoryProxy"],
			OptimismPortal:                    res.DeploymentsDiff["OptimismPortal"],
			OptimismPortalProxy:               res.DeploymentsDiff["OptimismPortalProxy"],
			ProxyAdmin:                        res.DeploymentsDiff["ProxyAdmin"],
			SystemConfig:                      res.DeploymentsDiff["SystemConfig"],
			SystemConfigProxy:                 res.DeploymentsDiff["SystemConfigProxy"],
			ProtocolVersions:                  common.Address{},
			ProtocolVersionsProxy:             common.Address{},
			DataAvailabilityChallenge:         common.Address{},
			DataAvailabilityChallengeProxy:    common.Address{},
		}
	}()

	deployerL2 := &deploy.FullL2{
		Args: struct {
			genesis.L2InitializationConfig
			genesis.L1DependenciesConfig
		}{
			L2InitializationConfig: l2Init,
			L1DependenciesConfig: genesis.L1DependenciesConfig{
				L1StandardBridgeProxy:       l1Deps.L1StandardBridgeProxy,
				L1CrossDomainMessengerProxy: l1Deps.L1CrossDomainMessengerProxy,
				L1ERC721BridgeProxy:         l1Deps.L1ERC721BridgeProxy,
				SystemConfigProxy:           l1Deps.SystemConfigProxy,
				OptimismPortalProxy:         l1Deps.OptimismPortalProxy,
				DAChallengeProxy:            l1Deps.DataAvailabilityChallengeProxy,
			},
		},
		Addresses: struct{}{},
	}
	res, err := script.Run(ch.w.ctx, ch.log, ch.w.cachePath, ch.state, deployerL2)
	ch.w.req.NoError(err)
	ch.state = res.Post
	ch.labels.AddLabels(res.LabelsDiff)
	ch.labels.LabelDeployments(res.DeploymentsDiff)

}
