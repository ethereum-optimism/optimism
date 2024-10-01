package p2p

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"

	g "github.com/anacrolix/generics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"

	. "github.com/ethereum-optimism/optimism/op-e2e"
	. "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	. "github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	. "github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
)

type altSyncSequencerP2pConfig struct {
	p2p.SetupP2P
}

const sequencerOutboundQueueSize = 1

func (me altSyncSequencerP2pConfig) ConfigureGossip(rollupCfg *rollup.Config) []pubsub.Option {
	options := me.SetupP2P.ConfigureGossip(rollupCfg)
	options = append(options, pubsub.WithPeerOutboundQueueSize(1))
	return options
}

// Run this with -ethLogVerbosity=1. This tests many nodes requesting ranges they're missing that
// aren't available over gossip when only a few nodes have the actual blocks. TestSystemP2PAltSync
// is the baby version where there's only a single new syncer and many nodes with the blocks.
func TestSystemP2PAltSyncLongStall(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitParallel(t)

	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Add(-10 * time.Second).Unix())
	cfg.Nodes[RoleSeq].P2P = altSyncSequencerP2pConfig{cfg.Nodes[RoleSeq].P2P}

	// remove default verifier node
	delete(cfg.Nodes, RoleVerif)

	// This is needed to have SystemConfig.Start set up the sequencer's host for us.
	g.MakeMap(&cfg.P2PTopology)
	cfg.P2PTopology[RoleSeq] = nil

	// Enable the P2P req-resp based sync
	cfg.P2PReqRespSync.Enabled = true
	cfg.P2PReqRespSync.ConfigureClient = func(syncClient *p2p.SyncClient) {
		syncClient.NewPeerRateLimiter = newInfLimiter
	}
	cfg.P2PReqRespSync.ConfigureServer = func(syncClient *p2p.ReqRespServer) {
		syncClient.GlobalRequestsRL = newInfLimiter()
	}

	// Disable batcher, so there will not be any L1 data to sync from
	cfg.DisableBatcher = true

	var (
		publishedMu sync.Mutex
		published   []eth.BlockID
	)
	getPublished := func() []eth.BlockID {
		publishedMu.Lock()
		defer publishedMu.Unlock()
		return published
	}
	seqTracer := new(FnTracer)
	// The sequencer still publishes the blocks to the tracer, even if they do not reach the network due to disabled P2P
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		publishedMu.Lock()
		defer publishedMu.Unlock()
		published = append(published, payload.ExecutionPayload.ID())
	}
	// Blocks are now received via the RPC based alt-sync method
	cfg.Nodes[RoleSeq].Tracer = seqTracer

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.NodeClient(RoleSeq)

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit a TX to L2 sequencer node
	receiptSeq := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)
	})
	t.Logf("tx receipt is in block %v", receiptSeq.BlockNumber)

	assert.EqualValues(t, cfg.DeployConfig.L2BlockTime, 1)

	// Gossip has been limited to a single block, so now we wait until the block for which we hold
	// the receipt is no longer available in the sequencer's outbound gossip window.
	targetPublishedLen := receiptSeq.BlockNumber.Int64() + sequencerOutboundQueueSize + 1
	for {
		blocksPublished := int64(len(getPublished()))
		if blocksPublished >= targetPublishedLen {
			break
		}
		t.Logf("waiting for blocks published (%v) >= %v", blocksPublished, targetPublishedLen)
		time.Sleep(time.Second)
	}
	// Give time for the outbound gossip queues to lose messages.
	time.Sleep(time.Second)
	t.Logf("starting syncers after %v blocks published", len(getPublished()))

	var syncers []*syncerType

	sequencerNode := sys.RollupNodes[RoleSeq]

	addSyncer := func(i int) {
		name := fmt.Sprintf("syncer-%d", i)
		newSyncer := makeSyncer(ctx, t, name, cfg, sys)
		// Link to all the other syncers
		for _, syncer := range syncers {
			linkAndConnectNodeNamesNodes(t, sys.Mocknet, syncer.peerId, newSyncer.peerId)
		}
		// And to the sequencer.
		linkAndConnectNodeNamesNodes(t, sys.Mocknet, sequencerNode.P2P().Host().ID(), newSyncer.peerId)
		syncers = append(syncers, newSyncer)
	}

	// Approximately mirroring the replica count in prod.
	for i := 0; i < 15; i++ {
		addSyncer(i)
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, syncer := range syncers {
		// Behold, Go 1.21.
		syncer := syncer
		eg.Go(func() error {
			syncer.requireAltSyncTx(ctx, t, receiptSeq, getPublished)
			t.Logf("%v synced", syncer.name)
			return nil
		})
	}
	require.NoError(t, eg.Wait())

	// Don't stop the nodes right away so they can sync from each other.
	for _, syncer := range syncers {
		syncer.stop()
	}

	altSyncSources := make(map[peer.ID]int)
	for _, syncer := range syncers {
		fmt.Printf("%v (%v)\n", syncer.peerId, syncer.name)
		for source, blocks := range syncer.syncedPayloads {
			fmt.Printf("  %v (%v):", source, len(blocks))
			for _, block := range blocks {
				fmt.Printf(" %v", block.blockId.Number)
			}
			fmt.Printf("\n")
		}
		for _, block := range syncer.syncedPayloads[p2p.PayloadSourceAltSync] {
			altSyncSources[block.from]++
		}
	}
	peerIds := make([]peer.ID, 0, len(altSyncSources))
	for key := range altSyncSources {
		peerIds = append(peerIds, key)
	}
	slices.SortFunc(peerIds, func(a, b peer.ID) int {
		return altSyncSources[b] - altSyncSources[a]
	})
	fmt.Printf("alt sync sources\n")
	for _, peerId := range peerIds {
		fmt.Printf("  %v: %v\n", altSyncSources[peerId], peerId)
	}
}

func linkNodes(t *testing.T, mocknet mocknet.Mocknet, a, b p2p.Node) {
	_, err := mocknet.LinkPeers(a.Host().ID(), b.Host().ID())
	require.NoError(t, err)
}

func connectNodes(t *testing.T, mocknet mocknet.Mocknet, a, b p2p.Node) {
	_, err := mocknet.ConnectPeers(a.Host().ID(), b.Host().ID())
	require.NoError(t, err)
}

func linkAndConnectNodeNamesNodes(t *testing.T, mocknet mocknet.Mocknet, a, b peer.ID) {
	_, err := mocknet.LinkPeers(a, b)
	require.NoError(t, err)
	_, err = mocknet.ConnectPeers(a, b)
	require.NoError(t, err)
}

// Go, y u no separate type and identifier namespaces?!
type syncerType struct {
	// The node name. Not sure how to access this just from the node...
	name           string
	syncedPayloads map[p2p.PayloadSource][]syncedPayload
	h              host.Host
	node           *rollupNode.OpNode
	l2Verif        *ethclient.Client
	stop           func()
	peerId         peer.ID
}

// Waits for the tx in the receipt to become available, then checks the alt sync payloads make
// sense.
func (syncer *syncerType) requireAltSyncTx(
	ctx context.Context,
	t *testing.T,
	receiptSeq *types.Receipt,
	getPublished func() []eth.BlockID,
) {
	// It may take a while to sync, but eventually we should see the sequenced data show up
	receiptVerif, err := wait.ForReceiptOK(ctx, syncer.l2Verif, receiptSeq.TxHash)
	require.Nil(t, err, "Waiting for L2 tx on verifier")

	require.Equal(t, receiptSeq, receiptVerif)

	require.Contains(
		t,
		syncer.altSyncedBlockIds(),
		eth.BlockID{Hash: receiptVerif.BlockHash, Number: receiptVerif.BlockNumber.Uint64()},
	)
	require.Subset(t, getPublished(), syncer.altSyncedBlockIds())
}

func (me *syncerType) altSyncedBlockIds() (ret []eth.BlockID) {
	for _, synced := range me.syncedPayloads[p2p.PayloadSourceAltSync] {
		ret = append(ret, synced.blockId)
	}
	return
}

type syncedPayload struct {
	from    peer.ID
	blockId eth.BlockID
}

func makeSyncer(ctx context.Context, t *testing.T, name string, cfg SystemConfig, sys *System) (syncer *syncerType) {
	// set up our syncer node, connect it to alice/bob
	cfg.Loggers[name] = testlog.Logger(t, log.LevelWarn).New("role", name)

	syncer = &syncerType{
		name: name,
	}
	g.MakeMapWithCap(&syncer.syncedPayloads, 2)

	// Create a peer
	var err error
	syncer.h, err = sys.NewMockNetPeer()
	require.NoError(t, err)

	var payloadMu sync.Mutex
	// Configure the new rollup node that'll be syncing
	syncNodeCfg := &rollupNode.Config{
		Driver:    driver.Config{VerifierConfDepth: 0},
		Rollup:    *sys.RollupConfig,
		P2PSigner: nil,
		RPC: rollupNode.RPCConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P: &p2p.Prepared{HostP2P: syncer.h, ReqRespSync: p2p.ReqRespSyncConfig{
			Enabled: true,
		}},
		Metrics:             rollupNode.MetricsConfig{Enabled: false}, // no metrics server
		Pprof:               oppprof.CLIConfig{},
		L1EpochPollInterval: time.Second * 10,
		Tracer: &FnTracer{
			L2PayloadInFunc: func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayloadEnvelope, source p2p.PayloadSource) error {
				payloadMu.Lock()
				defer payloadMu.Unlock()
				syncer.syncedPayloads[source] = append(syncer.syncedPayloads[source], syncedPayload{
					from:    from,
					blockId: payload.ExecutionPayload.ID(),
				})
				return nil
			},
		},
	}
	ConfigureL1(syncNodeCfg, sys.EthInstances["l1"], sys.L1BeaconEndpoint())
	syncerL2Engine, err := geth.InitL2(name, sys.L2GenesisCfg, cfg.JWTFilePath)
	require.NoError(t, err)
	require.NoError(t, syncerL2Engine.Node.Start())

	ConfigureL2(syncNodeCfg, syncerL2Engine, cfg.JWTSecret)

	syncer.node, err = rollupNode.New(ctx, syncNodeCfg, cfg.Loggers[name], "", metrics.NewMetrics(""))
	require.NoError(t, err)
	syncer.peerId = syncer.node.P2P().Host().ID()
	err = syncer.node.Start(ctx)
	require.NoError(t, err)
	syncer.stop = func() {
		require.NoError(t, syncer.node.Stop(ctx))
	}

	// connect here?

	rpc := syncerL2Engine.UserRPC().(endpoint.ClientRPC).ClientRPC()
	syncer.l2Verif = ethclient.NewClient(rpc)

	return
}

func newInfLimiter() *rate.Limiter {
	return rate.NewLimiter(rate.Inf, 0)
}
