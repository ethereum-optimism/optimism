package p2p

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	. "github.com/ethereum-optimism/optimism/op-e2e"
	. "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	. "github.com/ethereum-optimism/optimism/op-e2e/system/e2esys"
	. "github.com/ethereum-optimism/optimism/op-e2e/system/helpers"
)

func TestSystemP2PAltSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	InitParallel(t)

	cfg := DefaultSystemConfig(t)

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
	cfg.P2PReqRespSync.Enabled = true

	// Disable batcher, so there will not be any L1 data to sync from
	cfg.DisableBatcher = true

	var published []eth.BlockID
	seqTracer := new(FnTracer)
	// The sequencer still publishes the blocks to the tracer, even if they do not reach the network due to disabled P2P
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
		published = append(published, payload.ExecutionPayload.ID())
	}
	// Blocks are now received via the RPC based alt-sync method
	cfg.Nodes["sequencer"].Tracer = seqTracer

	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")

	l2Seq := sys.NodeClient("sequencer")

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Submit a TX to L2 sequencer node
	receiptSeq := SendL2Tx(t, cfg, l2Seq, ethPrivKey, func(opts *TxOpts) {
		opts.ToAddr = &common.Address{0xff, 0xff}
		opts.Value = big.NewInt(1_000_000_000)
	})

	// Gossip is able to respond to IWANT messages for the duration of heartbeat_time * message_window = 0.5 * 12 = 6
	// Wait till we pass that, and then we'll have missed some blocks that cannot be retrieved in any way from gossip
	time.Sleep(time.Second * 10)

	syncer := makeSyncer(ctx, t, "syncer", cfg, sys)
	defer syncer.stop()

	linkNodes(t, sys.Mocknet, sys.RollupNodes[RoleSeq].P2P(), syncer.node.P2P())
	connectNodes(t, sys.Mocknet, sys.RollupNodes[RoleSeq].P2P(), syncer.node.P2P())

	syncer.requireAltSyncTx(ctx, t, receiptSeq, func() []eth.BlockID {
		return published
	})
}
