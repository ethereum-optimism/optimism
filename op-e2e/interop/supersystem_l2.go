package interop

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batcherFlags "github.com/ethereum-optimism/optimism/op-batcher/flags"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/interopgen"
	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/services"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/setuputils"
	opnodeconfig "github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/endpoint"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

// l2Node holds per-chain resources for a single L2 node. The chain's op-node lives inside the
// supernode and is reached via a supernode-prefixed CL URL.
type l2Node struct {
	name         string
	l2Geth       *geth.GethInstance
	rollupClient *sources.RollupClient
	gethClient   *ethclient.Client
}

type l2Net struct {
	l2Out        *interopgen.L2Output
	chainID      *big.Int
	operatorKeys map[devkeys.ChainOperatorRole]ecdsa.PrivateKey
	contracts    map[string]interface{}
	userKeys     map[string]ecdsa.PrivateKey

	batcher *bss.BatcherService
	nodes   map[string]*l2Node
}

func (s *interopE2ESystem) L2GethEndpoint(id string, name string) endpoint.RPC {
	net := s.l2s[id]
	return net.nodes[name].l2Geth.UserRPC()
}
func (s *interopE2ESystem) L2GethClient(id string, name string) *ethclient.Client {
	net := s.l2s[id]
	node := net.nodes[name]
	if node.gethClient != nil {
		return node.gethClient
	}
	var ethClient services.EthInstance = node.l2Geth
	rpcEndpoint := ethClient.UserRPC()
	rpcCl := endpoint.DialRPC(
		endpoint.PreferAnyRPC,
		rpcEndpoint,
		func(v string) *rpc.Client {
			logger := testlog.Logger(s.t, log.LevelInfo).New("node", id)
			cl, err := dial.DialRPCClientWithTimeout(context.Background(), logger, v)
			require.NoError(s.t, err, "failed to dial eth node instance %s", id)
			return cl
		})
	node.gethClient = ethclient.NewClient(rpcCl)
	return node.gethClient
}

// L2RollupEndpoint returns the chain's CL RPC endpoint, served by the supernode at
// <supernode-base>/<chainID>. The name argument is unused — every chain has a single
// supernode-owned op-node.
func (s *interopE2ESystem) L2RollupEndpoint(id string, name string) endpoint.RPC {
	return endpoint.URL(s.chainURL(s.l2s[id].chainID))
}

func (s *interopE2ESystem) L2RollupClient(id string, name string) *sources.RollupClient {
	net := s.l2s[id]
	node := net.nodes[name]
	if node.rollupClient != nil {
		return node.rollupClient
	}
	cl, err := dial.DialRollupClientWithTimeout(
		context.Background(),
		s.logger,
		s.chainURL(net.chainID),
	)
	require.NoError(s.t, err, "failed to dial rollup client via supernode")
	node.rollupClient = cl
	return cl
}

// newL2 sets up the per-chain state needed before the supernode boots: keys, the L2 EL (op-geth),
// and an empty node map. The op-node is configured via newSupernodeNodeConfig and started by
// the supernode; the batcher is started after the supernode is up.
func (s *interopE2ESystem) newL2(id string, l2Out *interopgen.L2Output) l2Net {
	operatorKeys := s.newOperatorKeysForL2(l2Out)
	l2Geth := s.newGethForL2(id, "sequencer", l2Out)
	return l2Net{
		l2Out:        l2Out,
		chainID:      l2Out.Genesis.Config.ChainID,
		nodes:        map[string]*l2Node{"sequencer": {name: "sequencer", l2Geth: l2Geth}},
		operatorKeys: operatorKeys,
		userKeys:     make(map[string]ecdsa.PrivateKey),
		contracts:    make(map[string]interface{}),
	}
}

// newSupernodeNodeConfig builds the per-chain op-node configuration that the supernode runs
// internally as the chain's virtual op-node.
func (s *interopE2ESystem) newSupernodeNodeConfig(l2 l2Net, depSet depset.DependencySet) *opnodeconfig.Config {
	p2pKey := l2.operatorKeys[devkeys.SequencerP2PRole]
	p2pSigner := &p2p.PreparedSigner{Signer: opsigner.NewLocalSigner(&p2pKey)}

	l2Geth := l2.nodes["sequencer"].l2Geth

	cfg := &opnodeconfig.Config{
		L1: &opnodeconfig.L1EndpointConfig{
			L1NodeAddr:       s.l1.UserRPC().RPC(),
			L1TrustRPC:       false,
			L1RPCKind:        sources.RPCKindDebugGeth,
			BatchSize:        20,
			HttpPollInterval: 100 * time.Millisecond,
			MaxConcurrency:   10,
		},
		L1ChainConfig: s.l1.Backend.BlockChain().Config(),
		L2: &opnodeconfig.L2EndpointConfig{
			L2EngineAddr:      l2Geth.AuthRPC().RPC(),
			L2EngineJWTSecret: testingJWTSecret,
		},
		Beacon: &opnodeconfig.L1BeaconEndpointConfig{
			BeaconAddr: s.beacon.BeaconAddr(),
		},
		Driver: driver.Config{
			SequencerEnabled:   true,
			SequencerConfDepth: 2,
		},
		Rollup:        *l2.l2Out.RollupCfg,
		DependencySet: depSet,
		P2PSigner:     p2pSigner,
		RPC: oprpc.CLIConfig{
			ListenAddr:  "127.0.0.1",
			ListenPort:  0,
			EnableAdmin: true,
		},
		P2P:                             nil,
		L1EpochPollInterval:             2 * time.Second,
		Sync:                            sync.Config{SyncMode: sync.CLSync, SyncModeReqResp: true},
		ConfigPersistence:               opnodeconfig.DisabledConfigPersistence{},
		ExperimentalOPStackAPI:          true,
		IgnoreMissingPectraBlobSchedule: false,
	}
	require.NoError(s.t, cfg.Check(), "invalid supernode op-node config for chain %s", l2.chainID)
	return cfg
}

// newGethForL2 creates a new Geth instance for an L2 chain.
func (s *interopE2ESystem) newGethForL2(id string, node string, l2Out *interopgen.L2Output) *geth.GethInstance {
	jwtPath := writeDefaultJWT(s.t)
	name := "l2-" + id + "-" + node
	l2Geth, err := geth.InitL2(name, l2Out.Genesis, jwtPath,
		func(ethCfg *ethconfig.Config, nodeCfg *gn.Config) error {
			return nil
		})
	require.NoError(s.t, err)
	require.NoError(s.t, l2Geth.Node.Start())
	s.t.Cleanup(func() {
		s.t.Logf("Closing L2 geth of chain %s", id)
		closeErr := l2Geth.Close()
		s.t.Logf("Closed L2 geth of chain %s: %v", id, closeErr)
	})
	return l2Geth
}

// startBatcherForL2 starts the batcher for the given L2. Must be called after the supernode is
// running, because the batcher needs the supernode-routed CL RPC URL.
func (s *interopE2ESystem) startBatcherForL2(id string) {
	l2 := s.l2s[id]
	l2Geth := l2.nodes["sequencer"].l2Geth
	batcherSecret := l2.operatorKeys[devkeys.BatcherRole]
	logger := s.logger.New("role", "batcher"+id)
	daType := batcherFlags.CalldataType
	if s.config.BatcherUsesBlobs {
		daType = batcherFlags.BlobsType
	}
	batcherCLIConfig := &bss.CLIConfig{
		L1EthRpc:                 s.l1.UserRPC().RPC(),
		L2EthRpc:                 []string{l2Geth.UserRPC().RPC()},
		RollupRpc:                []string{s.chainURL(l2.chainID)},
		MaxPendingTransactions:   1,
		MaxChannelDuration:       1,
		MaxL1TxSize:              120_000,
		TestUseMaxTxSizeForBlobs: false,
		TargetNumFrames:          1,
		ApproxComprRatio:         0.4,
		SubSafetyMargin:          4,
		PollInterval:             50 * time.Millisecond,
		TxMgrConfig:              setuputils.NewTxMgrConfig(s.l1.UserRPC(), &batcherSecret),
		LogConfig: oplog.CLIConfig{
			Level:  log.LevelInfo,
			Format: oplog.FormatText,
		},
		Stopped:               false,
		BatchType:             derive.SpanBatchType,
		MaxBlocksPerSpanBatch: 10,
		DataAvailabilityType:  daType,
		CompressionAlgo:       derive.Brotli,
	}

	batcherContext, batcherCancel := context.WithCancel(context.Background())
	var closeAppFn context.CancelCauseFunc = func(cause error) {
		s.t.Fatalf("closeAppFn called, batcher hit a critical error: %v", cause)
		batcherCancel()
	}

	batcher, err := bss.BatcherServiceFromCLIConfig(
		batcherContext, closeAppFn, "0.0.1", batcherCLIConfig,
		logger.New("service", "batcher"))
	require.NoError(s.t, err)
	require.NoError(s.t, batcher.Start(context.Background()))
	s.t.Cleanup(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		s.t.Logf("Closing batcher of chain %s", id)
		_ = batcher.Stop(ctx)
		s.t.Logf("Closed batcher of chain %s", id)
	})
	l2.batcher = batcher
	s.l2s[id] = l2
}
