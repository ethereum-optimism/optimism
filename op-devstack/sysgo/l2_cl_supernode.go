package sysgo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

type SuperNode struct {
	mu sync.Mutex
	peerRegistry
	// Per-chain RPC routes to wait on after (re)start before considering the
	// supernode ready for peer-connector replay. Populated when a
	// SuperNodeProxy is attached.
	routes           []string
	sn               *supernode.Supernode
	cancel           context.CancelFunc
	httpProxy        *tcpproxy.Proxy
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	p                devtest.CommonT
	logger           log.Logger
	chains           []eth.ChainID
	l1UserRPC        string
	l1BeaconAddr     string

	// Configs stored for Start()/restart.
	snCfg  *snconfig.CLIConfig
	vnCfgs map[eth.ChainID]*config.Config
}

var _ L2CLNode = (*SuperNode)(nil)

func (n *SuperNode) UserRPC() string {
	return n.userRPC
}

func (n *SuperNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

func (n *SuperNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.startLocked()
}

// startLocked brings up the supernode and points the long-lived httpProxy
// at its newly-bound RPC port. The proxy is created on first start and
// reused so external callers see a stable URL across restarts. Caller must
// hold n.mu.
func (n *SuperNode) startLocked() {
	if n.sn != nil {
		n.logger.Warn("Supernode already started")
		return
	}

	n.p.Require().NotNil(n.snCfg, "supernode CLI config required")

	if n.httpProxy == nil {
		n.httpProxy = tcpproxy.New(n.logger.New("proxy", "supernode-http"))
		n.p.Require().NoError(n.httpProxy.Start(), "supernode http proxy failed to start")
		n.p.Cleanup(func() {
			_ = n.httpProxy.Close()
		})
		base := "http://" + n.httpProxy.Addr()
		n.userRPC = base
		n.interopEndpoint = base
	}

	ctx, cancel := context.WithCancel(n.p.Ctx())
	exitFn := func(err error) { n.p.Errorf("supernode critical error: %v", err) }
	sn, err := supernode.New(ctx, n.logger, "devstack", exitFn, n.snCfg, n.vnCfgs)
	n.p.Require().NoError(err, "supernode failed to create")
	n.sn = sn
	n.cancel = cancel

	n.p.Require().NoError(n.sn.Start(ctx))

	addr, err := n.sn.WaitRPCAddr(ctx)
	n.p.Require().NoError(err, "supernode failed to bind RPC address")
	n.httpProxy.SetUpstream(ProxyAddr(n.p.Require(), "http://"+addr))

	for _, route := range n.routes {
		waitForSupernodeRoute(n.p, n.logger, route)
	}
	for _, connect := range n.snapshotConnectors() {
		connect()
	}
}

func (n *SuperNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.stopLocked()
}

// stopLocked tears down the supernode instance, leaving httpProxy in place
// so a later startLocked can repoint it. Caller must hold n.mu.
func (n *SuperNode) stopLocked() {
	if n.sn == nil {
		n.logger.Warn("Supernode already stopped")
		return
	}
	if n.cancel != nil {
		n.cancel()
		n.cancel = nil
	}
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = n.sn.Stop(stopCtx)
	n.sn = nil
}

// InteropActivity returns the interop activity, or nil if the supernode is
// stopped or has no interop activity. The pointer is bound to the current
// instance; do not cache across StartWithFreshDataDir. Test-only.
func (n *SuperNode) InteropActivity() *interop.Interop {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		return nil
	}
	return n.sn.InteropActivity()
}

// StartWithFreshDataDir wipes the data dir and starts a fresh supernode.
// Pairs with Stop. Test-only.
func (n *SuperNode) StartWithFreshDataDir() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn != nil {
		return errors.New("sysgo: StartWithFreshDataDir called while supernode is still running")
	}
	return n.wipeDataDirAndStartLocked()
}

// wipeDataDirAndStartLocked must be called with n.mu held and n.sn nil.
func (n *SuperNode) wipeDataDirAndStartLocked() error {
	if n.snCfg == nil || n.snCfg.DataDir == "" {
		return errors.New("sysgo: fresh data dir restart requires a configured supernode DataDir")
	}
	n.logger.Info("wiping supernode data dir", "data_dir", n.snCfg.DataDir)
	if err := os.RemoveAll(n.snCfg.DataDir); err != nil {
		return fmt.Errorf("sysgo: wipe supernode data dir %s: %w", n.snCfg.DataDir, err)
	}
	if err := os.MkdirAll(n.snCfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("sysgo: recreate supernode data dir %s: %w", n.snCfg.DataDir, err)
	}
	n.startLocked()
	return nil
}

// SuperNodeProxy is a thin wrapper that points to a shared supernode instance.
type SuperNodeProxy struct {
	p                devtest.CommonT
	logger           log.Logger
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32

	// superNode is the underlying supernode that owns this proxy's RPC route.
	// Peer connectors registered on the proxy are forwarded so they replay
	// when the supernode restarts.
	superNode *SuperNode
}

var _ L2CLNode = (*SuperNodeProxy)(nil)
var _ PeerRegistrar = (*SuperNodeProxy)(nil)

func (n *SuperNodeProxy) Start()          {}
func (n *SuperNodeProxy) Stop()           {}
func (n *SuperNodeProxy) UserRPC() string { return n.userRPC }
func (n *SuperNodeProxy) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
}

func (n *SuperNodeProxy) RegisterPeerConnector(connect func()) {
	n.superNode.attachRoute(n.userRPC)
	n.superNode.RegisterPeerConnector(connect)
}

// attachRoute records a per-chain RPC route that startLocked must wait on
// after (re)start, so a peer-connector replay never fires before the
// supernode is actually serving the chain.
func (n *SuperNode) attachRoute(rpcEndpoint string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	for _, existing := range n.routes {
		if existing == rpcEndpoint {
			return
		}
	}
	n.routes = append(n.routes, rpcEndpoint)
}

// SupernodeConfig holds configuration options for the shared supernode.
type SupernodeConfig struct {
	// InteropActivationTimestamp enables the interop activity at the given timestamp.
	// Set to nil to disable interop (default). Non-nil (including 0) enables interop.
	InteropActivationTimestamp *uint64

	// UseGenesisInterop, when true, sets InteropActivationTimestamp to the genesis
	// timestamp of the first configured chain at deploy time. Takes effect inside
	// withSharedSupernodeCLsImpl after deployment, when the genesis time is known.
	UseGenesisInterop bool
}

// SupernodeOption is a functional option for configuring the supernode.
type SupernodeOption func(*SupernodeConfig)

// WithSupernodeInterop enables the interop activity with the given activation timestamp.
func WithSupernodeInterop(activationTimestamp uint64) SupernodeOption {
	return func(cfg *SupernodeConfig) {
		ts := activationTimestamp
		cfg.InteropActivationTimestamp = &ts
	}
}

// WithSupernodeInteropAtGenesis enables interop at the genesis timestamp of the first
// configured chain. The timestamp is resolved after deployment, when genesis is known.
func WithSupernodeInteropAtGenesis() SupernodeOption {
	return func(cfg *SupernodeConfig) {
		cfg.UseGenesisInterop = true
	}
}
