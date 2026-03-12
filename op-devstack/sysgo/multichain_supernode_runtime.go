package sysgo

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	faucetConfig "github.com/ethereum-optimism/optimism/op-faucet/config"
	"github.com/ethereum-optimism/optimism/op-faucet/faucet"
	fconf "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/config"
	ftypes "github.com/ethereum-optimism/optimism/op-faucet/faucet/backend/types"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
	sequencerConfig "github.com/ethereum-optimism/optimism/op-test-sequencer/config"
	testmetrics "github.com/ethereum-optimism/optimism/op-test-sequencer/metrics"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/fakepos"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/builders/standardbuilder"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/noopcommitter"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/committers/standardcommitter"
	workconfig "github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/config"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/nooppublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/publishers/standardpublisher"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/sequencers/fullseq"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/localkey"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work/signers/noopsigner"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func NewTwoL2SupernodeRuntime(t devtest.T) *MultiChainRuntime {
	runtime, _ := newTwoL2SupernodeRuntime(t, false, 0)
	return runtime
}

func NewTwoL2SupernodeInteropRuntime(t devtest.T, delaySeconds uint64) *MultiChainRuntime {
	return NewTwoL2SupernodeInteropRuntimeWithConfig(t, delaySeconds, PresetConfig{})
}

func NewTwoL2SupernodeInteropRuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	base, activationTime := newTwoL2SupernodeRuntimeWithConfig(t, true, delaySeconds, cfg)
	chainA := base.Chains["l2a"]
	chainB := base.Chains["l2b"]
	t.Require().NotNil(chainA, "missing l2a supernode chain")
	t.Require().NotNil(chainB, "missing l2b supernode chain")
	attachTestSequencerToRuntime(t, base, "test-sequencer-2l2")

	t.Logger().Info("configured supernode interop runtime",
		"genesis_time", chainA.Network.rollupCfg.Genesis.L2Time,
		"activation_time", activationTime,
		"delay_seconds", delaySeconds,
	)

	base.DelaySeconds = delaySeconds
	return base
}

func NewTwoL2SupernodeFollowL2RuntimeWithConfig(t devtest.T, delaySeconds uint64, cfg PresetConfig) *MultiChainRuntime {
	runtime := NewTwoL2SupernodeInteropRuntimeWithConfig(t, delaySeconds, cfg)
	addMultiChainFollowL2Node(t, runtime, "l2a", "follower")
	addMultiChainFollowL2Node(t, runtime, "l2b", "follower")
	return runtime
}

func newTwoL2SupernodeRuntime(t devtest.T, enableInterop bool, delaySeconds uint64) (*MultiChainRuntime, uint64) {
	return newTwoL2SupernodeRuntimeWithConfig(t, enableInterop, delaySeconds, PresetConfig{})
}

func NewTwoL2SupernodeRuntimeWithConfig(t devtest.T, cfg PresetConfig) *MultiChainRuntime {
	runtime, _ := newTwoL2SupernodeRuntimeWithConfig(t, false, 0, cfg)
	return runtime
}

func newSingleChainSupernodeRuntimeWithConfig(t devtest.T, interopAtGenesis bool, cfg PresetConfig) *MultiChainRuntime {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	migration, l1Net, l2Net, depSet, _ := buildSingleChainWorldWithInteropAndState(t, keys, interopAtGenesis, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	validateSimpleInteropPresetConfig(t, cfg, l2Net)

	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClock(t, l1Net, jwtPath, l1Clock)
	l2EL := startSequencerEL(t, l2Net, jwtPath, jwtSecret, NewELNodeIdentity(0))

	var depSetStatic *depset.StaticConfigDependencySet
	if depSet != nil {
		cast, ok := depSet.(*depset.StaticConfigDependencySet)
		require.True(ok, "expected static dependency set")
		depSetStatic = cast
	}

	supernode, l2CL := startSingleChainSharedSupernode(t, l1Net, l1EL, l1CL, l2Net, l2EL, depSetStatic, jwtSecret, interopAtGenesis)
	l2Batcher := startMinimalBatcher(t, keys, l2Net, l1EL, l2CL, l2EL, cfg.BatcherOptions...)
	l2Proposer := startMinimalProposer(t, keys, l2Net, l1EL, l2CL, cfg.ProposerOptions...)
	faucetService := startFaucets(t, keys, l1Net.ChainID(), l2Net.ChainID(), l1EL.UserRPC(), l2EL.UserRPC())

	return &MultiChainRuntime{
		Keys:          keys,
		Migration:     migration,
		DependencySet: depSet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:     "l2a",
				Network:  l2Net,
				EL:       l2EL,
				CL:       l2CL,
				Batcher:  l2Batcher,
				Proposer: l2Proposer,
			},
		},
		Supernode:     supernode,
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
	}
}

func newTwoL2SupernodeRuntimeWithConfig(t devtest.T, enableInterop bool, delaySeconds uint64, cfg PresetConfig) (*MultiChainRuntime, uint64) {
	require := t.Require()

	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(err, "failed to derive dev keys from mnemonic")

	wb, l1Net, l2ANet, l2BNet := buildTwoL2RuntimeWorld(t, keys, enableInterop, cfg.LocalContractArtifactsPath, cfg.DeployerOptions...)
	jwtPath, jwtSecret := writeJWTSecret(t)
	l1Clock := clock.SystemClock
	var timeTravelClock *clock.AdvancingClock
	if cfg.EnableTimeTravel {
		timeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		l1Clock = timeTravelClock
	}
	l1EL, l1CL := startInProcessL1WithClock(t, l1Net, jwtPath, l1Clock)

	l2AIdentity := NewELNodeIdentity(0)
	l2BIdentity := NewELNodeIdentity(0)
	l2AEL := startSequencerEL(t, l2ANet, jwtPath, jwtSecret, l2AIdentity)
	l2BEL := startSequencerEL(t, l2BNet, jwtPath, jwtSecret, l2BIdentity)

	var activationTime uint64
	var interopActivationTimestamp *uint64
	if enableInterop {
		activationTime = l2ANet.rollupCfg.Genesis.L2Time + delaySeconds
		interopActivationTimestamp = &activationTime
	}

	var depSet *depset.StaticConfigDependencySet
	if wb.outFullCfgSet.DependencySet != nil {
		cast, ok := wb.outFullCfgSet.DependencySet.(*depset.StaticConfigDependencySet)
		require.True(ok, "expected static dependency set")
		depSet = cast
	}

	supernode, l2ACL, l2BCL := startTwoL2SharedSupernode(
		t,
		l1Net,
		l1EL,
		l1CL,
		l2ANet,
		l2AEL,
		l2BNet,
		l2BEL,
		depSet,
		interopActivationTimestamp,
		jwtSecret,
	)

	l2ABatcher := startMinimalBatcher(t, keys, l2ANet, l1EL, l2ACL, l2AEL)
	l2AProposer := startMinimalProposer(t, keys, l2ANet, l1EL, l2ACL)
	l2BBatcher := startMinimalBatcher(t, keys, l2BNet, l1EL, l2BCL, l2BEL)
	l2BProposer := startMinimalProposer(t, keys, l2BNet, l1EL, l2BCL)

	faucetService := startFaucetsForRPCs(t, keys, map[eth.ChainID]string{
		l1Net.ChainID():  l1EL.UserRPC(),
		l2ANet.ChainID(): l2AEL.UserRPC(),
		l2BNet.ChainID(): l2BEL.UserRPC(),
	})

	return &MultiChainRuntime{
		Keys:          keys,
		Migration:     newInteropMigrationState(wb),
		DependencySet: wb.outFullCfgSet.DependencySet,
		L1Network:     l1Net,
		L1EL:          l1EL,
		L1CL:          l1CL,
		Chains: map[string]*MultiChainNodeRuntime{
			"l2a": {
				Name:     "l2a",
				Network:  l2ANet,
				EL:       l2AEL,
				CL:       l2ACL,
				Batcher:  l2ABatcher,
				Proposer: l2AProposer,
			},
			"l2b": {
				Name:     "l2b",
				Network:  l2BNet,
				EL:       l2BEL,
				CL:       l2BCL,
				Batcher:  l2BBatcher,
				Proposer: l2BProposer,
			},
		},
		Supernode:     supernode,
		FaucetService: faucetService,
		TimeTravel:    timeTravelClock,
	}, activationTime
}

func buildTwoL2RuntimeWorld(t devtest.T, keys devkeys.Keys, enableInterop bool, localContractArtifactsPath string, deployerOpts ...DeployerOption) (*worldBuilder, *L1Network, *L2Network, *L2Network) {
	wb := &worldBuilder{
		p:       t,
		logger:  t.Logger(),
		require: t.Require(),
		keys:    keys,
		builder: intentbuilder.New(),
	}

	applyConfigLocalContractSources(t, keys, wb.builder, localContractArtifactsPath)
	applyConfigCommons(t, keys, DefaultL1ID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2AID, wb.builder)
	applyConfigPrefundedL2(t, keys, DefaultL1ID, DefaultL2BID, wb.builder)
	if enableInterop {
		for _, l2Cfg := range wb.builder.L2s() {
			l2Cfg.WithForkAtGenesis(opforks.Interop)
		}
	}
	applyConfigDeployerOptions(t, keys, wb.builder, deployerOpts)
	wb.Build()

	t.Require().Len(wb.l2Chains, 2, "expected exactly two L2 chains in TwoL2 world")
	l1ID := eth.ChainIDFromUInt64(wb.output.AppliedIntent.L1ChainID)

	l1Net := &L1Network{
		name:      "l1",
		chainID:   l1ID,
		genesis:   wb.outL1Genesis,
		blockTime: 6,
	}

	l2ANet := l2NetworkFromWorldBuilder(t, wb, l1ID, DefaultL2AID, keys)
	l2BNet := l2NetworkFromWorldBuilder(t, wb, l1ID, DefaultL2BID, keys)

	return wb, l1Net, l2ANet, l2BNet
}

func l2NetworkFromWorldBuilder(t devtest.T, wb *worldBuilder, l1ChainID, l2ChainID eth.ChainID, keys devkeys.Keys) *L2Network {
	require := t.Require()

	l2Genesis, ok := wb.outL2Genesis[l2ChainID]
	require.Truef(ok, "missing L2 genesis for chain %s", l2ChainID)
	l2RollupCfg, ok := wb.outL2RollupCfg[l2ChainID]
	require.Truef(ok, "missing L2 rollup config for chain %s", l2ChainID)
	l2Dep, ok := wb.outL2Deployment[l2ChainID]
	require.Truef(ok, "missing L2 deployment for chain %s", l2ChainID)

	return &L2Network{
		name:       map[eth.ChainID]string{DefaultL2AID: "l2a", DefaultL2BID: "l2b"}[l2ChainID],
		chainID:    l2ChainID,
		l1ChainID:  l1ChainID,
		genesis:    l2Genesis,
		rollupCfg:  l2RollupCfg,
		deployment: l2Dep,
		opcmImpl:   wb.output.ImplementationsDeployment.OpcmImpl,
		mipsImpl:   wb.output.ImplementationsDeployment.MipsImpl,
		keys:       keys,
	}
}

func addMultiChainFollowL2Node(t devtest.T, runtime *MultiChainRuntime, chainKey string, name string) *SingleChainNodeRuntime {
	chain := runtime.Chains[chainKey]
	t.Require().NotNil(chain, "missing %s runtime chain", chainKey)
	t.Require().NotNil(chain.CL, "%s runtime chain missing CL follow source", chainKey)

	jwtPath := chain.EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)
	l2EL := startL2ELNode(t, chain.Network, jwtPath, jwtSecret, name, NewELNodeIdentity(0))
	l2CL := startL2CLNode(t, runtime.Keys, runtime.L1Network, chain.Network, runtime.L1EL, runtime.L1CL, l2EL, jwtSecret, l2CLNodeStartConfig{
		Key:            name,
		IsSequencer:    false,
		NoDiscovery:    true,
		EnableReqResp:  false,
		UseReqResp:     false,
		L2FollowSource: chain.CL.UserRPC(),
		DependencySet:  runtime.DependencySet,
	})

	connectL2ELPeers(t, t.Logger(), chain.EL.UserRPC(), l2EL.UserRPC(), false)
	connectL2CLPeers(t, t.Logger(), chain.CL, l2CL)

	node := &SingleChainNodeRuntime{
		Name:        name,
		IsSequencer: false,
		EL:          l2EL,
		CL:          l2CL,
	}
	if chain.Followers == nil {
		chain.Followers = make(map[string]*SingleChainNodeRuntime)
	}
	chain.Followers[name] = node
	return node
}

func startTwoL2SharedSupernode(
	t devtest.T,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	l2ANet *L2Network,
	l2AEL *OpGeth,
	l2BNet *L2Network,
	l2BEL *OpGeth,
	depSet *depset.StaticConfigDependencySet,
	interopActivationTimestamp *uint64,
	jwtSecret [32]byte,
) (*SuperNode, *SuperNodeProxy, *SuperNodeProxy) {
	require := t.Require()
	logger := t.Logger().New("component", "supernode")
	makeNodeCfg := func(l2Net *L2Network, l2EL L2ELNode) *opnodeconfig.Config {
		p2pKey, err := l2Net.keys.Secret(devkeys.SequencerP2PRole.Key(l2Net.ChainID().ToBig()))
		require.NoError(err, "need p2p key for supernode virtual sequencer")
		p2pConfig, p2pSignerSetup := newDevstackP2PConfig(
			t,
			logger.New("chain_id", l2Net.ChainID().String(), "component", "supernode-p2p"),
			l2Net.rollupCfg.BlockTime,
			false,
			true,
			hex.EncodeToString(crypto.FromECDSA(p2pKey)),
		)
		cfg := &opnodeconfig.Config{
			L1: &opnodeconfig.L1EndpointConfig{
				L1NodeAddr:       l1EL.UserRPC(),
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: time.Millisecond * 100,
				MaxConcurrency:   10,
				CacheSize:        0,
			},
			L1ChainConfig: l1Net.genesis.Config,
			L2: &opnodeconfig.L2EndpointConfig{
				L2EngineAddr:      l2EL.EngineRPC(),
				L2EngineJWTSecret: jwtSecret,
			},
			DependencySet:                   depSet,
			Beacon:                          &opnodeconfig.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
			Driver:                          driver.Config{SequencerEnabled: true, SequencerConfDepth: 2},
			Rollup:                          *l2Net.rollupCfg,
			P2PSigner:                       p2pSignerSetup,
			RPC:                             oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
			InteropConfig:                   &interop.Config{},
			P2P:                             p2pConfig,
			L1EpochPollInterval:             2 * time.Second,
			RuntimeConfigReloadInterval:     0,
			Sync:                            nodeSync.Config{SyncMode: nodeSync.CLSync, SyncModeReqResp: true},
			ConfigPersistence:               opnodeconfig.DisabledConfigPersistence{},
			Metrics:                         opmetrics.CLIConfig{},
			Pprof:                           oppprof.CLIConfig{},
			IgnoreMissingPectraBlobSchedule: false,
			ExperimentalOPStackAPI:          true,
		}
		require.NoError(cfg.Check(), "invalid supernode op-node config for chain %s", l2Net.ChainID())
		return cfg
	}

	vnCfgs := map[eth.ChainID]*opnodeconfig.Config{
		l2ANet.ChainID(): makeNodeCfg(l2ANet, l2AEL),
		l2BNet.ChainID(): makeNodeCfg(l2BNet, l2BEL),
	}
	chainIDs := []uint64{eth.EvilChainIDToUInt64(l2ANet.ChainID()), eth.EvilChainIDToUInt64(l2BNet.ChainID())}

	snCfg := &snconfig.CLIConfig{
		Chains:                     chainIDs,
		DataDir:                    t.TempDir(),
		L1NodeAddr:                 l1EL.UserRPC(),
		L1BeaconAddr:               l1CL.beaconHTTPAddr,
		RPCConfig:                  oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
		InteropActivationTimestamp: interopActivationTimestamp,
	}

	supernode := &SuperNode{
		userRPC:          "",
		interopEndpoint:  "",
		interopJwtSecret: jwtSecret,
		p:                t,
		logger:           logger,
		chains:           []eth.ChainID{l2ANet.ChainID(), l2BNet.ChainID()},
		l1UserRPC:        l1EL.UserRPC(),
		l1BeaconAddr:     l1CL.beaconHTTPAddr,
		snCfg:            snCfg,
		vnCfgs:           vnCfgs,
	}
	supernode.Start()
	t.Cleanup(supernode.Stop)

	base := supernode.UserRPC()
	l2ARPC := base + "/" + strconv.FormatUint(eth.EvilChainIDToUInt64(l2ANet.ChainID()), 10)
	l2BRPC := base + "/" + strconv.FormatUint(eth.EvilChainIDToUInt64(l2BNet.ChainID()), 10)

	waitForSupernodeRoute(t, logger, l2ARPC)
	waitForSupernodeRoute(t, logger, l2BRPC)

	l2ACL := &SuperNodeProxy{
		p:                t,
		logger:           logger,
		userRPC:          l2ARPC,
		interopEndpoint:  l2ARPC,
		interopJwtSecret: jwtSecret,
	}
	l2BCL := &SuperNodeProxy{
		p:                t,
		logger:           logger,
		userRPC:          l2BRPC,
		interopEndpoint:  l2BRPC,
		interopJwtSecret: jwtSecret,
	}

	return supernode, l2ACL, l2BCL
}

func startSingleChainSharedSupernode(
	t devtest.T,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	l2Net *L2Network,
	l2EL *OpGeth,
	depSet *depset.StaticConfigDependencySet,
	jwtSecret [32]byte,
	interopAtGenesis bool,
) (*SuperNode, *SuperNodeProxy) {
	require := t.Require()
	logger := t.Logger().New("component", "supernode")
	makeNodeCfg := func() *opnodeconfig.Config {
		p2pKey, err := l2Net.keys.Secret(devkeys.SequencerP2PRole.Key(l2Net.ChainID().ToBig()))
		require.NoError(err, "need p2p key for supernode virtual sequencer")
		p2pConfig, p2pSignerSetup := newDevstackP2PConfig(
			t,
			logger.New("chain_id", l2Net.ChainID().String(), "component", "supernode-p2p"),
			l2Net.rollupCfg.BlockTime,
			false,
			true,
			hex.EncodeToString(crypto.FromECDSA(p2pKey)),
		)
		cfg := &opnodeconfig.Config{
			L1: &opnodeconfig.L1EndpointConfig{
				L1NodeAddr:       l1EL.UserRPC(),
				L1TrustRPC:       false,
				L1RPCKind:        sources.RPCKindDebugGeth,
				RateLimit:        0,
				BatchSize:        20,
				HttpPollInterval: 100 * time.Millisecond,
				MaxConcurrency:   10,
				CacheSize:        0,
			},
			L1ChainConfig: l1Net.genesis.Config,
			L2: &opnodeconfig.L2EndpointConfig{
				L2EngineAddr:      l2EL.EngineRPC(),
				L2EngineJWTSecret: jwtSecret,
			},
			DependencySet:                   depSet,
			Beacon:                          &opnodeconfig.L1BeaconEndpointConfig{BeaconAddr: l1CL.beaconHTTPAddr},
			Driver:                          driver.Config{SequencerEnabled: true, SequencerConfDepth: 2},
			Rollup:                          *l2Net.rollupCfg,
			P2PSigner:                       p2pSignerSetup,
			RPC:                             oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
			InteropConfig:                   &interop.Config{},
			P2P:                             p2pConfig,
			L1EpochPollInterval:             2 * time.Second,
			Sync:                            nodeSync.Config{SyncMode: nodeSync.CLSync, SyncModeReqResp: true},
			ConfigPersistence:               opnodeconfig.DisabledConfigPersistence{},
			Metrics:                         opmetrics.CLIConfig{},
			Pprof:                           oppprof.CLIConfig{},
			ExperimentalOPStackAPI:          true,
			IgnoreMissingPectraBlobSchedule: false,
		}
		require.NoError(cfg.Check(), "invalid supernode op-node config for chain %s", l2Net.ChainID())
		return cfg
	}

	var interopActivationTimestamp *uint64
	if interopAtGenesis {
		ts := l2Net.rollupCfg.Genesis.L2Time
		interopActivationTimestamp = &ts
	}

	snCfg := &snconfig.CLIConfig{
		Chains:                     []uint64{eth.EvilChainIDToUInt64(l2Net.ChainID())},
		DataDir:                    t.TempDir(),
		L1NodeAddr:                 l1EL.UserRPC(),
		L1BeaconAddr:               l1CL.beaconHTTPAddr,
		RPCConfig:                  oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0, EnableAdmin: true},
		InteropActivationTimestamp: interopActivationTimestamp,
	}

	supernode := &SuperNode{
		userRPC:          "",
		interopEndpoint:  "",
		interopJwtSecret: jwtSecret,
		p:                t,
		logger:           logger,
		chains:           []eth.ChainID{l2Net.ChainID()},
		l1UserRPC:        l1EL.UserRPC(),
		l1BeaconAddr:     l1CL.beaconHTTPAddr,
		snCfg:            snCfg,
		vnCfgs: map[eth.ChainID]*opnodeconfig.Config{
			l2Net.ChainID(): makeNodeCfg(),
		},
	}
	supernode.Start()
	t.Cleanup(supernode.Stop)

	l2RPC := supernode.UserRPC() + "/" + strconv.FormatUint(eth.EvilChainIDToUInt64(l2Net.ChainID()), 10)
	waitForSupernodeRoute(t, logger, l2RPC)

	return supernode, &SuperNodeProxy{
		p:                t,
		logger:           logger,
		userRPC:          l2RPC,
		interopEndpoint:  l2RPC,
		interopJwtSecret: jwtSecret,
	}
}

func waitForSupernodeRoute(t devtest.T, logger log.Logger, rpcEndpoint string) {
	deadline := time.Now().Add(15 * time.Second)
	for {
		if time.Now().After(deadline) {
			t.Require().FailNowf("supernode route readiness", "timed out waiting for supernode route %s", rpcEndpoint)
		}

		rpcCl, err := client.NewRPC(t.Ctx(), logger, rpcEndpoint, client.WithLazyDial())
		if err == nil {
			var out any
			callErr := rpcCl.CallContext(t.Ctx(), &out, "optimism_rollupConfig")
			rpcCl.Close()
			if callErr == nil {
				return
			}
		}

		time.Sleep(200 * time.Millisecond)
	}
}

type l2TestSequencerTarget struct {
	chainID eth.ChainID
	l2EL    *OpGeth
	l2CL    L2CLNode
}

func attachTestSequencerToRuntime(t devtest.T, runtime *MultiChainRuntime, testSequencerName string) {
	t.Require().NotEmpty(runtime.Chains, "runtime must contain at least one chain")

	chainKeys := make([]string, 0, len(runtime.Chains))
	for key := range runtime.Chains {
		chainKeys = append(chainKeys, key)
	}
	sort.Strings(chainKeys)

	firstChain := runtime.Chains[chainKeys[0]]
	t.Require().NotNil(firstChain, "missing runtime chain %s", chainKeys[0])
	jwtPath := firstChain.EL.JWTPath()
	jwtSecret := readJWTSecretFromPath(t, jwtPath)

	targets := make([]l2TestSequencerTarget, 0, len(chainKeys))
	for _, key := range chainKeys {
		chain := runtime.Chains[key]
		t.Require().NotNil(chain, "missing runtime chain %s", key)
		l2EL, ok := chain.EL.(*OpGeth)
		t.Require().True(ok, "runtime chain %s must use op-geth for test sequencer", key)
		targets = append(targets, l2TestSequencerTarget{
			chainID: chain.Network.ChainID(),
			l2EL:    l2EL,
			l2CL:    chain.CL,
		})
	}

	testSequencer := startTestSequencerForL2Chains(
		t,
		runtime.Keys,
		testSequencerName,
		jwtPath,
		jwtSecret,
		runtime.L1Network,
		runtime.L1EL,
		runtime.L1CL,
		targets,
	)
	runtime.TestSequencer = newTestSequencerRuntime(testSequencer, "")
}

func startTestSequencerForL2Chains(
	t devtest.T,
	keys devkeys.Keys,
	testSequencerName string,
	jwtPath string,
	jwtSecret [32]byte,
	l1Net *L1Network,
	l1EL *L1Geth,
	l1CL *L1CLNode,
	targets []l2TestSequencerTarget,
) *testSequencer {
	require := t.Require()
	logger := t.Logger().New("component", "test-sequencer")

	require.NotEmpty(targets, "at least one L2 target is required")

	l1ELClient, err := ethclient.DialContext(t.Ctx(), l1EL.UserRPC())
	require.NoError(err, "failed to dial L1 EL RPC for test-sequencer")
	t.Cleanup(l1ELClient.Close)

	engineCl, err := dialEngine(t.Ctx(), l1EL.AuthRPC(), jwtSecret)
	require.NoError(err, "failed to dial L1 engine API for test-sequencer")
	t.Cleanup(func() {
		engineCl.inner.Close()
	})

	l1ChainID := l1Net.ChainID()
	bidL1 := seqtypes.BuilderID("test-l1-builder")
	cidL1 := seqtypes.CommitterID("test-noop-committer")
	sidL1 := seqtypes.SignerID("test-noop-signer")
	pidL1 := seqtypes.PublisherID("test-noop-publisher")
	seqIDL1 := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", l1ChainID))

	ensemble := &workconfig.Ensemble{
		Builders: map[seqtypes.BuilderID]*workconfig.BuilderEntry{
			bidL1: {
				L1: &fakepos.Config{
					ChainConfig:       l1Net.genesis.Config,
					EngineAPI:         engineCl,
					Backend:           l1ELClient,
					Beacon:            l1CL.beacon,
					FinalizedDistance: 20,
					SafeDistance:      10,
					BlockTime:         6,
				},
			},
		},
		Signers: map[seqtypes.SignerID]*workconfig.SignerEntry{
			sidL1: {
				Noop: &noopsigner.Config{},
			},
		},
		Committers: map[seqtypes.CommitterID]*workconfig.CommitterEntry{
			cidL1: {
				Noop: &noopcommitter.Config{},
			},
		},
		Publishers: map[seqtypes.PublisherID]*workconfig.PublisherEntry{
			pidL1: {
				Noop: &nooppublisher.Config{},
			},
		},
		Sequencers: map[seqtypes.SequencerID]*workconfig.SequencerEntry{
			seqIDL1: {
				Full: &fullseq.Config{
					ChainID:   l1ChainID,
					Builder:   bidL1,
					Signer:    sidL1,
					Committer: cidL1,
					Publisher: pidL1,
				},
			},
		},
	}

	sequencerIDs := map[eth.ChainID]seqtypes.SequencerID{
		l1ChainID: seqIDL1,
	}

	for i, target := range targets {
		suffix := ""
		if len(targets) > 1 {
			suffix = fmt.Sprintf("-%c", 'A'+i)
		}

		bid := seqtypes.BuilderID(fmt.Sprintf("test-standard-builder%s", suffix))
		cid := seqtypes.CommitterID(fmt.Sprintf("test-standard-committer%s", suffix))
		sid := seqtypes.SignerID(fmt.Sprintf("test-local-signer%s", suffix))
		pid := seqtypes.PublisherID(fmt.Sprintf("test-standard-publisher%s", suffix))
		seqID := seqtypes.SequencerID(fmt.Sprintf("test-seq-%s", target.chainID))

		p2pKey, err := keys.Secret(devkeys.SequencerP2PRole.Key(target.chainID.ToBig()))
		require.NoError(err, "need p2p key for test sequencer target %d", i)
		rawKey := hexutil.Bytes(crypto.FromECDSA(p2pKey))

		ensemble.Builders[bid] = &workconfig.BuilderEntry{
			Standard: &standardbuilder.Config{
				L1ChainConfig: l1Net.genesis.Config,
				L1EL:          endpoint.MustRPC{Value: endpoint.HttpURL(l1EL.UserRPC())},
				L2EL:          endpoint.MustRPC{Value: endpoint.HttpURL(target.l2EL.UserRPC())},
				L2CL:          endpoint.MustRPC{Value: endpoint.HttpURL(target.l2CL.UserRPC())},
			},
		}
		ensemble.Signers[sid] = &workconfig.SignerEntry{
			LocalKey: &localkey.Config{RawKey: &rawKey, ChainID: target.chainID},
		}
		ensemble.Committers[cid] = &workconfig.CommitterEntry{
			Standard: &standardcommitter.Config{RPC: endpoint.MustRPC{Value: endpoint.HttpURL(target.l2CL.UserRPC())}},
		}
		ensemble.Publishers[pid] = &workconfig.PublisherEntry{
			Standard: &standardpublisher.Config{RPC: endpoint.MustRPC{Value: endpoint.HttpURL(target.l2CL.UserRPC())}},
		}
		ensemble.Sequencers[seqID] = &workconfig.SequencerEntry{
			Full: &fullseq.Config{
				ChainID:             target.chainID,
				Builder:             bid,
				Signer:              sid,
				Committer:           cid,
				Publisher:           pid,
				SequencerConfDepth:  2,
				SequencerEnabled:    true,
				SequencerStopped:    false,
				SequencerMaxSafeLag: 0,
			},
		}

		sequencerIDs[target.chainID] = seqID
	}

	jobs := work.NewJobRegistry()
	startedEnsemble, err := ensemble.Start(t.Ctx(), &work.StartOpts{
		Log:     logger,
		Metrics: &testmetrics.NoopMetrics{},
		Jobs:    jobs,
	})
	require.NoError(err, "failed to start test-sequencer ensemble")

	cfg := &sequencerConfig.Config{
		MetricsConfig: opmetrics.CLIConfig{Enabled: false},
		PprofConfig:   oppprof.CLIConfig{ListenEnabled: false},
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelDebug,
			Format: oplog.FormatText,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		Ensemble:      startedEnsemble,
		JWTSecretPath: jwtPath,
		Version:       "dev",
		MockRun:       false,
	}

	sq, err := sequencer.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "failed to initialize test-sequencer service")
	require.NoError(sq.Start(t.Ctx()), "failed to start test-sequencer service")

	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(t.Ctx())
		cancel()
		logger.Info("Closing test-sequencer service")
		closeErr := sq.Stop(ctx)
		logger.Info("Closed test-sequencer service", "err", closeErr)
	})

	adminRPC := sq.RPC()
	controlRPCs := make(map[eth.ChainID]string, len(sequencerIDs))
	for chainID, seqID := range sequencerIDs {
		controlRPCs[chainID] = adminRPC + "/sequencers/" + seqID.String()
	}

	return &testSequencer{
		name:       testSequencerName,
		adminRPC:   adminRPC,
		jwtSecret:  jwtSecret,
		controlRPC: controlRPCs,
		service:    sq,
	}
}

func startFaucetsForRPCs(t devtest.T, keys devkeys.Keys, chainRPCs map[eth.ChainID]string) *faucet.Service {
	require := t.Require()
	logger := t.Logger().New("component", "faucet")

	funderKey, err := keys.Secret(devkeys.UserKey(funderMnemonicIndex))
	require.NoError(err, "need faucet funder key")
	funderKeyStr := hexutil.Encode(crypto.FromECDSA(funderKey))

	faucets := make(map[ftypes.FaucetID]*fconf.FaucetEntry, len(chainRPCs))
	for chainID, rpcURL := range chainRPCs {
		faucetID := ftypes.FaucetID(fmt.Sprintf("dev-faucet-%s", chainID))
		faucets[faucetID] = &fconf.FaucetEntry{
			ELRPC:   endpoint.MustRPC{Value: endpoint.URL(rpcURL)},
			ChainID: chainID,
			TxCfg: fconf.TxManagerConfig{
				PrivateKey: funderKeyStr,
			},
		}
	}

	cfg := &faucetConfig.Config{
		RPC: oprpc.CLIConfig{
			ListenAddr: "127.0.0.1",
		},
		Faucets: &fconf.Config{
			Faucets: faucets,
		},
	}

	srv, err := faucet.FromConfig(t.Ctx(), cfg, logger)
	require.NoError(err, "failed to create faucet service")
	require.NoError(srv.Start(t.Ctx()), "failed to start faucet service")

	t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // force-close
		logger.Info("Closing faucet service")
		closeErr := srv.Stop(ctx)
		logger.Info("Closed faucet service", "err", closeErr)
	})

	return srv
}
