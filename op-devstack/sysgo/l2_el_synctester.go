package sysgo

import (
	"fmt"
	"sync"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

// SyncTesterEL is an L2ELNode implementation that runs a sync tester service.
// It provides RPC endpoints that can be used by CL nodes for testing sync functionality.
type SyncTesterEL struct {
	mu sync.Mutex

	id      stack.L2ELNodeID
	l2Net   *L2Network
	jwtPath string

	authRPC string
	userRPC string

	authProxy *tcpproxy.Proxy
	userProxy *tcpproxy.Proxy

	config *SyncTesterELConfig
	p      devtest.P

	// Reference to the orchestrator to find the EL node to connect to
	orch *Orchestrator
}

type SyncTesterELConfig struct {
	FCUState     eth.FCUState
	ELSyncActive bool
	ELSyncTarget uint64
}

func (cfg *SyncTesterELConfig) Path() string {
	path := fmt.Sprintf("?latest=%d&safe=%d&finalized=%d", cfg.FCUState.Latest, cfg.FCUState.Safe, cfg.FCUState.Finalized)
	if cfg.ELSyncActive {
		path += fmt.Sprintf("&el_sync_target=%d", cfg.ELSyncTarget)
	}
	return path
}

func DefaultSyncTesterELConfig() *SyncTesterELConfig {
	return &SyncTesterELConfig{
		FCUState:     eth.FCUState{Latest: 0, Safe: 0, Finalized: 0},
		ELSyncActive: false,
		ELSyncTarget: 0,
	}
}

type SyncTesterELOption interface {
	Apply(p devtest.P, id stack.L2ELNodeID, cfg *SyncTesterELConfig)
}

// WithGlobalSyncTesterELOption applies the SyncTesterELOption to all SyncTesterEL instances in this orchestrator
func WithGlobalSyncTesterELOption(opt SyncTesterELOption) stack.Option[*Orchestrator] {
	return stack.BeforeDeploy(func(o *Orchestrator) {
		o.SyncTesterELOptions = append(o.SyncTesterELOptions, opt)
	})
}

type SyncTesterELOptionFn func(p devtest.P, id stack.L2ELNodeID, cfg *SyncTesterELConfig)

var _ SyncTesterELOption = SyncTesterELOptionFn(nil)

func (fn SyncTesterELOptionFn) Apply(p devtest.P, id stack.L2ELNodeID, cfg *SyncTesterELConfig) {
	fn(p, id, cfg)
}

// SyncTesterELOptionBundle a list of multiple SyncTesterELOption, to all be applied in order.
type SyncTesterELOptionBundle []SyncTesterELOption

var _ SyncTesterELOptionBundle = SyncTesterELOptionBundle(nil)

func (l SyncTesterELOptionBundle) Apply(p devtest.P, id stack.L2ELNodeID, cfg *SyncTesterELConfig) {
	for _, opt := range l {
		p.Require().NotNil(opt, "cannot Apply nil SyncTesterELOption")
		opt.Apply(p, id, cfg)
	}
}

var _ L2ELNode = (*SyncTesterEL)(nil)

func (n *SyncTesterEL) hydrate(system stack.ExtensibleSystem) {
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
	sysL2EL.SetLabel(match.LabelVendor, "sync-tester")
	l2Net.(stack.ExtensibleL2Network).AddL2ELNode(sysL2EL)
}

func (n *SyncTesterEL) Start() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// The SyncTesterEL should connect to the existing sync tester service
	// Get the endpoint from the orchestrator's syncTester service
	if n.orch.syncTester == nil || n.orch.syncTester.service == nil {
		n.p.Logger().Error("syncTester service not available in orchestrator")
		return
	}

	// Use NewEndpoint to get the correct session-specific endpoint for this chain ID
	endpoint := n.orch.syncTester.service.SyncTesterRPCPath(n.id.ChainID(), true)

	path := endpoint + n.config.Path()

	if n.authProxy == nil {
		n.authProxy = tcpproxy.New(n.p.Logger().New("proxy", "l2el-synctester-auth"))
		n.p.Require().NoError(n.authProxy.Start())
		n.p.Cleanup(func() {
			n.authProxy.Close()
		})

		rpc := "http://" + n.authProxy.Addr()
		n.authRPC = rpc + path
	}
	if n.userProxy == nil {
		n.userProxy = tcpproxy.New(n.p.Logger().New("proxy", "l2el-synctester-user"))
		n.p.Require().NoError(n.userProxy.Start())
		n.p.Cleanup(func() {
			n.userProxy.Close()
		})

		rpc := "http://" + n.userProxy.Addr()
		n.userRPC = rpc + path
	}

	sessionURL := n.orch.syncTester.service.RPC() + path

	n.authProxy.SetUpstream(ProxyAddr(n.p.Require(), sessionURL))
	n.userProxy.SetUpstream(ProxyAddr(n.p.Require(), sessionURL))
}

func (n *SyncTesterEL) Stop() {
	// The SyncTesterEL is just a proxy, so there's nothing to stop
}

func (n *SyncTesterEL) UserRPC() string {
	return n.userRPC
}

func (n *SyncTesterEL) EngineRPC() string {
	return n.authRPC
}

func (n *SyncTesterEL) JWTPath() string {
	return n.jwtPath
}

// WithSyncTesterL2ELNode creates a SyncTesterEL that satisfies the L2ELNode interface
// The sync tester acts as an EL node that can be used by CL nodes for testing sync.
func WithSyncTesterL2ELNode(id, readonlyEL stack.L2ELNodeID, opts ...SyncTesterELOption) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), id))
		require := p.Require()

		l2Net, ok := orch.l2Nets.Get(readonlyEL.ChainID())
		require.True(ok, "L2 network required")

		cfg := DefaultSyncTesterELConfig()
		orch.SyncTesterELOptions.Apply(p, id, cfg)       // apply global options
		SyncTesterELOptionBundle(opts).Apply(p, id, cfg) // apply specific options

		jwtPath, _ := orch.writeDefaultJWT()

		syncTesterEL := &SyncTesterEL{
			id:      id,
			l2Net:   l2Net,
			jwtPath: jwtPath,
			config:  cfg,
			p:       p,
			orch:    orch,
		}

		p.Logger().Info("Starting sync tester EL", "id", id)
		syncTesterEL.Start()
		p.Cleanup(syncTesterEL.Stop)
		p.Logger().Info("sync tester EL is ready", "userRPC", syncTesterEL.userRPC, "authRPC", syncTesterEL.authRPC)
		require.True(orch.l2ELs.SetIfMissing(id, syncTesterEL), "must be unique L2 EL node")
	})
}
