package op_e2e

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	batchermetrics "github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	proposermetrics "github.com/ethereum-optimism/optimism/op-proposer/metrics"
	l2os "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	geth_eth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"
)

// TestSystem_PeerScoring tests that the peer scoring system works as expected.
func TestSystem_PeerScoring(t *testing.T) {
	parallel(t)
	if !verboseGethNodes {
		log.Root().SetHandler(log.DiscardHandler())
	}

	cfg := DefaultSystemConfig(t)

	// Slow down L1 blocks so we can see the L2 blocks arrive before L1 blocks
	cfg.DeployConfig.L1BlockTime = 10

	// Dial the nodes to each other
	cfg.P2PTopology = map[string][]string{"verifier": {"sequencer"}}

	// Set peer scoring for each node, but without banning
	for _, node := range cfg.Nodes {
		params, err := p2p.GetPeerScoreParams("light", 2)
		require.NoError(t, err)
		bandScoreThresholds, err := p2p.NewBandScorer("-40:graylist;-20:restricted;0:nopx;20:friend;")
		require.NoError(t, err)
		p2pPrivKey := secp256k1.NewPrivateKey(&secp256k1.ModNScalar{})
		node.P2P = &p2p.Config{
			Priv:                (*crypto.Secp256k1PrivateKey)(p2pPrivKey),
			PeerScoring:         params,
			BanningEnabled:      false,
			BandScoreThresholds: *bandScoreThresholds,
		}
		// We only want to set the signer of the first verifier
		// if name == "verifier" {
		// 	node.P2PSigner = &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(cfg.Secrets.Alice)}
		// }
	}

	var published, received []common.Hash
	seqTracer, verifTracer := new(FnTracer), new(FnTracer)
	seqTracer.OnPublishL2PayloadFn = func(ctx context.Context, payload *eth.ExecutionPayload) {
		published = append(published, payload.BlockHash)
	}
	verifTracer.OnUnsafeL2PayloadFn = func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) {
		received = append(received, payload.BlockHash)
	}
	cfg.Nodes["sequencer"].Tracer = seqTracer
	cfg.Nodes["verifier"].Tracer = verifTracer

	sys, p2pNodes, err := startConfig(&cfg)
	require.NoError(t, err, "Error starting up system")
	defer sys.Close()

	l2Seq := sys.Clients["sequencer"]
	l2Verif := sys.Clients["verifier"]

	// Transactor Account
	ethPrivKey := cfg.Secrets.Alice

	// Create a malicious node

	// Connect the malicious verifier node to the mocknet
	verifierPeerID := p2pNodes["verifier"].HostP2P.ID()
	// TODO: Construct a malicious node and get the peer id here since p2pNodes["malicious"] is nil
	maliciousPeerID := p2pNodes["malicious"].HostP2P.ID()
	_, err = sys.Mocknet.ConnectPeers(verifierPeerID, maliciousPeerID)
	require.NoError(t, err, "failed to setup mocknet connection between %s and %s", verifierPeerID, maliciousPeerID)

	// TODO: Have the malicious verifier node broadcast a tx to the second verifier node

	// Submit TX to L2 sequencer node
	toAddr := common.Address{0xff, 0xff}
	tx := types.MustSignNewTx(ethPrivKey, types.LatestSignerForChainID(cfg.L2ChainIDBig()), &types.DynamicFeeTx{
		ChainID:   cfg.L2ChainIDBig(),
		Nonce:     0,
		To:        &toAddr,
		Value:     big.NewInt(1_000_000_000),
		GasTipCap: big.NewInt(10),
		GasFeeCap: big.NewInt(200),
		Gas:       21000,
	})
	err = l2Seq.SendTransaction(context.Background(), tx)
	require.NoError(t, err, "Sending L2 tx to sequencer")

	// Wait for tx to be mined on the L2 sequencer chain
	receiptSeq, err := waitForTransaction(tx.Hash(), l2Seq, 10*time.Duration(sys.RollupConfig.BlockTime)*time.Second)
	require.NoError(t, err, "Waiting for L2 tx on sequencer")

	// Wait until the block it was first included in shows up in the safe chain on the verifier
	receiptVerif, err := waitForTransaction(tx.Hash(), l2Verif, 10*time.Duration(sys.RollupConfig.BlockTime)*time.Second)
	require.NoError(t, err, "Waiting for L2 tx on verifier")
	require.Equal(t, receiptSeq, receiptVerif)

	// Verify that everything that was received was published
	require.GreaterOrEqual(t, len(published), len(received))
	require.ElementsMatch(t, published, received[:len(published)])

	// Verify that the tx was received via p2p
	require.Contains(t, received, receiptVerif.BlockHash)

	// fmt.Printf("Publishing L2 payload to the first verifier rollup node...")
	// require.NoError(t, sys.RollupNodes["verifier"].PublishL2Payload(context.Background(), &eth.ExecutionPayload{
	// 	BlockHash: common.HexToHash("0xdeadbeef"),
	// 	From
	// }))
}

func startConfig(cfg *SystemConfig) (*System, map[string]*p2p.Prepared, error) {
	sys := &System{
		cfg:         *cfg,
		Nodes:       make(map[string]*node.Node),
		Backends:    make(map[string]*geth_eth.Ethereum),
		Clients:     make(map[string]*ethclient.Client),
		RollupNodes: make(map[string]*rollupNode.OpNode),
	}
	didErrAfterStart := false
	defer func() {
		if didErrAfterStart {
			for _, node := range sys.RollupNodes {
				node.Close()
			}
			for _, node := range sys.Nodes {
				node.Close()
			}
		}
	}()

	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig)
	if err != nil {
		return nil, nil, err
	}

	for addr, amount := range cfg.Premine {
		if existing, ok := l1Genesis.Alloc[addr]; ok {
			l1Genesis.Alloc[addr] = core.GenesisAccount{
				Code:    existing.Code,
				Storage: existing.Storage,
				Balance: amount,
				Nonce:   existing.Nonce,
			}
		} else {
			l1Genesis.Alloc[addr] = core.GenesisAccount{
				Balance: amount,
				Nonce:   0,
			}
		}
	}

	l1Block := l1Genesis.ToBlock()
	l2Genesis, err := genesis.BuildL2DeveloperGenesis(cfg.DeployConfig, l1Block)
	if err != nil {
		return nil, nil, err
	}
	for addr, amount := range cfg.Premine {
		if existing, ok := l2Genesis.Alloc[addr]; ok {
			l2Genesis.Alloc[addr] = core.GenesisAccount{
				Code:    existing.Code,
				Storage: existing.Storage,
				Balance: amount,
				Nonce:   existing.Nonce,
			}
		} else {
			l2Genesis.Alloc[addr] = core.GenesisAccount{
				Balance: amount,
				Nonce:   0,
			}
		}
	}

	makeRollupConfig := func() rollup.Config {
		return rollup.Config{
			Genesis: rollup.Genesis{
				L1: eth.BlockID{
					Hash:   l1Block.Hash(),
					Number: 0,
				},
				L2: eth.BlockID{
					Hash:   l2Genesis.ToBlock().Hash(),
					Number: 0,
				},
				L2Time:       uint64(cfg.DeployConfig.L1GenesisBlockTimestamp),
				SystemConfig: e2eutils.SystemConfigFromDeployConfig(cfg.DeployConfig),
			},
			BlockTime:              cfg.DeployConfig.L2BlockTime,
			MaxSequencerDrift:      cfg.DeployConfig.MaxSequencerDrift,
			SeqWindowSize:          cfg.DeployConfig.SequencerWindowSize,
			ChannelTimeout:         cfg.DeployConfig.ChannelTimeout,
			L1ChainID:              cfg.L1ChainIDBig(),
			L2ChainID:              cfg.L2ChainIDBig(),
			BatchInboxAddress:      cfg.DeployConfig.BatchInboxAddress,
			DepositContractAddress: predeploys.DevOptimismPortalAddr,
			L1SystemConfigAddress:  predeploys.DevSystemConfigAddr,
			RegolithTime:           cfg.DeployConfig.RegolithTime(uint64(cfg.DeployConfig.L1GenesisBlockTimestamp)),
		}
	}
	defaultConfig := makeRollupConfig()
	sys.RollupConfig = &defaultConfig

	// Initialize nodes
	l1Node, l1Backend, err := initL1Geth(cfg, l1Genesis, cfg.GethOptions["l1"]...)
	if err != nil {
		return nil, nil, err
	}
	sys.Nodes["l1"] = l1Node
	sys.Backends["l1"] = l1Backend

	for name := range cfg.Nodes {
		node, backend, err := initL2Geth(name, big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, cfg.JWTFilePath, cfg.GethOptions[name]...)
		if err != nil {
			return nil, nil, err
		}
		sys.Nodes[name] = node
		sys.Backends[name] = backend
	}

	// Start
	err = l1Node.Start()
	if err != nil {
		didErrAfterStart = true
		return nil, nil, err
	}
	err = l1Backend.StartMining(1)
	if err != nil {
		didErrAfterStart = true
		return nil, nil, err
	}
	for name, node := range sys.Nodes {
		if name == "l1" {
			continue
		}
		err = node.Start()
		if err != nil {
			didErrAfterStart = true
			return nil, nil, err
		}
	}

	// Configure connections to L1 and L2 for rollup nodes.
	l1EndpointConfig := l1Node.WSEndpoint()
	useHTTP := os.Getenv("OP_E2E_USE_HTTP") == "true"
	if useHTTP {
		log.Info("using HTTP client")
		l1EndpointConfig = l1Node.HTTPEndpoint()
	}

	for name, rollupCfg := range cfg.Nodes {
		l2EndpointConfig := sys.Nodes[name].WSAuthEndpoint()
		if useHTTP {
			l2EndpointConfig = sys.Nodes[name].HTTPAuthEndpoint()
		}
		rollupCfg.L1 = &rollupNode.L1EndpointConfig{
			L1NodeAddr: l1EndpointConfig,
			L1TrustRPC: false,
			L1RPCKind:  sources.RPCKindBasic,
		}
		rollupCfg.L2 = &rollupNode.L2EndpointConfig{
			L2EngineAddr:      l2EndpointConfig,
			L2EngineJWTSecret: cfg.JWTSecret,
		}
		rollupCfg.L2Sync = &rollupNode.PreparedL2SyncEndpoint{
			Client:   nil,
			TrustRPC: false,
		}
	}

	// Geth Clients
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	l1Srv, err := l1Node.RPCHandler()
	if err != nil {
		didErrAfterStart = true
		return nil, nil, err
	}
	l1Client := ethclient.NewClient(rpc.DialInProc(l1Srv))
	sys.Clients["l1"] = l1Client
	for name, node := range sys.Nodes {
		client, err := ethclient.DialContext(ctx, node.WSEndpoint())
		if err != nil {
			didErrAfterStart = true
			return nil, nil, err
		}
		sys.Clients[name] = client
	}

	_, err = waitForBlock(big.NewInt(2), l1Client, 6*time.Second*time.Duration(cfg.DeployConfig.L1BlockTime))
	if err != nil {
		return nil, nil, fmt.Errorf("waiting for blocks: %w", err)
	}

	// Create the custom mocknet
	sys.Mocknet = mocknet.New()
	p2pNodes := make(map[string]*p2p.Prepared)

	// Helper function to initialize a mocknet host
	initHostMaybe := func(name string) (*p2p.Prepared, error) {
		if p, ok := p2pNodes[name]; ok {
			return p, nil
		}
		h, err := sys.Mocknet.GenPeer()
		if err != nil {
			return nil, fmt.Errorf("failed to init p2p host for node %s", name)
		}
		h.Network()
		_, ok := cfg.Nodes[name]
		if !ok {
			return nil, fmt.Errorf("node %s from p2p topology not found in actual nodes map", name)
		}
		p := &p2p.Prepared{
			HostP2P:   h,
			LocalNode: nil,
			UDPv5:     nil,
		}
		p2pNodes[name] = p
		return p, nil
	}

	// Create the verifier and sequencer hosts
	verifierHost, err := initHostMaybe("verifier")
	if err != nil {
		return nil, nil, err
	}
	sequencerHost, err := initHostMaybe("sequencer")
	if err != nil {
		return nil, nil, err
	}
	if _, err := sys.Mocknet.LinkPeers(verifierHost.HostP2P.ID(), sequencerHost.HostP2P.ID()); err != nil {
		return nil, nil, err
	}

	// Start the nodes
	sys.RollupNodes["sequencer"], err = startNode(cfg, "sequencer", &p2pNodes, makeRollupConfig)
	if err != nil {
		return nil, nil, err
	}
	sys.RollupNodes["verifier"], err = startNode(cfg, "verifier", &p2pNodes, makeRollupConfig)
	if err != nil {
		return nil, nil, err
	}

	// Node topology:
	//
	//		 sequencer
	//  	 / /
	//	verifier = malicious

	if _, err := connectNodes(p2pNodes["sequencer"], p2pNodes["verifier"], sys.Mocknet); err != nil {
		return nil, nil, err
	}
	if _, err := connectNodes(p2pNodes["verifier"], p2pNodes["sequencer"], sys.Mocknet); err != nil {
		return nil, nil, err
	}
	if _, err := connectNodes(p2pNodes["verifier"], p2pNodes["malicious"], sys.Mocknet); err != nil {
		return nil, nil, err
	}
	if _, err := connectNodes(p2pNodes["malicious"], p2pNodes["verifier"], sys.Mocknet); err != nil {
		return nil, nil, err
	}

	// L2Output Submitter
	sys.L2OutputSubmitter, err = l2os.NewL2OutputSubmitterFromCLIConfig(l2os.CLIConfig{
		L1EthRpc:                  sys.Nodes["l1"].WSEndpoint(),
		RollupRpc:                 sys.RollupNodes["sequencer"].HTTPEndpoint(),
		L2OOAddress:               predeploys.DevL2OutputOracleAddr.String(),
		PollInterval:              50 * time.Millisecond,
		NumConfirmations:          1,
		ResubmissionTimeout:       3 * time.Second,
		SafeAbortNonceTooLowCount: 3,
		AllowNonFinalized:         cfg.NonFinalizedProposals,
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
		PrivateKey: hexPriv(cfg.Secrets.Proposer),
	}, sys.cfg.Loggers["proposer"], proposermetrics.NoopMetrics)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to setup l2 output submitter: %w", err)
	}

	if err := sys.L2OutputSubmitter.Start(); err != nil {
		return nil, nil, fmt.Errorf("unable to start l2 output submitter: %w", err)
	}

	// Batch Submitter
	sys.BatchSubmitter, err = bss.NewBatchSubmitterFromCLIConfig(bss.CLIConfig{
		L1EthRpc:                  sys.Nodes["l1"].WSEndpoint(),
		L2EthRpc:                  sys.Nodes["sequencer"].WSEndpoint(),
		RollupRpc:                 sys.RollupNodes["sequencer"].HTTPEndpoint(),
		MaxChannelDuration:        1,
		MaxL1TxSize:               120_000,
		TargetL1TxSize:            100_000,
		TargetNumFrames:           1,
		ApproxComprRatio:          0.4,
		SubSafetyMargin:           4,
		PollInterval:              50 * time.Millisecond,
		NumConfirmations:          1,
		ResubmissionTimeout:       5 * time.Second,
		SafeAbortNonceTooLowCount: 3,
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
		PrivateKey: hexPriv(cfg.Secrets.Batcher),
	}, sys.cfg.Loggers["batcher"], batchermetrics.NoopMetrics)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup batch submitter: %w", err)
	}

	// Batcher may be enabled later
	if !sys.cfg.DisableBatcher {
		if err := sys.BatchSubmitter.Start(); err != nil {
			return nil, nil, fmt.Errorf("unable to start batch submitter: %w", err)
		}
	}

	return sys, p2pNodes, nil
}

func connectNodes(nodeA *p2p.Prepared, nodeB *p2p.Prepared, mocknet mocknet.Mocknet) (network.Conn, error) {
	return mocknet.ConnectPeers(nodeA.HostP2P.ID(), nodeB.HostP2P.ID())
}

func startNode(cfg *SystemConfig, name string, p2pNodes *map[string]*p2p.Prepared, makeRollupConfig func() rollup.Config) (*rollupNode.OpNode, error) {
	fmt.Printf("Starting node: %s with config: %+v\n", name, cfg.Nodes[name])
	nodeConfig := cfg.Nodes[name]
	c := *nodeConfig
	c.Rollup = makeRollupConfig()

	if p, ok := (*p2pNodes)[name]; ok {
		c.P2P = p

		if c.Driver.SequencerEnabled && c.P2PSigner == nil {
			c.P2PSigner = &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(cfg.Secrets.SequencerP2P)}
		}
	}

	c.Rollup.LogDescription(cfg.Loggers[name], chaincfg.L2ChainIDToNetworkName)

	snapLog := log.New()
	snapLog.SetHandler(log.DiscardHandler())
	node, err := rollupNode.New(context.Background(), &c, cfg.Loggers[name], snapLog, "", metrics.NewMetrics(""))
	if err != nil {
		return nil, err
	}
	err = node.Start(context.Background())
	if err != nil {
		return nil, err
	}

	return node, nil
}
