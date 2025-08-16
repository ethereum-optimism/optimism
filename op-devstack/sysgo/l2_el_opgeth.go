package sysgo

import (
	"sync"

	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	gn "github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/geth"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type OpGeth struct {
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

var _ L2ELNode = (*OpGeth)(nil)

func (n *OpGeth) UserRPC() string {
	return n.userRPC
}

func (n *OpGeth) EngineRPC() string {
	return n.authRPC
}

func (n *OpGeth) JWTPath() string {
	return n.jwtPath
}

func (n *OpGeth) hydrate(system stack.ExtensibleSystem) {
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

func (n *OpGeth) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.l2Geth != nil {
		n.logger.Warn("op-geth already started")
		return
	}

	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.logger.New("proxy", "l2el-auth"))
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})
		n.authRPC = "ws://" + n.authProxy.Addr()
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.logger.New("proxy", "l2el-user"))
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
	n.authProxy.SetUpstream(ProxyAddr(require, l2Geth.AuthRPC().RPC()))
	n.userProxy.SetUpstream(ProxyAddr(require, l2Geth.UserRPC().RPC()))
}

func (n *OpGeth) Stop() {
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

func WithOpGeth(id stack.L2ELNodeID, opts ...L2ELOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(id.ChainID())
		require.True(ok, "L2 network required")

		cfg := DefaultL2ELConfig()
		orch.l2ELOptions.Apply(p, id, cfg)       // apply global options
		L2ELOptionBundle(opts).Apply(p, id, cfg) // apply specific options

		jwtPath, _ := orch.writeDefaultJWT()

		useInterop := l2Net.genesis.Config.InteropTime != nil

		supervisorRPC := ""
		if useInterop {
			require.NotNil(cfg.SupervisorID, "supervisor is required for interop")
			sup, ok := orch.supervisors.Get(*cfg.SupervisorID)
			require.True(ok, "supervisor is required for interop")
			supervisorRPC = sup.userRPC
		}

		logger := p.Logger()

		l2EL := &OpGeth{
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
