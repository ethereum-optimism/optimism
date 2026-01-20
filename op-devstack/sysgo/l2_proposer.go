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
	id      stack.ComponentID
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
	l2Net := system.L2Network(stack.ByID[stack.L2Network](stack.NewL2NetworkID(p.id.ChainID())))
	l2Net.(stack.ExtensibleL2Network).AddL2Proposer(bFrontend)
}

type ProposerOption func(id stack.ComponentID, cfg *ps.CLIConfig)

func WithProposerOption(opt ProposerOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.proposerOptions = append(o.proposerOptions, opt)
	})
}

func WithProposer(proposerID stack.ComponentID, l1ELID stack.ComponentID,
	l2CLID *stack.ComponentID, supervisorID *stack.ComponentID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		WithProposerPostDeploy(orch, proposerID, l1ELID, l2CLID, supervisorID, nil)
	})
}

func WithSuperProposer(proposerID stack.ComponentID, l1ELID stack.ComponentID,
	supervisorID *stack.ComponentID) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithProposerPostDeploy(orch, proposerID, l1ELID, nil, supervisorID, nil)
	})
}

func WithSupernodeProposer(proposerID stack.ComponentID, l1ELID stack.ComponentID,
	supernodeID *stack.SupernodeID) stack.Option[*Orchestrator] {
	return stack.Finally(func(orch *Orchestrator) {
		WithProposerPostDeploy(orch, proposerID, l1ELID, nil, nil, supernodeID)
	})
}

func WithProposerPostDeploy(orch *Orchestrator, proposerID stack.ComponentID, l1ELID stack.ComponentID,
	l2CLID *stack.ComponentID, supervisorID *stack.ComponentID, supernodeID *stack.SupernodeID) {
	ctx := orch.P().Ctx()
	ctx = stack.ContextWithID(ctx, proposerID)
	p := orch.P().WithCtx(ctx)

	require := p.Require()
	proposerCID := proposerID
	require.False(orch.registry.Has(proposerCID), "proposer must not already exist")
	if supervisorID != nil && supernodeID != nil {
		require.Fail("cannot have both supervisorID and supernodeID set for proposer")
	}

	proposerSecret, err := orch.keys.Secret(devkeys.ProposerRole.Key(proposerID.ChainID().ToBig()))
	require.NoError(err)

	logger := p.Logger()
	logger.Info("Proposer key acquired", "addr", crypto.PubkeyToAddress(proposerSecret.PublicKey))

	l1EL, ok := orch.GetL1EL(l1ELID)
	require.True(ok)

	l2Net, ok := orch.GetL2Network(stack.NewL2NetworkID(proposerID.ChainID()))
	require.True(ok)
	disputeGameFactoryAddr := l2Net.deployment.DisputeGameFactoryProxyAddr()
	disputeGameType := 1 // Permissioned game type is the only one currently deployed
	if orch.wb.outInteropMigration != nil {
		disputeGameFactoryAddr = orch.wb.outInteropMigration.DisputeGameFactory
		disputeGameType = 4 // SUPER_CANNON
	}

	proposerCLIConfig := &ps.CLIConfig{
		L1EthRpc:          l1EL.UserRPC(),
		PollInterval:      500 * time.Millisecond,
		AllowNonFinalized: true,
		TxMgrConfig:       setuputils.NewTxMgrConfig(endpoint.URL(l1EL.UserRPC()), proposerSecret),
		RPCConfig: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
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

	// If supervisor is available, use it. Otherwise, connect to L2 CL.
	switch {
	case supervisorID != nil:
		supervisorNode, ok := orch.GetSupervisor(*supervisorID)
		require.True(ok, "supervisor not found")
		proposerCLIConfig.SupervisorRpcs = []string{supervisorNode.UserRPC()}
	case supernodeID != nil:
		supernode, ok := orch.supernodes.Get(*supernodeID)
		require.True(ok, "supernode not found")
		proposerCLIConfig.SuperNodeRpcs = []string{supernode.UserRPC()}
	default:
		require.NotNil(l2CLID, "need L2 CL to connect to when no supervisor")
		l2CL, ok := orch.GetL2CL(*l2CLID)
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
	orch.registry.Register(proposerID, prop)
}
