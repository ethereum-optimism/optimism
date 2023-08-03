package op_e2e

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
	"os"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/p2p/store"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	ds "github.com/ipfs/go-datastore"
	"github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	geth_eth "github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
	"github.com/stretchr/testify/require"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-batcher/compressor"
	batchermetrics "github.com/ethereum-optimism/optimism/op-batcher/metrics"
	"github.com/ethereum-optimism/optimism/op-bindings/predeploys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-node/chaincfg"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/driver"
	"github.com/ethereum-optimism/optimism/op-node/sources"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	proposermetrics "github.com/ethereum-optimism/optimism/op-proposer/metrics"
	l2os "github.com/ethereum-optimism/optimism/op-proposer/proposer"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var (
	testingJWTSecret = [32]byte{123}
)

func newTxMgrConfig(l1Addr string, privKey *ecdsa.PrivateKey) txmgr.CLIConfig {
	return txmgr.CLIConfig{
		L1RPCURL:                  l1Addr,
		PrivateKey:                hexPriv(privKey),
		NumConfirmations:          1,
		SafeAbortNonceTooLowCount: 3,
		ResubmissionTimeout:       3 * time.Second,
		ReceiptQueryInterval:      50 * time.Millisecond,
		NetworkTimeout:            2 * time.Second,
		TxNotInMempoolTimeout:     2 * time.Minute,
	}
}

func DefaultSystemConfig(t *testing.T) SystemConfig {
	secrets, err := e2eutils.DefaultMnemonicConfig.Secrets()
	require.NoError(t, err)
	deployConfig := config.DeployConfig.Copy()
	deployConfig.L1GenesisBlockTimestamp = hexutil.Uint64(time.Now().Unix())
	require.NoError(t, deployConfig.Check())
	l1Deployments := config.L1Deployments.Copy()
	require.NoError(t, l1Deployments.Check())

	require.Equal(t, secrets.Addresses().Batcher, deployConfig.BatchSenderAddress)
	require.Equal(t, secrets.Addresses().SequencerP2P, deployConfig.P2PSequencerAddress)
	require.Equal(t, secrets.Addresses().Proposer, deployConfig.L2OutputOracleProposer)

	// Tests depend on premine being filled with secrets addresses
	premine := make(map[common.Address]*big.Int)
	for _, addr := range secrets.Addresses().All() {
		premine[addr] = new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))
	}

	return SystemConfig{
		Secrets:                secrets,
		Premine:                premine,
		DeployConfig:           deployConfig,
		L1Deployments:          config.L1Deployments,
		L1InfoPredeployAddress: predeploys.L1BlockAddr,
		JWTFilePath:            writeDefaultJWT(t),
		JWTSecret:              testingJWTSecret,
		Nodes: map[string]*rollupNode.Config{
			"sequencer": {
				Driver: driver.Config{
					VerifierConfDepth:  0,
					SequencerConfDepth: 0,
					SequencerEnabled:   true,
				},
				// Submitter PrivKey is set in system start for rollup nodes where sequencer = true
				RPC: rollupNode.RPCConfig{
					ListenAddr:  "127.0.0.1",
					ListenPort:  0,
					EnableAdmin: true,
				},
				L1EpochPollInterval: time.Second * 2,
				ConfigPersistence:   &rollupNode.DisabledConfigPersistence{},
			},
			"verifier": {
				Driver: driver.Config{
					VerifierConfDepth:  0,
					SequencerConfDepth: 0,
					SequencerEnabled:   false,
				},
				L1EpochPollInterval: time.Second * 4,
				ConfigPersistence:   &rollupNode.DisabledConfigPersistence{},
			},
		},
		Loggers: map[string]log.Logger{
			"verifier":  testlog.Logger(t, log.LvlInfo).New("role", "verifier"),
			"sequencer": testlog.Logger(t, log.LvlInfo).New("role", "sequencer"),
			"batcher":   testlog.Logger(t, log.LvlInfo).New("role", "batcher"),
			"proposer":  testlog.Logger(t, log.LvlCrit).New("role", "proposer"),
		},
		GethOptions:                map[string][]GethOption{},
		P2PTopology:                nil, // no P2P connectivity by default
		NonFinalizedProposals:      false,
		BatcherTargetL1TxSizeBytes: 100_000,
	}
}

func writeDefaultJWT(t *testing.T) string {
	// Sadly the geth node config cannot load JWT secret from memory, it has to be a file
	jwtPath := path.Join(t.TempDir(), "jwt_secret")
	if err := os.WriteFile(jwtPath, []byte(hexutil.Encode(testingJWTSecret[:])), 0600); err != nil {
		t.Fatalf("failed to prepare jwt file for geth: %v", err)
	}
	return jwtPath
}

type DepositContractConfig struct {
	L2Oracle           common.Address
	FinalizationPeriod *big.Int
}

type SystemConfig struct {
	Secrets                *e2eutils.Secrets
	L1InfoPredeployAddress common.Address

	DeployConfig  *genesis.DeployConfig
	L1Deployments *genesis.L1Deployments

	JWTFilePath string
	JWTSecret   [32]byte

	Premine        map[common.Address]*big.Int
	Nodes          map[string]*rollupNode.Config // Per node config. Don't use populate rollup.Config
	Loggers        map[string]log.Logger
	GethOptions    map[string][]GethOption
	ProposerLogger log.Logger
	BatcherLogger  log.Logger

	// map of outbound connections to other nodes. Node names prefixed with "~" are unconnected but linked.
	// A nil map disables P2P completely.
	// Any node name not in the topology will not have p2p enabled.
	P2PTopology map[string][]string

	// Enables req-resp sync in the P2P nodes
	P2PReqRespSync bool

	// If the proposer can make proposals for L2 blocks derived from L1 blocks which are not finalized on L1 yet.
	NonFinalizedProposals bool

	// Explicitly disable batcher, for tests that rely on unsafe L2 payloads
	DisableBatcher bool

	// Target L1 tx size for the batcher transactions
	BatcherTargetL1TxSizeBytes uint64

	// SupportL1TimeTravel determines if the L1 node supports quickly skipping forward in time
	SupportL1TimeTravel bool
}

type System struct {
	cfg SystemConfig

	RollupConfig *rollup.Config

	L2GenesisCfg *core.Genesis

	// Connections to running nodes
	Nodes             map[string]*node.Node
	Backends          map[string]*geth_eth.Ethereum
	Clients           map[string]*ethclient.Client
	RollupNodes       map[string]*rollupNode.OpNode
	L2OutputSubmitter *l2os.L2OutputSubmitter
	BatchSubmitter    *bss.BatchSubmitter
	Mocknet           mocknet.Mocknet

	// TimeTravelClock is nil unless SystemConfig.SupportL1TimeTravel was set to true
	// It provides access to the clock instance used by the L1 node. Calling TimeTravelClock.AdvanceBy
	// allows tests to quickly time travel L1 into the future.
	// Note that this time travel may occur in a single block, creating a very large difference in the Time
	// on sequential blocks.
	TimeTravelClock *clock.AdvancingClock
}

func (sys *System) NodeEndpoint(name string) string {
	return selectEndpoint(sys.Nodes[name])
}

func (sys *System) Close() {
	if sys.L2OutputSubmitter != nil {
		sys.L2OutputSubmitter.Stop()
	}
	if sys.BatchSubmitter != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		sys.BatchSubmitter.StopIfRunning(ctx)
	}

	for _, node := range sys.RollupNodes {
		node.Close()
	}
	for _, node := range sys.Nodes {
		node.Close()
	}
	sys.Mocknet.Close()
}

type systemConfigHook func(sCfg *SystemConfig, s *System)

type SystemConfigOption struct {
	key    string
	role   string
	action systemConfigHook
}

type SystemConfigOptions struct {
	opts map[string]systemConfigHook
}

func NewSystemConfigOptions(_opts []SystemConfigOption) (SystemConfigOptions, error) {
	opts := make(map[string]systemConfigHook)
	for _, opt := range _opts {
		if _, ok := opts[opt.key+":"+opt.role]; ok {
			return SystemConfigOptions{}, fmt.Errorf("duplicate option for key %s and role %s", opt.key, opt.role)
		}
		opts[opt.key+":"+opt.role] = opt.action
	}

	return SystemConfigOptions{
		opts: opts,
	}, nil
}

func (s *SystemConfigOptions) Get(key, role string) (systemConfigHook, bool) {
	v, ok := s.opts[key+":"+role]
	return v, ok
}

func (cfg SystemConfig) Start(_opts ...SystemConfigOption) (*System, error) {
	opts, err := NewSystemConfigOptions(_opts)
	if err != nil {
		return nil, err
	}

	sys := &System{
		cfg:         cfg,
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

	c := clock.SystemClock
	if cfg.SupportL1TimeTravel {
		sys.TimeTravelClock = clock.NewAdvancingClock(100 * time.Millisecond)
		c = sys.TimeTravelClock
	}

	if err := cfg.DeployConfig.Check(); err != nil {
		return nil, err
	}

	l1Genesis, err := genesis.BuildL1DeveloperGenesis(cfg.DeployConfig, config.L1Allocs, config.L1Deployments, true)
	if err != nil {
		return nil, err
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
	l2Genesis, err := genesis.BuildL2Genesis(cfg.DeployConfig, l1Block)
	if err != nil {
		return nil, err
	}
	sys.L2GenesisCfg = l2Genesis
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
			DepositContractAddress: cfg.DeployConfig.OptimismPortalProxy,
			L1SystemConfigAddress:  cfg.DeployConfig.SystemConfigProxy,
			RegolithTime:           cfg.DeployConfig.RegolithTime(uint64(cfg.DeployConfig.L1GenesisBlockTimestamp)),
		}
	}
	defaultConfig := makeRollupConfig()
	if err := defaultConfig.Check(); err != nil {
		return nil, err
	}
	sys.RollupConfig = &defaultConfig

	// Initialize nodes
	l1Node, l1Backend, err := initL1Geth(&cfg, l1Genesis, c, cfg.GethOptions["l1"]...)
	if err != nil {
		return nil, err
	}
	sys.Nodes["l1"] = l1Node
	sys.Backends["l1"] = l1Backend

	for name := range cfg.Nodes {
		node, backend, err := initL2Geth(name, big.NewInt(int64(cfg.DeployConfig.L2ChainID)), l2Genesis, cfg.JWTFilePath, cfg.GethOptions[name]...)
		if err != nil {
			return nil, err
		}
		sys.Nodes[name] = node
		sys.Backends[name] = backend
	}

	// Start
	err = l1Node.Start()
	if err != nil {
		didErrAfterStart = true
		return nil, err
	}
	for name, node := range sys.Nodes {
		if name == "l1" {
			continue
		}
		err = node.Start()
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
	}

	// Configure connections to L1 and L2 for rollup nodes.
	// TODO: refactor testing to use in-process rpc connections instead of websockets.

	for name, rollupCfg := range cfg.Nodes {
		configureL1(rollupCfg, l1Node)
		configureL2(rollupCfg, sys.Nodes[name], cfg.JWTSecret)

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
		return nil, err
	}
	l1Client := ethclient.NewClient(rpc.DialInProc(l1Srv))
	sys.Clients["l1"] = l1Client
	for name, node := range sys.Nodes {
		client, err := ethclient.DialContext(ctx, node.WSEndpoint())
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		sys.Clients[name] = client
	}

	_, err = waitForBlock(big.NewInt(2), l1Client, 6*time.Second*time.Duration(cfg.DeployConfig.L1BlockTime))
	if err != nil {
		return nil, fmt.Errorf("waiting for blocks: %w", err)
	}

	sys.Mocknet = mocknet.New()

	p2pNodes := make(map[string]*p2p.Prepared)
	if cfg.P2PTopology != nil {
		// create the peer if it doesn't exist yet.
		initHostMaybe := func(name string) (*p2p.Prepared, error) {
			if p, ok := p2pNodes[name]; ok {
				return p, nil
			}
			h, err := sys.newMockNetPeer()
			if err != nil {
				return nil, fmt.Errorf("failed to init p2p host for node %s", name)
			}
			h.Network()
			_, ok := cfg.Nodes[name]
			if !ok {
				return nil, fmt.Errorf("node %s from p2p topology not found in actual nodes map", name)
			}
			// TODO we can enable discv5 in the testnodes to test discovery of new peers.
			// Would need to mock though, and the discv5 implementation does not provide nice mocks here.
			p := &p2p.Prepared{
				HostP2P:           h,
				LocalNode:         nil,
				UDPv5:             nil,
				EnableReqRespSync: cfg.P2PReqRespSync,
			}
			p2pNodes[name] = p
			return p, nil
		}
		for k, vs := range cfg.P2PTopology {
			peerA, err := initHostMaybe(k)
			if err != nil {
				return nil, fmt.Errorf("failed to setup mocknet peer %s", k)
			}
			for _, v := range vs {
				v = strings.TrimPrefix(v, "~")
				peerB, err := initHostMaybe(v)
				if err != nil {
					return nil, fmt.Errorf("failed to setup mocknet peer %s (peer of %s)", v, k)
				}
				if _, err := sys.Mocknet.LinkPeers(peerA.HostP2P.ID(), peerB.HostP2P.ID()); err != nil {
					return nil, fmt.Errorf("failed to setup mocknet link between %s and %s", k, v)
				}
				// connect the peers after starting the full rollup node
			}
		}
	}

	// Don't log state snapshots in test output
	snapLog := log.New()
	snapLog.SetHandler(log.DiscardHandler())

	// Rollup nodes

	// Ensure we are looping through the nodes in alphabetical order
	ks := make([]string, 0, len(cfg.Nodes))
	for k := range cfg.Nodes {
		ks = append(ks, k)
	}
	// Sort strings in ascending alphabetical order
	sort.Strings(ks)

	for _, name := range ks {
		nodeConfig := cfg.Nodes[name]
		c := *nodeConfig // copy
		c.Rollup = makeRollupConfig()
		if err := c.LoadPersisted(cfg.Loggers[name]); err != nil {
			return nil, err
		}

		if p, ok := p2pNodes[name]; ok {
			c.P2P = p

			if c.Driver.SequencerEnabled && c.P2PSigner == nil {
				c.P2PSigner = &p2p.PreparedSigner{Signer: p2p.NewLocalSigner(cfg.Secrets.SequencerP2P)}
			}
		}

		c.Rollup.LogDescription(cfg.Loggers[name], chaincfg.L2ChainIDToNetworkName)

		node, err := rollupNode.New(context.Background(), &c, cfg.Loggers[name], snapLog, "", metrics.NewMetrics(""))
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		err = node.Start(context.Background())
		if err != nil {
			didErrAfterStart = true
			return nil, err
		}
		sys.RollupNodes[name] = node

		if action, ok := opts.Get("afterRollupNodeStart", name); ok {
			action(&cfg, sys)
		}
	}

	if cfg.P2PTopology != nil {
		// We only set up the connections after starting the actual nodes,
		// so GossipSub and other p2p protocols can be started before the connections go live.
		// This way protocol negotiation happens correctly.
		for k, vs := range cfg.P2PTopology {
			peerA := p2pNodes[k]
			for _, v := range vs {
				unconnected := strings.HasPrefix(v, "~")
				if unconnected {
					v = v[1:]
				}
				if !unconnected {
					peerB := p2pNodes[v]
					if _, err := sys.Mocknet.ConnectPeers(peerA.HostP2P.ID(), peerB.HostP2P.ID()); err != nil {
						return nil, fmt.Errorf("failed to setup mocknet connection between %s and %s", k, v)
					}
				}
			}
		}
	}

	// Don't start batch submitter and proposer if there's no sequencer.
	if sys.RollupNodes["sequencer"] == nil {
		return sys, nil
	}

	// L2Output Submitter
	sys.L2OutputSubmitter, err = l2os.NewL2OutputSubmitterFromCLIConfig(l2os.CLIConfig{
		L1EthRpc:          sys.Nodes["l1"].WSEndpoint(),
		RollupRpc:         sys.RollupNodes["sequencer"].HTTPEndpoint(),
		L2OOAddress:       config.L1Deployments.L2OutputOracleProxy.Hex(),
		PollInterval:      50 * time.Millisecond,
		TxMgrConfig:       newTxMgrConfig(sys.Nodes["l1"].WSEndpoint(), cfg.Secrets.Proposer),
		AllowNonFinalized: cfg.NonFinalizedProposals,
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
	}, sys.cfg.Loggers["proposer"], proposermetrics.NoopMetrics)
	if err != nil {
		return nil, fmt.Errorf("unable to setup l2 output submitter: %w", err)
	}

	if err := sys.L2OutputSubmitter.Start(); err != nil {
		return nil, fmt.Errorf("unable to start l2 output submitter: %w", err)
	}

	// Batch Submitter
	sys.BatchSubmitter, err = bss.NewBatchSubmitterFromCLIConfig(bss.CLIConfig{
		L1EthRpc:               sys.Nodes["l1"].WSEndpoint(),
		L2EthRpc:               sys.Nodes["sequencer"].WSEndpoint(),
		RollupRpc:              sys.RollupNodes["sequencer"].HTTPEndpoint(),
		MaxPendingTransactions: 0,
		MaxChannelDuration:     1,
		MaxL1TxSize:            240_000,
		CompressorConfig: compressor.CLIConfig{
			TargetL1TxSizeBytes: cfg.BatcherTargetL1TxSizeBytes,
			TargetNumFrames:     1,
			ApproxComprRatio:    0.4,
		},
		SubSafetyMargin: 4,
		PollInterval:    50 * time.Millisecond,
		TxMgrConfig:     newTxMgrConfig(sys.Nodes["l1"].WSEndpoint(), cfg.Secrets.Batcher),
		LogConfig: oplog.CLIConfig{
			Level:  "info",
			Format: "text",
		},
	}, sys.cfg.Loggers["batcher"], batchermetrics.NoopMetrics)
	if err != nil {
		return nil, fmt.Errorf("failed to setup batch submitter: %w", err)
	}

	// Batcher may be enabled later
	if !sys.cfg.DisableBatcher {
		if err := sys.BatchSubmitter.Start(); err != nil {
			return nil, fmt.Errorf("unable to start batch submitter: %w", err)
		}
	}

	return sys, nil
}

// IP6 range that gets blackholed (in case our traffic ever makes it out onto
// the internet).
var blackholeIP6 = net.ParseIP("100::")

// mocknet doesn't allow us to add a peerstore without fully creating the peer ourselves
func (sys *System) newMockNetPeer() (host.Host, error) {
	sk, _, err := ic.GenerateECDSAKeyPair(rand.Reader)
	if err != nil {
		return nil, err
	}
	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return nil, err
	}
	suffix := id
	if len(id) > 8 {
		suffix = id[len(id)-8:]
	}
	ip := append(net.IP{}, blackholeIP6...)
	copy(ip[net.IPv6len-len(suffix):], suffix)
	a, err := ma.NewMultiaddr(fmt.Sprintf("/ip6/%s/tcp/4242", ip))
	if err != nil {
		return nil, fmt.Errorf("failed to create test multiaddr: %w", err)
	}
	p, err := peer.IDFromPublicKey(sk.GetPublic())
	if err != nil {
		return nil, err
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, err
	}
	ps.AddAddr(p, a, peerstore.PermanentAddrTTL)
	_ = ps.AddPrivKey(p, sk)
	_ = ps.AddPubKey(p, sk.GetPublic())

	ds := sync.MutexWrap(ds.NewMapDatastore())
	eps, err := store.NewExtendedPeerstore(context.Background(), log.Root(), clock.SystemClock, ps, ds, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return sys.Mocknet.AddPeerWithPeerstore(p, eps)
}

func selectEndpoint(node *node.Node) string {
	useHTTP := os.Getenv("OP_E2E_USE_HTTP") == "true"
	if useHTTP {
		log.Info("using HTTP client")
		return node.HTTPEndpoint()
	}
	return node.WSEndpoint()
}

func configureL1(rollupNodeCfg *rollupNode.Config, l1Node *node.Node) {
	l1EndpointConfig := selectEndpoint(l1Node)
	rollupNodeCfg.L1 = &rollupNode.L1EndpointConfig{
		L1NodeAddr:       l1EndpointConfig,
		L1TrustRPC:       false,
		L1RPCKind:        sources.RPCKindBasic,
		RateLimit:        0,
		BatchSize:        20,
		HttpPollInterval: time.Millisecond * 100,
	}
}
func configureL2(rollupNodeCfg *rollupNode.Config, l2Node *node.Node, jwtSecret [32]byte) {
	useHTTP := os.Getenv("OP_E2E_USE_HTTP") == "true"
	l2EndpointConfig := l2Node.WSAuthEndpoint()
	if useHTTP {
		l2EndpointConfig = l2Node.HTTPAuthEndpoint()
	}

	rollupNodeCfg.L2 = &rollupNode.L2EndpointConfig{
		L2EngineAddr:      l2EndpointConfig,
		L2EngineJWTSecret: jwtSecret,
	}
}

func (cfg SystemConfig) L1ChainIDBig() *big.Int {
	return new(big.Int).SetUint64(cfg.DeployConfig.L1ChainID)
}

func (cfg SystemConfig) L2ChainIDBig() *big.Int {
	return new(big.Int).SetUint64(cfg.DeployConfig.L2ChainID)
}

func hexPriv(in *ecdsa.PrivateKey) string {
	b := e2eutils.EncodePrivKey(in)
	return hexutil.Encode(b)
}
