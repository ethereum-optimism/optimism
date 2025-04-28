package p2p

import (
	"context"
	"math/big"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	"github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestSystemP2PAltSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	op_e2e.InitParallel(t)

	cfg := e2esys.DefaultSystemConfig(t)

	// remove default verifier node
	delete(cfg.Nodes, "verifier")
	// Add more verifier nodes
	cfg.Nodes["alice"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Nodes["bob"] = &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   false,
		},
		L1EpochPollInterval: time.Second * 4,
	}
	cfg.Loggers["alice"] = testlog.Logger(t, log.LevelInfo).New("role", "alice")
	cfg.Loggers["bob"] = testlog.Logger(t, log.LevelInfo).New("role", "bob")

	// connect the nodes
	cfg.P2PTopology = map[string][]string{
		"sequencer": {"alice", "bob"},
		"alice":     {"sequencer", "bob"},
		"bob":       {"alice", "sequencer"},
	}
	// Enable the P2P req-resp based sync
	cfg.P2PReqRespSync = true

	// Disable batcher, so there will not be any L1 data to sync from
	cfg.DisableBatcher = true

	var published []string
	seqTracer := new(opnode.FnTracer)
	// The sequencer still publishes the blocks to the tracer, even if they do not reach the network due to disabled P2P
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		published = append(published, payload.ExecutionPayload.ID().String())
	}
	// Blocks are now received via the RPC based alt-sync method
	cfg.Nodes["sequencer"].Tracer = seqTracer

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit a TX to L2 sequencer node
	receiptSeq := helpers.SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *helpers.TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)
	})

	// Gossip is able to respond to IWANT messages for the duration of heartbeat_time * message_window = 0.5 * 12 = 6
	// Wait till we pass that, and then we'll have missed some blocks that cannot be retrieved in any way from gossip
	time.Sleep(time.Second * 10)

	// set up our syncer node, connect it to alice/bob
	cfg.Loggers["syncer"] = testlog.Logger(t, log.LevelInfo).New("role", "syncer")

	// Create a peer, and hook up alice and bob
	h, err := sys.NewMockNetPeer()
	require.NoError(t, err)
	_, err = sys.Mocknet.LinkPeers(sys.RollupNodes["alice"].P2P().Host().ID(), h.ID())
	require.NoError(t, err)
	_, err = sys.Mocknet.LinkPeers(sys.RollupNodes["bob"].P2P().Host().ID(), h.ID())
	require.NoError(t, err)

	// Configure the new rollup node that'll be syncing
	var syncedPayloads []string
	syncNodeCfg := &rollupNode.Config{
		Driver:    driver.Config{VerifierConfDepth: 0},
		Rollup:    *sys.RollupConfig,
		P2PSigner: nil,
		RPC: rollupNode.RPCConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P:                 &p2p.Prepared{HostP2P: h, EnableReqRespSync: true},
		Metrics:             rollupNode.MetricsConfig{Enabled: false}, // no metrics server
		Pprof:               oppprof.CLIConfig{},
		L1EpochPollInterval: time.Second * 10,
		Tracer: &opnode.FnTracer{
			OnUnsafeL2PayloadFn: func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope) {
				syncedPayloads = append(syncedPayloads, payload.ExecutionPayload.ID().String())
			},
		},
	}
	e2esys.ConfigureL1(syncNodeCfg, sys.EthInstances["l1"], sys.L1BeaconEndpoint())
	syncerL2Engine, err := geth.InitL2("syncer", sys.L2GenesisCfg, cfg.JWTFilePath)
	require.NoError(t, err)
	require.NoError(t, syncerL2Engine.Node.Start())

	e2esys.ConfigureL2(syncNodeCfg, syncerL2Engine, cfg.JWTSecret)

	syncerNode, err := rollupNode.New(ctx, syncNodeCfg, cfg.Loggers["syncer"], "", metrics.NewMetrics(""))
	require.NoError(t, err)
	err = syncerNode.Start(ctx)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, syncerNode.Stop(ctx))
	}()

	// connect alice and bob to our new syncer node
	_, err = sys.Mocknet.ConnectPeers(sys.RollupNodes["alice"].P2P().Host().ID(), syncerNode.P2P().Host().ID())
	require.NoError(t, err)
	_, err = sys.Mocknet.ConnectPeers(sys.RollupNodes["bob"].P2P().Host().ID(), syncerNode.P2P().Host().ID())
	require.NoError(t, err)

	rpc := syncerL2Engine.UserRPC().(endpoint.ClientRPC).ClientRPC()
	l2Verif := ethclient.NewClient(rpc)

	// It may take a while to sync, but eventually we should see the sequenced data show up
	receiptVerif, err := wait.ForReceiptOK(ctx, l2Verif, receiptSeq.TxHash)
	require.Nil(t, err, "Waiting for L2 tx on verifier")

	require.Equal(t, receiptSeq, receiptVerif)

	// Verify that the tx was received via P2P sync
	require.Contains(t, syncedPayloads, eth.BlockID{Hash: receiptVerif.BlockHash, Number: receiptVerif.BlockNumber.Uint64()}.String())

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(syncedPayloads))
	require.Subset(t, published, syncedPayloads)
}
