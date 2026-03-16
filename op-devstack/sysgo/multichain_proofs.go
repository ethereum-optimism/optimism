package sysgo

import (
	"context"
	"runtime"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opchallenger "github.com/ethereum-optimism/optimism/op-challenger"
	challengermetrics "github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	sharedchallenger "github.com/ethereum-optimism/optimism/op-devstack/shared/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	ps "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

func withSuperProofsDeployerFeature(cfg PresetConfig) PresetConfig {
	cfg.DeployerOptions = append([]DeployerOption{
		WithDevFeatureEnabled(deployer.OptimismPortalInteropDevFlag),
	}, cfg.DeployerOptions...)
	return cfg
}

func orderedRuntimeChains(runtime *MultiChainRuntime) []*MultiChainNodeRuntime {
	keys := make([]string, 0, len(runtime.Chains))
	for key := range runtime.Chains {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	chains := make([]*MultiChainNodeRuntime, 0, len(keys))
	for _, key := range keys {
		chains = append(chains, runtime.Chains[key])
	}
	return chains
}

func attachSupervisorSuperProofs(t devtest.T, runtime *MultiChainRuntime, cfg PresetConfig) *MultiChainRuntime {
	chains := orderedRuntimeChains(runtime)
	t.Require().NotEmpty(chains, "supervisor superproofs runtime must contain at least one chain")
	t.Require().NotNil(runtime.PrimarySupervisor, "supervisor superproofs runtime must provide a supervisor")

	proofChain := chains[0]
	cls := make([]L2CLNode, 0, len(chains))
	nets := make([]*L2Network, 0, len(chains))
	els := make([]L2ELNode, 0, len(chains))
	for _, chain := range chains {
		t.Require().NotNil(chain, "runtime chain entry must not be nil")
		cls = append(cls, chain.CL)
		nets = append(nets, chain.Network)
		els = append(els, chain.EL)
	}

	superrootTime := awaitSuperrootTime(t, cls...)
	superRoot := getSupervisorSuperRoot(t, runtime.PrimarySupervisor, superrootTime)
	migrateSuperRoots(t, runtime.Keys, runtime.Migration, runtime.L1Network.ChainID(), runtime.L1EL, superRoot, superrootTime, proofChain.Network.ChainID())

	challenger := startInteropChallenger(
		t,
		runtime.Keys,
		runtime.L1Network,
		runtime.L1EL,
		runtime.L1CL,
		runtime.DependencySet,
		runtime.PrimarySupervisor.UserRPC(),
		false,
		nets,
		els,
		cfg.EnableCannonKonaForChall,
	)
	runtime.L2ChallengerConfig = challenger.Config()

	_ = startSuperProposer(
		t,
		runtime.Keys,
		"main",
		proofChain.Network.ChainID(),
		runtime.L1EL,
		proofChain.Network,
		runtime.PrimarySupervisor.UserRPC(),
		"",
		cfg.ProposerOptions...,
	)

	return runtime
}

func attachSupernodeSuperProofs(t devtest.T, runtime *MultiChainRuntime, cfg PresetConfig) *MultiChainRuntime {
	chains := orderedRuntimeChains(runtime)
	t.Require().NotEmpty(chains, "supernode superproofs runtime must contain at least one chain")
	t.Require().NotNil(runtime.Supernode, "supernode superproofs runtime must provide a supernode")

	proofChain := chains[0]
	cls := make([]L2CLNode, 0, len(chains))
	nets := make([]*L2Network, 0, len(chains))
	els := make([]L2ELNode, 0, len(chains))
	for _, chain := range chains {
		t.Require().NotNil(chain, "runtime chain entry must not be nil")
		cls = append(cls, chain.CL)
		nets = append(nets, chain.Network)
		els = append(els, chain.EL)
	}

	superrootTime := awaitSuperrootTime(t, cls...)
	superRoot := getSupernodeSuperRoot(t, runtime.Supernode, superrootTime)
	migrateSuperRoots(t, runtime.Keys, runtime.Migration, runtime.L1Network.ChainID(), runtime.L1EL, superRoot, superrootTime, proofChain.Network.ChainID())

	challenger := startInteropChallenger(
		t,
		runtime.Keys,
		runtime.L1Network,
		runtime.L1EL,
		runtime.L1CL,
		runtime.DependencySet,
		runtime.Supernode.UserRPC(),
		true,
		nets,
		els,
		cfg.EnableCannonKonaForChall,
	)
	runtime.L2ChallengerConfig = challenger.Config()

	_ = startSuperProposer(
		t,
		runtime.Keys,
		"main",
		proofChain.Network.ChainID(),
		runtime.L1EL,
		proofChain.Network,
		"",
		runtime.Supernode.UserRPC(),
		cfg.ProposerOptions...,
	)

	return runtime
}

func NewSimpleInteropSuperProofsRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperProofsDeployerFeature(cfg)
	return attachSupervisorSuperProofs(t, NewSimpleInteropRuntimeWithConfig(t, cfg), cfg)
}

func NewTwoL2SupernodeProofsRuntimeWithConfig(t devtest.T, interopAtGenesis bool, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperProofsDeployerFeature(cfg)
	runtime, _ := newTwoL2SupernodeRuntimeWithConfig(t, interopAtGenesis, 0, cfg)
	attachTestSequencerToRuntime(t, runtime, "test-sequencer-2l2")
	return attachSupernodeSuperProofs(t, runtime, cfg)
}

func NewSingleChainSupernodeProofsRuntimeWithConfig(t devtest.T, interopAtGenesis bool, cfg PresetConfig) *MultiChainRuntime {
	cfg = withSuperProofsDeployerFeature(cfg)
	runtime := newSingleChainSupernodeRuntimeWithConfig(t, interopAtGenesis, cfg)
	attachTestSequencerToRuntime(t, runtime, "dev")
	return attachSupernodeSuperProofs(t, runtime, cfg)
}

func startSuperProposer(
	t devtest.T,
	keys devkeys.Keys,
	proposerName string,
	proposerChainID eth.ChainID,
	l1EL L1ELNode,
	l2Net *L2Network,
	supervisorRPC string,
	supernodeRPC string,
	proposerOpts ...ProposerOption,
) *L2Proposer {
	require := t.Require()

	proposerSecret, err := keys.Secret(devkeys.ProposerRole.Key(proposerChainID.ToBig()))
	require.NoError(err)

	logger := t.Logger().New("component", "l2-proposer")
	logger.Info("Proposer key acquired", "addr", crypto.PubkeyToAddress(proposerSecret.PublicKey))

	proposerCLIConfig := &ps.CLIConfig{
		L1EthRpc:                     l1EL.UserRPC(),
		PollInterval:                 500 * time.Millisecond,
		AllowNonFinalized:            true,
		TxMgrConfig:                  setuputils.NewTxMgrConfig(endpoint.URL(l1EL.UserRPC()), proposerSecret),
		RPCConfig:                    oprpc.CLIConfig{ListenAddr: "127.0.0.1"},
		LogConfig:                    oplog.CLIConfig{Level: log.LvlInfo, Format: oplog.FormatText},
		MetricsConfig:                opmetrics.CLIConfig{},
		PprofConfig:                  oppprof.CLIConfig{},
		DGFAddress:                   l2Net.deployment.DisputeGameFactoryProxyAddr().Hex(),
		ProposalInterval:             6 * time.Second,
		DisputeGameType:              superCannonGameType,
		ActiveSequencerCheckDuration: 5 * time.Second,
		WaitNodeSync:                 false,
	}
	for _, opt := range proposerOpts {
		if opt == nil {
			continue
		}
		opt(NewComponentTarget(proposerName, proposerChainID), proposerCLIConfig)
	}
	switch {
	case supernodeRPC != "":
		proposerCLIConfig.SuperNodeRpcs = []string{supernodeRPC}
	case supervisorRPC != "":
		proposerCLIConfig.SupervisorRpcs = []string{supervisorRPC}
	default:
		require.FailNow("need supervisor or supernode RPC for super proposer")
	}

	proposer, err := ps.ProposerServiceFromCLIConfig(t.Ctx(), "0.0.1", proposerCLIConfig, logger)
	require.NoError(err)
	require.NoError(proposer.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing proposer")
		_ = proposer.Stop(ctx)
		logger.Info("Closed proposer")
	})

	return &L2Proposer{
		name:    proposerName,
		chainID: proposerChainID,
		service: proposer,
		userRPC: proposer.HTTPEndpoint(),
	}
}

func startInteropChallenger(
	t devtest.T,
	keys devkeys.Keys,
	l1Net *L1Network,
	l1EL L1ELNode,
	l1CL *L1CLNode,
	depSet depset.DependencySet,
	superRPC string,
	useSuperNode bool,
	l2Nets []*L2Network,
	l2ELs []L2ELNode,
	enableCannonKona bool,
) *L2Challenger {
	require := t.Require()
	require.NotEmpty(l2Nets, "at least one L2 network is required")
	require.Len(l2ELs, len(l2Nets), "need matching L2 ELs for challenger")

	challengerSecret, err := keys.Secret(devkeys.ChallengerRole.Key(l2Nets[0].ChainID().ToBig()))
	require.NoError(err)

	logger := t.Logger().New("component", "l2-challenger")
	logger.Info("Challenger key acquired", "addr", crypto.PubkeyToAddress(challengerSecret.PublicKey))

	l2ELRPCs := make([]string, len(l2ELs))
	rollupCfgs := make([]*rollup.Config, len(l2Nets))
	l2Geneses := make([]*core.Genesis, len(l2Nets))
	l2ChainIDs := make([]eth.ChainID, len(l2Nets))
	for i := range l2Nets {
		l2ELRPCs[i] = l2ELs[i].UserRPC()
		rollupCfgs[i] = l2Nets[i].rollupCfg
		l2Geneses[i] = l2Nets[i].genesis
		l2ChainIDs[i] = l2Nets[i].ChainID()
	}
	staticDepSet, ok := depSet.(*depset.StaticConfigDependencySet)
	require.True(ok, "expected static dependency set for super challenger")

	options := []sharedchallenger.Option{
		sharedchallenger.WithFactoryAddress(l2Nets[0].deployment.DisputeGameFactoryProxyAddr()),
		sharedchallenger.WithPrivKey(challengerSecret),
		sharedchallenger.WithDepset(staticDepSet),
		sharedchallenger.WithCannonConfig(rollupCfgs, l1Net.genesis, l2Geneses, sharedchallenger.InteropVariant),
		sharedchallenger.WithSuperCannonGameType(),
		sharedchallenger.WithSuperPermissionedGameType(),
	}
	if enableCannonKona {
		t.Log("Enabling cannon-kona for super challenger")
		options = append(options,
			sharedchallenger.WithCannonKonaInteropConfig(rollupCfgs, l1Net.genesis, l2Geneses),
			sharedchallenger.WithSuperCannonKonaGameType(),
			sharedchallenger.WithExperimentalWitnessEndpoint(),
		)
	}
	cfg, err := sharedchallenger.NewInteropChallengerConfig(
		t.TempDir(),
		l1EL.UserRPC(),
		l1CL.beaconHTTPAddr,
		superRPC,
		l2ELRPCs,
		options...,
	)
	require.NoError(err, "failed to create interop challenger config")
	cfg.UseSuperNode = useSuperNode

	svc, err := opchallenger.Main(t.Ctx(), logger, cfg, challengermetrics.NoopMetrics)
	require.NoError(err)
	require.NoError(svc.Start(t.Ctx()))
	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing challenger")
		timer := time.AfterFunc(time.Minute, func() {
			if svc.Stopped() {
				return
			}
			buf := make([]byte, 1<<20)
			stackLen := runtime.Stack(buf, true)
			logger.Error("Challenger failed to stop; printing all goroutine stacks:\n%v", string(buf[:stackLen]))
		})
		_ = svc.Stop(ctx)
		timer.Stop()
		logger.Info("Closed challenger")
	})

	return &L2Challenger{
		name:     "main",
		chainIDs: l2ChainIDs,
		service:  svc,
		config:   cfg,
	}
}
