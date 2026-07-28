package sysgo

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-node/config"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

type OpNode struct {
	mu sync.Mutex

	name      string
	opNode    *opnode.Opnode
	userRPC   string
	cfg       *config.Config
	syncMode  nodeSync.Mode
	p         devtest.CommonT
	logger    log.Logger
	userProxy *tcpproxy.Proxy
	clock     clock.Clock
}

// SyncMode reports the sync mode the op-node was started with
// (VerifierSyncMode for verifiers, SequencerSyncMode for sequencers).
func (n *OpNode) SyncMode() nodeSync.Mode {
	return n.syncMode
}

var _ L2CLNode = (*OpNode)(nil)

func (n *OpNode) UserRPC() string {
	return n.userRPC
}

func (n *OpNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode != nil {
		n.logger.Warn("Op-node already started")
		return
	}

	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.logger.New("proxy", "l2cl-user"))
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "http://" + n.userProxy.Addr()
	}
	n.logger.Info("Starting op-node")
	opNode, err := opnode.NewOpnode(n.logger, n.cfg, n.clock, func(err error) {
		// Use Errorf (non-fatal) instead of Require().NoError (fatal) because this
		// callback is invoked from the event-processing goroutine, not the test
		// goroutine. Require/FailNow calls runtime.Goexit() which would only exit
		// the event goroutine, not the test, leaving the test to hang until timeout.
		n.p.Errorf("op-node critical error: %v", err)
	})
	n.p.Require().NoError(err, "op-node failed to start")
	n.logger.Info("Started op-node")
	n.opNode = opNode

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), opNode.UserRPC().RPC()))
}

func (n *OpNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode == nil {
		n.logger.Warn("Op-node already stopped")
		return
	}
	n.clearProxyUpstreams()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	n.logger.Info("Closing op-node")
	closeErr := n.opNode.Stop(ctx)
	n.logger.Info("Closed op-node", "err", closeErr)

	n.opNode = nil
}

func (n *OpNode) StartControlled(ctx context.Context) error {
	return runControlStart(ctx, n.Running, n.Start)
}

func (n *OpNode) StopControlled(ctx context.Context) error {
	n.mu.Lock()
	if n.opNode == nil {
		n.mu.Unlock()
		return nil
	}
	opNode := n.opNode
	n.clearProxyUpstreams() // before Stop: the node frees its ports during shutdown
	n.mu.Unlock()

	err := opNode.Stop(ctx)
	if err != nil && !opNode.Stopped() {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode == opNode {
		n.opNode = nil
	}
	return nil
}

// clearProxyUpstreams unsets the proxy upstreams. It must run before the
// node is asked to stop: the node frees its OS-assigned ports early in
// shutdown, and they may be rebound by another process, so a stale upstream
// would silently cross-wire this node's endpoints to it. Callers must hold n.mu.
func (n *OpNode) clearProxyUpstreams() {
	if n.userProxy != nil {
		n.userProxy.ClearUpstream()
	}
}

func (n *OpNode) Running() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.opNode != nil
}
