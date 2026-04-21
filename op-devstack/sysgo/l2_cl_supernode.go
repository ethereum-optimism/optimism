package sysgo

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	snconfig "github.com/ethereum-optimism/optimism/op-supernode/config"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
)

var errSupernodeNotRunning = errors.New("sysgo: supernode is not running")

type SuperNode struct {
	mu               sync.Mutex
	sn               *supernode.Supernode
	cancel           context.CancelFunc
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
	if n.sn != nil {
		n.logger.Warn("Supernode already started")
		return
	}

	n.p.Require().NotNil(n.snCfg, "supernode CLI config required")

	ctx, cancel := context.WithCancel(n.p.Ctx())
	exitFn := func(err error) { n.p.Errorf("supernode critical error: %v", err) }
	sn, err := supernode.New(ctx, n.logger, "devstack", exitFn, n.snCfg, n.vnCfgs)
	n.p.Require().NoError(err, "supernode failed to create")
	n.sn = sn
	n.cancel = cancel

	n.p.Require().NoError(n.sn.Start(ctx))

	addr, err := n.sn.WaitRPCAddr(ctx)
	n.p.Require().NoError(err, "supernode failed to bind RPC address")
	base := "http://" + addr
	n.userRPC = base
	n.interopEndpoint = base
}

func (n *SuperNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		n.logger.Warn("Supernode already stopped")
		return
	}
	if n.cancel != nil {
		n.cancel()
	}
	// Attempt graceful stop
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = n.sn.Stop(stopCtx)
	n.sn = nil
}

// InteropActivity returns the interop activity running inside the supernode,
// or nil if the supernode is stopped or has no interop activity. Callers must
// not cache the returned pointer across RestartInteropActivity, which swaps
// the activity for a fresh instance. For integration test control only.
func (n *SuperNode) InteropActivity() *interop.Interop {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		return nil
	}
	return n.sn.InteropActivity()
}

// RestartInteropActivity stops the running interop activity, optionally
// wipes its on-disk logs DBs, and launches a fresh instance against the
// still-running supernode. For integration test control only.
func (n *SuperNode) RestartInteropActivity(wipeLogsDBs bool) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.sn == nil {
		return errSupernodeNotRunning
	}
	return n.sn.RestartInteropActivity(wipeLogsDBs)
}

// SuperNodeProxy is a thin wrapper that points to a shared supernode instance.
type SuperNodeProxy struct {
	p                devtest.CommonT
	logger           log.Logger
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
}

var _ L2CLNode = (*SuperNodeProxy)(nil)

func (n *SuperNodeProxy) Start()          {}
func (n *SuperNodeProxy) Stop()           {}
func (n *SuperNodeProxy) UserRPC() string { return n.userRPC }
func (n *SuperNodeProxy) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	return n.interopEndpoint, n.interopJwtSecret
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
