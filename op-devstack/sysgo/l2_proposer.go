package sysgo

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
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
	id      stack.L2ProposerID
	service *ps.ProposerService
	userRPC string
}

func (p *L2Proposer) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), p.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	bFrontend := shim.NewL2Proposer(shim.L2ProposerConfig{
		CommonConfig: shim.NewCommonConfig(system.T()),
		ID:           p.id,
		Client:       rpcCl,
	})
	l2Net := system.L2Network(stack.L2NetworkID(p.id.ChainID()))
	l2Net.(stack.ExtensibleL2Network).AddL2Proposer(bFrontend)
}

type ProposerOption func(id stack.L2ProposerID, cfg *ps.CLIConfig)

func WithProposerOption(opt ProposerOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.proposerOptions = append(o.proposerOptions, opt)
	})
}

func WithProposer(proposerID stack.L2ProposerID, l1ELID stack.L1ELNodeID,
	l2CLID *stack.L2CLNodeID, supervisorID *stack.SupervisorID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		WithProposerPostDeploy(orch, proposerID, l1ELID, l2CLID, supervisorID)
	})
}

func WithSuperProposer(proposerID stack.L2ProposerID, l1ELID stack.L1ELNodeID,
	supervisorID *stack.SupervisorID) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithProposerPostDeploy(orch, proposerID, l1ELID, nil, supervisorID)
	})
}

func WithProposerPostDeploy(orch *Orchestrator, proposerID stack.L2ProposerID, l1ELID stack.L1ELNodeID,
	l2CLID *stack.L2CLNodeID, supervisorID *stack.SupervisorID) {
	ctx := orch.P().Ctx()
	ctx = stack.ContextWithID(ctx, proposerID)
	p := orch.P().WithCtx(ctx)

	require := p.Require()
	require.False(orch.proposers.Has(proposerID), "proposer must not already exist")

	proposerSecret, err := orch.keys.Secret(devkeys.ProposerRole.Key(proposerID.ChainID().ToBig()))
	require.NoError(err)

	logger := p.Logger()
	logger.Info("Proposer key acquired", "addr", crypto.PubkeyToAddress(proposerSecret.PublicKey))

	l1EL, ok := orch.l1ELs.Get(l1ELID)
	require.True(ok)

	l2Net, ok := orch.l2Nets.Get(proposerID.ChainID())
	require.True(ok)
	disputeGameFactoryAddr := l2Net.deployment.DisputeGameFactoryProxyAddr()
	disputeGameType := 1 // Permissioned game type is the only one currently deployed
	if orch.wb.outInteropMigration != nil {
		disputeGameFactoryAddr = orch.wb.outInteropMigration.DisputeGameFactory
		disputeGameType = 4 // SUPER_CANNON
	}

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
		DisputeGameType:              uint32(disputeGameType),
		ActiveSequencerCheckDuration: time.Second * 5,
		WaitNodeSync:                 false,
	}
	for _, opt := range orch.proposerOptions {
		opt(proposerID, proposerCLIConfig)
	}

	// If interop is scheduled, or if we cannot do the pre-interop connection, then set up with supervisor
	if l2Net.genesis.Config.InteropTime != nil || l2CLID == nil {
		require.NotNil(supervisorID, "need supervisor to connect to in interop")
		supervisorNode, ok := orch.supervisors.Get(*supervisorID)
		require.True(ok)
		proposerCLIConfig.SupervisorRpcs = []string{supervisorNode.userRPC}
	} else {
		require.NotNil(l2CLID, "need L2 CL to connect to pre-interop")
		l2CL, ok := orch.l2CLs.Get(*l2CLID)
		require.True(ok)
		proposerCLIConfig.RollupRpc = l2CL.UserRPC()
	}

	proposer, err := ps.ProposerServiceFromCLIConfig(ctx, "0.0.1", proposerCLIConfig, logger)
	require.NoError(err)

	require.NoError(proposer.Start(ctx))
	p.Cleanup(func() {
		ctx, cancel := context.WithCancel(ctx)
		cancel() // force-quit
		logger.Info("Closing proposer")
		_ = proposer.Stop(ctx)
		logger.Info("Closed proposer")
	})

	prop := &L2Proposer{
		id:      proposerID,
		service: proposer,
		userRPC: proposer.HTTPEndpoint(),
	}
	orch.proposers.Set(proposerID, prop)
}
