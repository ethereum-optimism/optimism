package systemgo

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/devnet-sdk/system2"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
)

type L2Proposer struct {
	service *ps.ProposerService
	userRPC string
}

func WithProposer(proposerID system2.L2ProposerID, l1ELID system2.L1ELNodeID,
	l2CLID *system2.L2CLNodeID, supervisorID *system2.SupervisorID) system2.Option {
	return func(setup *system2.Setup) {
		orch := setup.Orchestrator.(*Orchestrator)
		setup.Require.False(orch.proposers.Has(proposerID), "proposer must not already exist")

		proposerSecret, err := orch.keys.Secret(devkeys.ProposerRole.Key(proposerID.ChainID.ToBig()))
		setup.Require.NoError(err)

		logger := setup.Log.New("id", proposerID)
		logger.Info("Proposer key acquired", "addr", crypto.PubkeyToAddress(proposerSecret.PublicKey))

		l1EL, ok := orch.l1ELs.Get(l1ELID)
		setup.Require.True(ok)

		l2ID := setup.System.L2NetworkID(proposerID.ChainID)
		l2Net := setup.System.L2Network(l2ID).(system2.ExtensibleL2Network)
		disputeGameFactoryAddr := l2Net.Deployment().DisputeGameFactoryProxyAddr()

		proposerCLIConfig := &ps.CLIConfig{
			L1EthRpc:          l1EL.userRPC,
			L2OOAddress:       "", // legacy, not used, fault-proofs support only for now.
			PollInterval:      500 * time.Millisecond,
			AllowNonFinalized: true,
			TxMgrConfig:       setuputils.NewTxMgrConfig(endpoint.URL(l1EL.userRPC), proposerSecret),
			RPCConfig:         oprpc.CLIConfig{},
			LogConfig: oplog.CLIConfig{
				Level:  log.LvlInfo,
				Format: oplog.FormatText,
			},
			MetricsConfig:                opmetrics.CLIConfig{},
			PprofConfig:                  oppprof.CLIConfig{},
			DGFAddress:                   disputeGameFactoryAddr.Hex(),
			ProposalInterval:             6 * time.Second,
			DisputeGameType:              1, // Permissioned game type is the only one currently deployed
			ActiveSequencerCheckDuration: time.Second * 5,
			WaitNodeSync:                 false,
		}

		if l2Net.ChainConfig().InteropTime != nil {
			setup.Require.NotNil(supervisorID, "need supervisor to connect to in interop")
			supervisorNode, ok := orch.supervisors.Get(*supervisorID)
			setup.Require.True(ok)
			proposerCLIConfig.SupervisorRpcs = []string{supervisorNode.userRPC}
		} else {
			setup.Require.NotNil(*l2CLID, "need L2 CL to connect to pre-interop")
			l2CL, ok := orch.l2CLs.Get(*l2CLID)
			setup.Require.True(ok)
			proposerCLIConfig.RollupRpc = l2CL.rpc
		}

		proposer, err := ps.ProposerServiceFromCLIConfig(context.Background(), "0.0.1", proposerCLIConfig, logger)
		setup.Require.NoError(err)

		setup.Require.NoError(proposer.Start(setup.Ctx))
		orch.t.Cleanup(func() {
			ctx, cancel := context.WithCancel(setup.Ctx)
			cancel() // force-quit
			logger.Info("Closing proposer")
			_ = proposer.Stop(ctx)
			logger.Info("Closed proposer")
		})

		p := &L2Proposer{
			service: proposer,
			userRPC: proposer.HTTPEndpoint(),
		}
		orch.proposers.Set(proposerID, p)

		rpcCl, err := client.NewRPC(setup.Ctx, setup.Log, p.userRPC, client.WithLazyDial())
		setup.Require.NoError(err)

		bFrontend := system2.NewL2Proposer(system2.L2ProposerConfig{
			CommonConfig: setup.CommonConfig(),
			ID:           proposerID,
			Client:       rpcCl,
		})
		l2Net.AddL2Proposer(bFrontend)
	}
}
