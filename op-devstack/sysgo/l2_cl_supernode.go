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

var errSupernodeNotRunning = errors.New("sysgo: supernode is not running")

type SuperNode struct {
	mu           sync.Mutex
	sn           *supernode.Supernode
	cancel       context.CancelFunc
	httpProxy    *tcpproxy.Proxy
	userRPC      string
	p            devtest.CommonT
	logger       log.Logger
	chains       []eth.ChainID
	l1UserRPC    string
	l1BeaconAddr string

	// Configs stored for Start()/restart.
	snCfg  *snconfig.CLIConfig
	vnCfgs map[eth.ChainID]*config.Config
}

var _ L2CLNode = (*SuperNode)(nil)

func (n *SuperNode) UserRPC() string {
	return n.userRPC
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
		n.userRPC = "http://" + n.httpProxy.Addr()
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
// instance; do not cache across RestartWithFreshDataDir. Test-only.
func (n *SuperNode) InteropActivity() *interop.Interop {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		return nil
	}
	return n.sn.InteropActivity()
}

// RestartWithFreshDataDir stops the supernode, deletes its on-disk data
// directory, and starts a fresh supernode against the same chain
// containers, virtual nodes, and externally-visible RPC address. Test-only.
func (n *SuperNode) RestartWithFreshDataDir() error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		return errSupernodeNotRunning
	}
	if n.snCfg == nil || n.snCfg.DataDir == "" {
		return errors.New("sysgo: RestartWithFreshDataDir requires a configured supernode DataDir")
	}
	n.logger.Info("restarting supernode with fresh data dir", "data_dir", n.snCfg.DataDir)
	n.stopLocked()
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
	p       devtest.CommonT
	logger  log.Logger
	userRPC string
}

var _ L2CLNode = (*SuperNodeProxy)(nil)

func (n *SuperNodeProxy) Start()          {}
func (n *SuperNodeProxy) Stop()           {}
func (n *SuperNodeProxy) UserRPC() string { return n.userRPC }

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
