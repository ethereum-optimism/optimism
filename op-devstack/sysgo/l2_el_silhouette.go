package sysgo

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum-optimism/optimism/op-supernode/silhouette"
)

// SilhouetteEL is a standalone, restartable execution-layer component. It occupies the same
// engine-RPC slot as op-reth beneath an otherwise stock verifier, but renders proof-derived public
// blocks instead of executing private transactions.
type SilhouetteEL struct {
	mu sync.Mutex

	p         devtest.CommonT
	logger    log.Logger
	supernode *SuperNode
	rollup    *rollup.Config
	cfg       silhouette.Config
	privateEL L2ELNode
	jwtSecret [32]byte
	dataDir   string

	proxy   *tcpproxy.Proxy
	rpcURL  string
	server  *oprpc.Server
	facts   *silhouette.FactStore
	cancel  context.CancelFunc
	done    chan struct{}
	private client.RPC

	deniedChecker func(uint64, common.Hash) (bool, error)
}

var _ L2ELNode = (*SilhouetteEL)(nil)

func NewSilhouetteEL(p devtest.CommonT, logger log.Logger, supernode *SuperNode, rollupCfg *rollup.Config,
	cfg silhouette.Config, privateEL L2ELNode, jwtSecret [32]byte, dataDir string,
) *SilhouetteEL {
	return &SilhouetteEL{
		p: p, logger: logger, supernode: supernode, rollup: rollupCfg, cfg: cfg,
		privateEL: privateEL, jwtSecret: jwtSecret, dataDir: dataDir,
	}
}

func (n *SilhouetteEL) UserRPC() string   { return n.rpcURL }
func (n *SilhouetteEL) EngineRPC() string { return n.rpcURL }
func (n *SilhouetteEL) JWTPath() string   { return "" }

func (n *SilhouetteEL) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.server != nil {
		n.logger.Warn("Silhouette EL already started")
		return
	}
	require := n.p.Require()
	require.NotNil(n.supernode.L1Client(), "Silhouette EL needs the runtime's shared L1 client")
	require.NotNil(n.supernode.BeaconClient(), "Silhouette EL needs the runtime's shared beacon client")
	if n.proxy == nil {
		n.proxy = tcpproxy.New(n.logger.New("proxy", "silhouette-el"))
		require.NoError(n.proxy.Start())
		n.p.Cleanup(func() { _ = n.proxy.Close() })
		n.rpcURL = "http://" + n.proxy.Addr()
	}

	l1Chain, err := silhouette.L1ChainConfig(&n.cfg)
	require.NoError(err)
	verifier, err := n.cfg.NewVerifier()
	require.NoError(err)
	facts, err := silhouette.OpenFactStore(n.dataDir)
	require.NoError(err, "open persistent Silhouette EL")
	if n.deniedChecker != nil {
		facts.SetDeniedChecker(func(number uint64, hash common.Hash) (bool, error) {
			return n.deniedChecker(number, hash)
		})
	}
	source := silhouette.NewDataSource(n.logger.New("component", "proof-source"), &n.cfg, n.rollup,
		l1Chain, n.rollup.Genesis.SystemConfig, n.supernode.L1Client(), n.supernode.BeaconClient(), verifier, facts)
	shim := silhouette.NewShim(n.logger.New("component", "engine"), n.rollup, l1Chain,
		n.rollup.Genesis.SystemConfig, n.supernode.L1Client(), facts)
	privateRPC, err := client.NewRPC(n.p.Ctx(), n.logger, n.privateEL.EngineRPC(),
		client.WithGethRPCOptions(gethrpc.WithHTTPAuth(gn.NewJWTAuth(n.jwtSecret))),
		client.WithCallTimeout(30*time.Second))
	if err != nil {
		_ = facts.Close()
		require.NoError(err, "dial private replacement engine")
	}
	replacementEngine, err := sources.NewEngineClient(privateRPC, n.logger.New("component", "replacement-engine"),
		nil, sources.EngineClientDefaultConfig(n.rollup))
	if err != nil {
		privateRPC.Close()
		_ = facts.Close()
		require.NoError(err, "create private replacement engine client")
	}
	shim.SetReplacementBuilder(silhouette.NewEngineReplacementBuilder(replacementEngine))
	start := n.cfg.L1StartBlock
	if start == 0 {
		start = n.rollup.Genesis.L1.Number
	}
	tracker := silhouette.NewProvenHeadTracker(n.logger.New("component", "proof-walker"),
		source, n.supernode.L1Client(), start, 100*time.Millisecond)
	server := shim.Standalone("127.0.0.1", 0)
	if err := server.Start(); err != nil {
		privateRPC.Close()
		_ = facts.Close()
		require.NoError(err)
	}
	ctx, cancel := context.WithCancel(n.p.Ctx())
	done := make(chan struct{})
	go func() {
		defer close(done)
		tracker.Run(ctx)
	}()
	n.proxy.SetUpstream(ProxyAddr(require, "http://"+server.Endpoint()))
	n.server, n.facts, n.cancel, n.done, n.private = server, facts, cancel, done, privateRPC
	n.logger.Info("started persistent Silhouette EL", "rpc", n.rpcURL, "chain", n.rollup.L2ChainID,
		"data_dir", n.dataDir, "l1_cursor", tracker.Cursor())
}

func (n *SilhouetteEL) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.server == nil {
		n.logger.Warn("Silhouette EL already stopped")
		return
	}
	n.proxy.ClearUpstream()
	n.cancel()
	_ = n.server.Stop()
	select {
	case <-n.done:
	case <-time.After(5 * time.Second):
		n.p.Errorf("Silhouette EL proof walker did not stop within 5s")
	}
	n.private.Close()
	n.p.Require().NoError(n.facts.Close(), "close persistent Silhouette EL")
	n.server, n.facts, n.cancel, n.done, n.private = nil, nil, nil, nil, nil
	n.logger.Info("stopped persistent Silhouette EL")
}

func (n *SilhouetteEL) Running() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.server != nil
}

func (n *SilhouetteEL) SetDeniedChecker(checker func(uint64, common.Hash) (bool, error)) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.deniedChecker = checker
	if n.facts != nil {
		n.facts.SetDeniedChecker(func(number uint64, hash common.Hash) (bool, error) {
			return checker(number, hash)
		})
	}
}
