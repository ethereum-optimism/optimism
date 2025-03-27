package systemgo

import (
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	gn "github.com/ethereum/go-ethereum/node"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

type L2ELNode struct {
	authRPC string
	userRPC string
}

func WithL2ELNode(id system2.L2ELNodeID, supervisorID *system2.SupervisorID) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)

		l2ID := setup.System.L2NetworkID(id.ChainID)

		l2Net, ok := orch.l2Nets.Get(l2ID)
		setup.Require.True(ok, "L2 network required")

		sysL2Net := setup.System.L2Network(l2ID).(system2.ExtensibleL2Network)

		jwtPath, _ := orch.writeDefaultJWT()

		useInterop := sysL2Net.ChainConfig().InteropTime != nil

		supervisorRPC := ""
		if useInterop {
			setup.Require.NotNil(supervisorID, "supervisor is required for interop")
			sup, ok := orch.supervisors.Get(*supervisorID)
			setup.Require.True(ok, "supervisor is required for interop")
			supervisorRPC = sup.userRPC
		}

		l2Geth, err := geth.InitL2(id.String(), l2Net.genesis, jwtPath,
			func(ethCfg *ethconfig.Config, nodeCfg *gn.Config) error {
				ethCfg.InteropMessageRPC = supervisorRPC
				ethCfg.InteropMempoolFiltering = true // TODO option
				return nil
			})
		setup.Require.NoError(err)
		setup.Require.NoError(l2Geth.Node.Start())
		orch.t.Cleanup(func() {
			setup.Log.Info("Closing op-geth", "id", id)
			closeErr := l2Geth.Close()
			setup.Log.Info("Closed op-geth", "id", id, "err", closeErr)
		})

		rpcCl, err := client.NewRPC(setup.Ctx, setup.Log, l2Geth.UserRPC().RPC(), client.WithLazyDial())
		setup.Require.NoError(err)

		l2EL := &L2ELNode{
			authRPC: l2Geth.AuthRPC().RPC(),
			userRPC: l2Geth.UserRPC().RPC(),
		}
		setup.Require.True(orch.l2ELs.SetIfMissing(id, l2EL), "must be unique L2 EL node")

		sysL2EL := system2.NewL2ELNode(system2.L2ELNodeConfig{
			ELNodeConfig: system2.ELNodeConfig{
				CommonConfig: setup.CommonConfig(),
				Client:       rpcCl,
				ChainID:      id.ChainID,
			},
			ID: id,
		})
		sysL2Net.AddL2ELNode(sysL2EL)
	}
}
