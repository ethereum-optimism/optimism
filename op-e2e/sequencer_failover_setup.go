package op_e2e

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	con "github.com/ethereum-optimism/optimism/op-conductor/conductor"
	conrpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const (
	sequencer1Name = "sequencer1"
	sequencer2Name = "sequencer2"
	sequencer3Name = "sequencer3"
	verifierName   = "verifier"

	localhost = "127.0.0.1"
)

type conductor struct {
	service       *con.OpConductor
	client        conrpc.API
	consensusPort int
}

func (c *conductor) ConsensusEndpoint() string {
	return fmt.Sprintf("%s:%d", localhost, c.consensusPort)
}

func setupSequencerFailoverTest(t *testing.T) (*System, map[string]*conductor) {
	InitParallel(t)
	ctx := context.Background()

	// 3 sequencers, 1 verifier, 1 active sequencer.
	cfg := sequencerFailoverSystemConfig(t)
	sys, err := cfg.Start(t)
	require.NoError(t, err)

	// 1 batcher that listens to all 3 sequencers, in started mode.
	setupBatcher(t, sys)

	// 3 conductors that connects to 1 sequencer each.
	conductors := make(map[string]*conductor)

	// initialize all conductors in paused mode
	conductorCfgs := []struct {
		name      string
		bootstrap bool
	}{
		{sequencer1Name, true}, // one in bootstrap mode so that we can form a cluster.
		{sequencer2Name, false},
		{sequencer3Name, false},
	}
	for _, cfg := range conductorCfgs {
		cfg := cfg
		nodePRC := sys.RollupNodes[cfg.name].HTTPEndpoint()
		engineRPC := sys.EthInstances[cfg.name].HTTPEndpoint()
		conductors[cfg.name] = setupConductor(t, cfg.name, t.TempDir(), nodePRC, engineRPC, cfg.bootstrap, *sys.RollupConfig)
	}

	// form a cluster
	c1 := conductors[sequencer1Name]
	c2 := conductors[sequencer2Name]
	c3 := conductors[sequencer3Name]

	require.NoError(t, waitForLeadershipChange(t, c1, true))
	require.NoError(t, c1.client.AddServerAsVoter(ctx, sequencer2Name, c2.ConsensusEndpoint()))
	require.NoError(t, c1.client.AddServerAsVoter(ctx, sequencer3Name, c3.ConsensusEndpoint()))
	require.True(t, leader(t, ctx, c1))
	require.False(t, leader(t, ctx, c2))
	require.False(t, leader(t, ctx, c3))

	// weirdly, batcher does not submit a batch until unsafe block 9.
	// It became normal after that and submits a batch every L1 block (2s) per configuration.
	// Since our health monitor checks on safe head progression, wait for batcher to become normal before proceeding.
	require.NoError(t, wait.ForNextSafeBlock(ctx, sys.Clients[sequencer1Name]))
	require.NoError(t, wait.ForNextSafeBlock(ctx, sys.Clients[sequencer1Name]))
	require.NoError(t, wait.ForNextSafeBlock(ctx, sys.Clients[sequencer1Name]))

	// make sure conductor reports all sequencers as healthy, this means they're syncing correctly.
	require.True(t, healthy(t, ctx, c1))
	require.True(t, healthy(t, ctx, c2))
	require.True(t, healthy(t, ctx, c3))

	// unpause all conductors
	require.NoError(t, c1.client.Resume(ctx))
	require.NoError(t, c2.client.Resume(ctx))
	require.NoError(t, c3.client.Resume(ctx))

	// final check, make sure everything is in the right place
	require.True(t, conductorActive(t, ctx, c1))
	require.True(t, conductorActive(t, ctx, c2))
	require.True(t, conductorActive(t, ctx, c3))

	require.True(t, sequencerActive(t, ctx, sys.RollupClient(sequencer1Name)))
	require.False(t, sequencerActive(t, ctx, sys.RollupClient(sequencer2Name)))
	require.False(t, sequencerActive(t, ctx, sys.RollupClient(sequencer3Name)))

	require.True(t, healthy(t, ctx, c1))
	require.True(t, healthy(t, ctx, c2))
	require.True(t, healthy(t, ctx, c3))

	return sys, conductors
}

func setupConductor(
	t *testing.T,
	serverID, dir, nodePRC, engineRPC string,
	bootstrap bool,
	rollupCfg rollup.Config,
) *conductor {
	// it's unfortunate that it is not possible to pass 0 as consensus port and get back the actual assigned port from raft implementation.
	// So we find an available port and pass it in to avoid test flakiness (avoid port already in use error).
	consensusPort := findAvailablePort(t)
	cfg := con.Config{
		ConsensusAddr:  localhost,
		ConsensusPort:  consensusPort,
		RaftServerID:   serverID,
		RaftStorageDir: dir,
		RaftBootstrap:  bootstrap,
		NodeRPC:        nodePRC,
		ExecutionRPC:   engineRPC,
		Paused:         true,
		HealthCheck: con.HealthCheckConfig{
			Interval:     1, // per test setup, l2 block time is 1s.
			SafeInterval: 4, // per test setup (l1 block time = 2s, max channel duration = 1, 2s buffer)
			MinPeerCount: 2, // per test setup, each sequencer has 2 peers
		},
		RollupCfg: rollupCfg,
		LogConfig: oplog.CLIConfig{
			Level: log.LvlInfo,
			Color: false,
		},
		RPC: oprpc.CLIConfig{
			ListenAddr: localhost,
			ListenPort: 0,
		},
	}

	ctx := context.Background()
	service, err := con.New(ctx, &cfg, testlog.Logger(t, log.LvlInfo), "0.0.1")
	require.NoError(t, err)
	err = service.Start(ctx)
	require.NoError(t, err)

	rawClient, err := rpc.DialContext(ctx, service.HTTPEndpoint())
	require.NoError(t, err)
	client := conrpc.NewAPIClient(rawClient)

	return &conductor{
		service:       service,
		client:        client,
		consensusPort: consensusPort,
	}
}

func setupBatcher(t *testing.T, sys *System) {
	var batchType uint = derive.SingularBatchType
	if sys.Cfg.DeployConfig.L2GenesisDeltaTimeOffset != nil && *sys.Cfg.DeployConfig.L2GenesisDeltaTimeOffset == hexutil.Uint64(0) {
		batchType = derive.SpanBatchType
	}
	batcherMaxL1TxSizeBytes := sys.Cfg.BatcherMaxL1TxSizeBytes
	if batcherMaxL1TxSizeBytes == 0 {
		batcherMaxL1TxSizeBytes = 240_000
	}

	// enable active sequencer follow mode.
	l2EthRpc := strings.Join([]string{
		sys.EthInstances[sequencer1Name].WSEndpoint(),
		sys.EthInstances[sequencer2Name].WSEndpoint(),
		sys.EthInstances[sequencer3Name].WSEndpoint(),
	}, ",")
	rollupRpc := strings.Join([]string{
		sys.RollupNodes[sequencer1Name].HTTPEndpoint(),
		sys.RollupNodes[sequencer2Name].HTTPEndpoint(),
		sys.RollupNodes[sequencer3Name].HTTPEndpoint(),
	}, ",")
	batcherCLIConfig := &bss.CLIConfig{
		L1EthRpc:               sys.EthInstances["l1"].WSEndpoint(),
		L2EthRpc:               l2EthRpc,
		RollupRpc:              rollupRpc,
		MaxPendingTransactions: 0,
		MaxChannelDuration:     1,
		MaxL1TxSize:            batcherMaxL1TxSizeBytes,
		CompressorConfig: compressor.CLIConfig{
			TargetL1TxSizeBytes: sys.Cfg.BatcherTargetL1TxSizeBytes,
			TargetNumFrames:     1,
			ApproxComprRatio:    0.4,
		},
		SubSafetyMargin: 0,
		PollInterval:    50 * time.Millisecond,
		TxMgrConfig:     newTxMgrConfig(sys.EthInstances["l1"].WSEndpoint(), sys.Cfg.Secrets.Batcher),
		LogConfig: oplog.CLIConfig{
			Level:  log.LvlInfo,
			Format: oplog.FormatText,
		},
		Stopped:              false,
		BatchType:            batchType,
		DataAvailabilityType: batcherFlags.CalldataType,
	}

	batcher, err := bss.BatcherServiceFromCLIConfig(context.Background(), "0.0.1", batcherCLIConfig, sys.Cfg.Loggers["batcher"])
	require.NoError(t, err)
	err = batcher.Start(context.Background())
	require.NoError(t, err)
	sys.BatchSubmitter = batcher
}

func sequencerFailoverSystemConfig(t *testing.T) SystemConfig {
	cfg := DefaultSystemConfig(t)
	delete(cfg.Nodes, "sequencer")
	cfg.Nodes[sequencer1Name] = sequencerCfg(true)
	cfg.Nodes[sequencer2Name] = sequencerCfg(false)
	cfg.Nodes[sequencer3Name] = sequencerCfg(false)

	delete(cfg.Loggers, "sequencer")
	cfg.Loggers[sequencer1Name] = testlog.Logger(t, log.LvlInfo).New("role", sequencer1Name)
	cfg.Loggers[sequencer2Name] = testlog.Logger(t, log.LvlInfo).New("role", sequencer2Name)
	cfg.Loggers[sequencer3Name] = testlog.Logger(t, log.LvlInfo).New("role", sequencer3Name)

	cfg.P2PTopology = map[string][]string{
		sequencer1Name: {sequencer2Name, sequencer3Name},
		sequencer2Name: {sequencer3Name, verifierName},
		sequencer3Name: {verifierName, sequencer1Name},
		verifierName:   {sequencer1Name, sequencer2Name},
	}

	return cfg
}

func sequencerCfg(sequencerEnabled bool) *rollupNode.Config {
	return &rollupNode.Config{
		Driver: driver.Config{
			VerifierConfDepth:  0,
			SequencerConfDepth: 0,
			SequencerEnabled:   sequencerEnabled,
		},
		// Submitter PrivKey is set in system start for rollup nodes where sequencer = true
		RPC: rollupNode.RPCConfig{
			ListenAddr:  localhost,
			ListenPort:  0,
			EnableAdmin: true,
		},
		L1EpochPollInterval:         time.Second * 2,
		RuntimeConfigReloadInterval: time.Minute * 10,
		ConfigPersistence:           &rollupNode.DisabledConfigPersistence{},
		Sync:                        sync.Config{SyncMode: sync.CLSync},
	}
}

func waitForLeadershipChange(t *testing.T, c *conductor, leader bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			isLeader, err := c.client.Leader(ctx)
			if err != nil {
				return err
			}
			if isLeader == leader {
				return nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

func leader(t *testing.T, ctx context.Context, con *conductor) bool {
	leader, err := con.client.Leader(ctx)
	require.NoError(t, err)
	return leader
}

func healthy(t *testing.T, ctx context.Context, con *conductor) bool {
	healthy, err := con.client.SequencerHealthy(ctx)
	require.NoError(t, err)
	return healthy
}

func conductorActive(t *testing.T, ctx context.Context, con *conductor) bool {
	active, err := con.client.Active(ctx)
	require.NoError(t, err)
	return active
}

func sequencerActive(t *testing.T, ctx context.Context, rollupClient *sources.RollupClient) bool {
	active, err := rollupClient.SequencerActive(ctx)
	require.NoError(t, err)
	return active
}

func findAvailablePort(t *testing.T) int {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			t.Error("Failed to find available port")
		default:
			port := rand.Intn(65535-1024) + 1024 // Random port in the range 1024-65535
			addr := fmt.Sprintf("127.0.0.1:%d", port)
			l, err := net.Listen("tcp", addr)
			if err == nil {
				l.Close() // Close the listener and return the port if it's available
				return port
			}
		}
	}
}
