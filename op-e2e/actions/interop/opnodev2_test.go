package interop

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-e2e/actions/helpers"
	"github.com/ethereum-optimism/optimism/op-e2e/actions/interop/dsl"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/fakebeacon"
	cfgv2 "github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	opv2 "github.com/ethereum-optimism/optimism/op-node/opnv2/service"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend"
	"github.com/ethereum-optimism/optimism/op-service/event"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

func TestOpnodeV2(gt *testing.T) {
	t := helpers.NewDefaultTesting(gt)
	is := dsl.SetupInterop(t)
	actors := is.CreateActors()
	logger := is.Log
	ctx := t.Ctx()

	// Wrap blob store with a beacon endpoint,
	// in v2 we avoid the in-process RPC for simpler setup.
	beaconNode := fakebeacon.NewBeacon(logger, actors.L1Miner.BlobStore(),
		is.Out.L1.Genesis.Timestamp, 12)
	require.NoError(t, beaconNode.Start("127.0.0.1:0"))
	t.Cleanup(func() {
		beaconNode.Close()
	})

	// Setup connections and configs
	l1Cfg := cfgv2.DefaultL1EndpointConfig()
	l1Cfg.L1NodeAddr = actors.L1Miner.HTTPEndpoint()

	beaconCfg := cfgv2.DefaultBeaconEndpointConfig()
	beaconCfg.BeaconAddr = beaconNode.BeaconAddr()

	cfgSetV2 := depset.StaticRollupConfigSetV2{}
	for _, chainID := range is.CfgSet.Chains() {
		cfgSetV2[chainID] = is.Out.L2s[chainID.String()].RollupCfg
	}
	depSet := is.CfgSet.DependencySet.(*depset.StaticConfigDependencySet)

	jwtPath := e2eutils.WriteDefaultJWT(t)
	engA := helpers.NewL2Engine(t, logger.New("role", "EngA"),
		is.Out.L2s[actors.ChainA.ChainID.String()].Genesis, jwtPath)
	engB := helpers.NewL2Engine(t, logger.New("role", "EngB"),
		is.Out.L2s[actors.ChainA.ChainID.String()].Genesis, jwtPath)

	engineAddrs := []string{
		engA.HTTPEndpoint(),
		engB.HTTPEndpoint(),
	}
	jwtPaths := []string{jwtPath}

	datadir := t.TempDir()

	cfg := &cfgv2.Config{
		Version:   "v0.0.0",
		LogConfig: oplog.CLIConfig{},
		MetricsConfig: opmetrics.CLIConfig{
			Enabled: false,
		},
		PprofConfig: oppprof.CLIConfig{
			ListenEnabled: false,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		L1:          l1Cfg,
		Beacon:      beaconCfg,
		L1ConfDepth: 0,
		L2: &cfgv2.L2ELsConfig{
			L2EngineAddrs:      engineAddrs,
			L2EngineJWTSecrets: jwtPaths,
			L2EngineRpcTimeout: time.Second * 10,
			L2ReadAddrs:        nil, // no read-only addrs yet
		},
		P2P:                   nil, // p2p disabled in action-tests
		RollupConfigSetSource: cfgSetV2,
		DependencySetSource:   depSet,
		SynchronousProcessors: true,
		Datadir:               datadir,
		Cancel: func(cause error) {
			t.Fatal("unexpected early exit", cause)
		},
	}

	// Create the node
	srv, err := opv2.FromConfig(ctx, cfg, logger.New("role", "opv2"))
	require.NoError(t, err)

	n := &OpnodeV2{
		Node:    srv,
		Backend: srv.Backend().(*backend.Backend),
		Drain:   srv.Backend().(*backend.Backend).Executor().(event.Drainer),
	}
	// Hook up the action test actor to the event system of the service,
	// so we can trigger/observe things.
	n.Backend.EventSystem().Register("action-test", n)

	require.NoError(t, srv.Start(ctx))
	t.Cleanup(func() {
		err := srv.Stop(ctx)
		if err != nil {
			t.Log(err)
		}
	})

	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainA.Sequencer.SyncSupervisor(t)
	actors.Supervisor.ProcessFull(t)
	actors.ChainA.Sequencer.ActL2PipelineFull(t)

	// Create an L1 block, then an L2 block, and submit the L2 block
	actors.L1Miner.ActEmptyBlock(t)
	actors.ChainA.Sequencer.ActL2PipelineFull(t)
	actors.ChainA.Sequencer.ActL2EmptyBlock(t)
	actors.ChainA.Batcher.ActSubmitAll(t)
	actors.L1Miner.ActL1StartBlock(12)(t)
	actors.L1Miner.ActL1IncludeTx(actors.ChainA.BatcherAddr)(t)
	actors.L1Miner.ActL1EndBlock(t)

	// Check if the op-node v2 can sync the block
	n.SignalLatestL1(t)
	n.ProcessEvents(t)
	latest, err := n.Backend.LocalUnsafe(t.Ctx(), actors.ChainA.ChainID)
	require.NoError(t, err)
	require.Equal(t, latest.Number, uint64(1), "must have synced a L2 block via L1")
}

type OpnodeV2 struct {
	Node    *opv2.Service
	Backend *backend.Backend
	Drain   event.Drainer
	Emitter event.Emitter
}

func (n *OpnodeV2) OnEvent(ctx context.Context, ev event.Event) bool {
	// anything interesting for testing we can intercept here
	return false
}

func (n *OpnodeV2) AttachEmitter(em event.Emitter) {
	n.Emitter = em
}

var _ event.AttachEmitter = (*OpnodeV2)(nil)
var _ event.Deriver = (*OpnodeV2)(nil)

// force the node to observe the latest L1 state
func (n *OpnodeV2) SignalLatestL1(t helpers.Testing) {
	require.NoError(t, n.Backend.PullLatestL1(t.Ctx()))
}

// force the node to observe the finalized L1 state
func (n *OpnodeV2) SignalFinalizedL1(t helpers.Testing) {
	require.NoError(t, n.Backend.PullFinalizedL1(t.Ctx()))
}

// make the node process the queue of events till it is empty
func (n *OpnodeV2) ProcessEvents(t helpers.Testing) {
	require.NoError(t, n.Drain.Drain())
}
