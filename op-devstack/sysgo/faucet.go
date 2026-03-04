package sysgo

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-faucet/config"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type FaucetService struct {
	service *faucet.Service
}

func (n *FaucetService) hydrate(system stack.ExtensibleSystem) {
	if n == nil || n.service == nil {
		return
	}

	require := system.T().Require()

	for faucetID, chainID := range n.service.Faucets() {
		faucetRPC := n.service.FaucetEndpoint(faucetID)
		rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), faucetRPC, client.WithLazyDial())
		require.NoError(err)
		system.T().Cleanup(rpcCl.Close)

		id := stack.NewFaucetID(faucetID.String(), chainID)
		front := shim.NewFaucet(shim.FaucetConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			ID:           id,
			Client:       rpcCl,
		})
		net := system.Network(chainID).(stack.ExtensibleNetwork)
		net.AddFaucet(front)
	}

	// Label the default faucets, in case we have multiple
	for chainID, faucetID := range n.service.Defaults() {
		id := stack.NewFaucetID(faucetID.String(), chainID)
		net := system.Network(chainID).(stack.ExtensibleNetwork)
		net.Faucet(stack.ByID[stack.Faucet](id)).SetLabel("default", "true")
	}
}

func isL2ELFaucetKind(kind stack.ComponentKind) bool {
	switch kind {
	case stack.KindL2ELNode, stack.KindRollupBoostNode, stack.KindOPRBuilderNode:
		return true
	default:
		return false
	}
}

func WithFaucets(l1ELs []stack.ComponentID, l2ELs []stack.ComponentID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		require.NotEmpty(l2ELs, "need at least one L2 EL for faucet service")
		for i, l1ELID := range l1ELs {
			require.Equalf(stack.KindL1ELNode, l1ELID.Kind(), "l1ELs[%d] must be kind %s", i, stack.KindL1ELNode)
			require.Truef(l1ELID.HasChainID(), "l1ELs[%d] must be chain-scoped", i)
		}
		for i, l2ELID := range l2ELs {
			require.Truef(isL2ELFaucetKind(l2ELID.Kind()),
				"l2ELs[%d] must be one of %s, %s, %s",
				i, stack.KindL2ELNode, stack.KindRollupBoostNode, stack.KindOPRBuilderNode)
			require.Truef(l2ELID.HasChainID(), "l2ELs[%d] must be chain-scoped", i)
		}

		faucetID := stack.NewFaucetID("dev-faucet", l2ELs[0].ChainID())
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), faucetID))
		require = p.Require()

		require.Nil(orch.faucet, "can only support a single faucet-service in sysgo")

		funderKey, err := orch.keys.Secret(devkeys.UserKey(funderMnemonicIndex))
		require.NoError(err, "need funder key")
		funderKeyStr := hexutil.Encode(crypto.FromECDSA(funderKey))

		faucets := make(map[ftypes.FaucetID]*fconf.FaucetEntry)
		for _, elID := range l1ELs {
			id := ftypes.FaucetID(fmt.Sprintf("dev-faucet-%s", elID.ChainID()))
			require.NotContains(faucets, id, "one faucet per chain only")

			el, ok := orch.GetL1EL(elID)
			require.True(ok, "need L1 EL for faucet", elID)

			faucets[id] = &fconf.FaucetEntry{
				ELRPC:   endpoint.MustRPC{Value: endpoint.URL(el.UserRPC())},
				ChainID: elID.ChainID(),
				TxCfg: fconf.TxManagerConfig{
					PrivateKey: funderKeyStr,
				},
			}
		}
		for _, elID := range l2ELs {
			id := ftypes.FaucetID(fmt.Sprintf("dev-faucet-%s", elID.ChainID()))
			require.NotContains(faucets, id, "one faucet per chain only")

			el, ok := orch.GetL2EL(elID)
			require.True(ok, "need L2 EL for faucet", elID)

			faucets[id] = &fconf.FaucetEntry{
				ELRPC:   endpoint.MustRPC{Value: endpoint.URL(el.UserRPC())},
				ChainID: elID.ChainID(),
				TxCfg: fconf.TxManagerConfig{
					PrivateKey: funderKeyStr,
				},
			}
		}
		cfg := &config.Config{
			RPC: oprpc.CLIConfig{
				ListenAddr: "127.0.0.1",
			},
			Faucets: &fconf.Config{
				Faucets: faucets,
			},
		}
		logger := p.Logger()
		srv, err := faucet.FromConfig(p.Ctx(), cfg, logger)
		require.NoError(err, "must setup faucet service")
		require.NoError(srv.Start(p.Ctx()))
		p.Cleanup(func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // force-quit
			logger.Info("Closing faucet")
			_ = srv.Stop(ctx)
			logger.Info("Closed faucet")
		})

		orch.faucet = &FaucetService{service: srv}
	})
}
