package sysgo

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/opnode"
	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/log"
)

type OpNode struct {
	mu sync.Mutex

	name             string
	opNode           *opnode.Opnode
	userRPC          string
	interopEndpoint  string
	interopJwtSecret eth.Bytes32
	cfg              *config.Config
	p                devtest.CommonT
	logger           log.Logger
	userProxy        *tcpproxy.Proxy
	interopProxy     *tcpproxy.Proxy
	clock            clock.Clock
}

var _ L2CLNode = (*OpNode)(nil)

func (n *OpNode) UserRPC() string {
	return n.userRPC
}

func (n *OpNode) InteropRPC() (endpoint string, jwtSecret eth.Bytes32) {
	// Make sure to use the proxied interop endpoint
	return n.interopEndpoint, n.interopJwtSecret
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
	if n.interopProxy == nil {
		n.interopProxy = tcpproxy.New(n.logger.New("proxy", "l2cl-interop"))
		n.p.Require().NoError(n.interopProxy.Start())
		n.p.Cleanup(func() {
			n.interopProxy.Close()
		})
		n.interopEndpoint = "ws://" + n.interopProxy.Addr()
	}
	n.logger.Info("Starting op-node")
	opNode, err := opnode.NewOpnode(n.logger, n.cfg, n.clock, func(err error) {
		n.p.Require().NoError(err, "op-node critical error")
	})
	n.p.Require().NoError(err, "op-node failed to start")
	n.logger.Info("Started op-node")
	n.opNode = opNode

	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), opNode.UserRPC().RPC()))

	interopEndpoint, interopJwtSecret := opNode.InteropRPC()
	n.interopProxy.SetUpstream(ProxyAddr(n.p.Require(), interopEndpoint))
	n.interopJwtSecret = interopJwtSecret
}

func (n *OpNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.opNode == nil {
		n.logger.Warn("Op-node already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	n.logger.Info("Closing op-node")
	closeErr := n.opNode.Stop(ctx)
	n.logger.Info("Closed op-node", "err", closeErr)

	n.opNode = nil
}
