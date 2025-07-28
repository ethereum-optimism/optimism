package sysgo

import (
	"context"
	"net"
	"net/url"
	"slices"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
)

type L2ELNode struct {
	mu sync.Mutex

	p             devtest.P
	logger        log.Logger
	id            stack.L2ELNodeID
	l2Net         *L2Network
	jwtPath       string
	supervisorRPC string
	l2Geth        *geth.GethInstance

	authRPC string
	userRPC string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy
}

func (n *L2ELNode) hydrate(system stack.ExtensibleSystem) {
	require := system.T().Require()
	rpcCl, err := client.NewRPC(system.T().Ctx(), system.Logger(), n.userRPC, client.WithLazyDial())
	require.NoError(err)
	system.T().Cleanup(rpcCl.Close)

	l2Net := system.L2Network(stack.L2NetworkID(n.id.ChainID()))
	sysL2EL := shim.NewL2ELNode(shim.L2ELNodeConfig{
		RollupCfg: l2Net.RollupConfig(),
		ELNodeConfig: shim.ELNodeConfig{
			CommonConfig: shim.NewCommonConfig(system.T()),
			Client:       rpcCl,
			ChainID:      n.id.ChainID(),
		},
		ID: n.id,
	})
	sysL2EL.SetLabel(match.LabelVendor, string(match.OpGeth))
	l2Net.(stack.ExtensibleL2Network).AddL2ELNode(sysL2EL)
}

func (n *L2ELNode) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.l2Geth != nil {
		n.logger.Warn("op-geth already started")
		return
	}

	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.logger)
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.logger)
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})
		n.userRPC = "ws://" + n.userProxy.Addr()
	}

	require := n.p.Require()
	l2Geth, err := geth.InitL2(n.id.String(), n.l2Net.genesis, n.jwtPath,
		func(ethCfg *ethconfig.Config, nodeCfg *gn.Config) error {
			ethCfg.InteropMessageRPC = n.supervisorRPC
			ethCfg.InteropMempoolFiltering = true
			nodeCfg.P2P = p2p.Config{
				NoDiscovery: true,
				ListenAddr:  "127.0.0.1:0",
				MaxPeers:    10,
			}
			return nil
		})
	require.NoError(err)
	require.NoError(l2Geth.Node.Start())
	n.l2Geth = l2Geth
	n.authProxy.SetUpstream(proxyAddr(require, l2Geth.AuthRPC().RPC()))
	n.userProxy.SetUpstream(proxyAddr(require, l2Geth.UserRPC().RPC()))
}

func (n *L2ELNode) Stop() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.l2Geth == nil {
		n.logger.Warn("op-geth already stopped")
		return
	}
	n.logger.Info("Closing op-geth", "id", n.id)
	closeErr := n.l2Geth.Close()
	n.logger.Info("Closed op-geth", "id", n.id, "err", closeErr)
	n.l2Geth = nil
}

func proxyAddr(require *testreq.Assertions, urlStr string) string {
	u, err := url.Parse(urlStr)
	require.NoError(err)
	return net.JoinHostPort(u.Hostname(), u.Port())
}

func WithL2ELNode(id stack.L2ELNodeID, supervisorID *stack.SupervisorID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))

		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(id.ChainID())
		require.True(ok, "L2 network required")

		jwtPath, _ := orch.writeDefaultJWT()

		useInterop := l2Net.genesis.Config.InteropTime != nil

		supervisorRPC := ""
		if useInterop {
			require.NotNil(supervisorID, "supervisor is required for interop")
			sup, ok := orch.supervisors.Get(*supervisorID)
			require.True(ok, "supervisor is required for interop")
			supervisorRPC = sup.userRPC
		}

		logger := p.Logger()

		l2EL := &L2ELNode{
			id:            id,
			p:             orch.P(),
			logger:        logger,
			l2Net:         l2Net,
			jwtPath:       jwtPath,
			supervisorRPC: supervisorRPC,
		}
		l2EL.Start()
		p.Cleanup(func() {
			l2EL.Stop()
		})
		require.True(orch.l2ELs.SetIfMissing(id, l2EL), "must be unique L2 EL node")
	})
}

func WithL2ELP2PConnection(l2EL1ID, l2EL2ID stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		require := orch.P().Require()

		l2EL1, ok := orch.l2ELs.Get(l2EL1ID)
		require.True(ok, "looking for L2 EL node 1 to connect p2p")
		l2EL2, ok := orch.l2ELs.Get(l2EL2ID)
		require.True(ok, "looking for L2 EL node 2 to connect p2p")
		require.Equal(l2EL1.l2Net.rollupCfg.L2ChainID, l2EL2.l2Net.rollupCfg.L2ChainID, "must be same l2 chain")

		ctx := orch.P().Ctx()
		logger := orch.P().Logger()

		rpc1, err := dial.DialRPCClientWithTimeout(ctx, logger, l2EL1.userRPC)
		require.NoError(err, "failed to connect to el1 rpc")
		defer rpc1.Close()
		rpc2, err := dial.DialRPCClientWithTimeout(ctx, logger, l2EL2.userRPC)
		require.NoError(err, "failed to connect to el2 rpc")
		defer rpc2.Close()

		ConnectP2P(orch.P().Ctx(), require, rpc1, rpc2)
	})
}

type RpcCaller interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

// ConnectP2P creates a p2p peer connection between node1 and node2.
func ConnectP2P(ctx context.Context, require *testreq.Assertions, initiator RpcCaller, acceptor RpcCaller) {
	var targetInfo p2p.NodeInfo
	require.NoError(acceptor.CallContext(ctx, &targetInfo, "admin_nodeInfo"), "get node info")

	var peerAdded bool
	require.NoError(initiator.CallContext(ctx, &peerAdded, "admin_addPeer", targetInfo.Enode), "add peer")
	require.True(peerAdded, "should have added peer successfully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	err := wait.For(ctx, time.Second, func() (bool, error) {
		var peers []peer
		if err := initiator.CallContext(ctx, &peers, "admin_peers"); err != nil {
			return false, err
		}
		return slices.ContainsFunc(peers, func(p peer) bool {
			return p.ID == targetInfo.ID
		}), nil
	})
	require.NoError(err, "The peer was not connected")
}

type peer struct {
	ID string `json:"id"`
}
