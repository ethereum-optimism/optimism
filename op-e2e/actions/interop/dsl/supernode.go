package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	opnodeinterop "github.com/ethereum-optimism/optimism/op-node/rollup/interop"
	opnodesync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oppprof "github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	supernodecfg "github.com/ethereum-optimism/optimism/op-supernode/config"
	opsupernode "github.com/ethereum-optimism/optimism/op-supernode/supernode"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

const interopL1BeaconSlotTime = uint64(6)

var _ apis.SupernodeQueryAPI = (*SuperNodeActor)(nil)

// SuperNodeActor exposes a real op-supernode instance to interop action tests.
type SuperNodeActor struct {
	*sources.SuperNodeClient
}

func NewSuperNode(t helpers.Testing, logger log.Logger, l1Miner *helpers.L1Miner, chains ...*Chain) *SuperNodeActor {
	logger = logger.New("role", "supernode")
	supernodeDataDir := t.TempDir()
	logger.Info("supernode data dir", "dir", supernodeDataDir)
	beacon := newSuperNodeBeacon(t, logger, l1Miner)
	cfg := newSuperNodeConfig(t, l1Miner, beacon, supernodeDataDir, chains)
	vnCfgs := newVirtualNodeConfigs(l1Miner, beacon.BeaconAddr(), chains)

	runCtx, cancel := context.WithCancelCause(t.Ctx())
	service, err := opsupernode.New(runCtx, logger.New("service", "op-supernode"), "test", cancel, cfg, vnCfgs)
	require.NoError(t, err)
	require.NoError(t, service.Start(runCtx))

	rpcAddr, err := service.WaitRPCAddr(runCtx)
	require.NoError(t, err)

	client, err := dial.DialSuperNodeClientWithTimeout(runCtx, logger, "http://"+rpcAddr)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = beacon.Close()
	})
	t.Cleanup(func() {
		cancel(context.Canceled)
		require.NoError(t, service.Stop(context.Background()))
	})
	t.Cleanup(client.Close)

	return &SuperNodeActor{
		SuperNodeClient: client,
	}
}

func newSuperNodeBeacon(t helpers.Testing, logger log.Logger, l1Miner *helpers.L1Miner) *fakebeacon.FakeBeacon {
	genesisBlock := l1Miner.L1Chain().GetBlockByNumber(0)
	require.NotNil(t, genesisBlock)

	beacon := fakebeacon.NewBeacon(
		logger.New("component", "fake-beacon"),
		l1Miner.BlobStore(),
		genesisBlock.Time(),
		interopL1BeaconSlotTime,
	)
	require.NoError(t, beacon.Start("127.0.0.1:0"))
	return beacon
}

func newSuperNodeConfig(t helpers.Testing, l1Miner *helpers.L1Miner, beacon *fakebeacon.FakeBeacon, dataDir string, chains []*Chain) *supernodecfg.CLIConfig {
	chainIDs := make([]uint64, 0, len(chains))
	for _, chain := range chains {
		chainID, ok := chain.ChainID.Uint64()
		require.True(t, ok, "chain ID must fit in uint64")
		chainIDs = append(chainIDs, chainID)
	}

	cfg := &supernodecfg.CLIConfig{
		Chains:                     chainIDs,
		DataDir:                    dataDir,
		L1NodeAddr:                 l1Miner.WSEndpoint(),
		L1BeaconAddr:               beacon.BeaconAddr(),
		RPCConfig:                  oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0},
		MetricsConfig:              opmetrics.DefaultCLIConfig(),
		PprofConfig:                oppprof.DefaultCLIConfig(),
		InteropActivationTimestamp: sharedInteropActivationTimestamp(t, chains),
	}
	require.NoError(t, cfg.Check())
	return cfg
}

func newVirtualNodeConfigs(l1Miner *helpers.L1Miner, beaconAddr string, chains []*Chain) map[eth.ChainID]*opnodecfg.Config {
	vnCfgs := make(map[eth.ChainID]*opnodecfg.Config, len(chains))
	for _, chain := range chains {
		vnCfgs[chain.ChainID] = &opnodecfg.Config{
			L1: &opnodecfg.PreparedL1Endpoint{
				Client:          l1Miner.RPCClient(),
				TrustRPC:        false,
				RPCProviderKind: sources.RPCKindDebugGeth,
			},
			L2: &opnodecfg.PreparedL2Endpoints{
				Client: chain.SequencerEngine.RPCClient(),
			},
			Beacon: &opnodecfg.L1BeaconEndpointConfig{
				BeaconAddr: beaconAddr,
			},
			Driver:                      driver.Config{SequencerEnabled: false},
			Rollup:                      *chain.RollupCfg,
			L1ChainConfig:               chain.L1ChainConfig,
			DependencySet:               chain.DependencySet,
			InteropConfig:               &opnodeinterop.Config{},
			RPC:                         oprpc.CLIConfig{ListenAddr: "127.0.0.1", ListenPort: 0},
			L1EpochPollInterval:         250 * time.Millisecond,
			ConfigPersistence:           opnodecfg.DisabledConfigPersistence{},
			RuntimeConfigReloadInterval: 0,
			Sync: opnodesync.Config{
				SyncMode:                       opnodesync.CLSync,
				SkipSyncStartCheck:             false,
				SupportsPostFinalizationELSync: false,
			},
		}
	}
	return vnCfgs
}

func sharedInteropActivationTimestamp(t helpers.Testing, chains []*Chain) *uint64 {
	if len(chains) == 0 {
		return nil
	}

	var interopTime *uint64
	for _, chain := range chains {
		resolvedInteropTime := chain.RollupCfg.InteropTime
		if resolvedInteropTime != nil && *resolvedInteropTime == 0 {
			ts := chain.RollupCfg.Genesis.L2Time
			resolvedInteropTime = &ts
		}
		switch {
		case interopTime == nil && resolvedInteropTime == nil:
			continue
		case interopTime == nil:
			ts := *resolvedInteropTime
			interopTime = &ts
		default:
			require.NotNilf(t, resolvedInteropTime, "interop chains disagree on interop activation", "chain %v is missing an interop activation timestamp", chain.ChainID)
			require.Equalf(t, *interopTime, *resolvedInteropTime, "interop chains disagree on interop activation", "chain %v has interop activation %d, expected %d", chain.ChainID, *resolvedInteropTime, *interopTime)
		}
	}

	return interopTime
}
