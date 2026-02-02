package sysgo

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/shim"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-interop-filter/filter"
	"github.com/ethereum-optimism/optimism/op-service/client"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/testutils/tcpproxy"
)

type InteropFilter struct {
	mu sync.Mutex

	id      stack.InteropFilterID
	userRPC string

	cfg    *filter.Config
	p      devtest.P
	logger log.Logger

	service *filter.Service

	proxy *tcpproxy.Proxy
}

var _ stack.Lifecycle = (*InteropFilter)(nil)

func (f *InteropFilter) hydrate(sys stack.ExtensibleSystem) {
	tlog := sys.Logger().New("id", f.id)
	filterClient, err := client.NewRPC(sys.T().Ctx(), tlog, f.userRPC, client.WithLazyDial())
	sys.T().Require().NoError(err)
	sys.T().Cleanup(filterClient.Close)

	sys.AddInteropFilter(shim.NewInteropFilter(shim.InteropFilterConfig{
		CommonConfig: shim.NewCommonConfig(sys.T()),
		ID:           f.id,
		Client:       filterClient,
	}))
}

func (f *InteropFilter) UserRPC() string {
	return f.userRPC
}

func (f *InteropFilter) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.service != nil {
		f.logger.Warn("InteropFilter already started")
		return
	}

	if f.proxy == nil {
		f.proxy = tcpproxy.New(f.logger.New("proxy", "interop-filter"))
		f.p.Require().NoError(f.proxy.Start())
		f.p.Cleanup(func() {
			f.proxy.Close()
		})
		f.userRPC = "http://" + f.proxy.Addr()
	}

	svc, err := filter.NewService(context.Background(), f.cfg, f.logger)
	f.p.Require().NoError(err)

	f.service = svc
	f.logger.Info("Starting interop filter")
	err = svc.Start(context.Background())
	f.p.Require().NoError(err, "interop filter failed to start")
	f.logger.Info("Started interop filter")
	f.proxy.SetUpstream(ProxyAddr(f.p.Require(), svc.HTTPEndpoint()))
}

func (f *InteropFilter) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.service == nil {
		f.logger.Warn("InteropFilter already stopped")
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // force-quit
	f.logger.Info("Closing interop filter")
	closeErr := f.service.Stop(ctx)
	f.logger.Info("Closed interop filter", "err", closeErr)

	f.service = nil
}

// WithInteropFilter creates an interop filter service that connects to the specified L2 EL nodes.
// The filter will ingest logs from all specified L2 nodes and provide cross-chain validation.
func WithInteropFilter(filterID stack.InteropFilterID, clusterID stack.ClusterID, l2ELIDs []stack.L2ELNodeID) stack.Option[*Orchestrator] {
	return stack.AfterDeploy(func(orch *Orchestrator) {
		p := orch.P().WithCtx(stack.ContextWithID(orch.P().Ctx(), filterID))
		require := p.Require()

		cluster, ok := orch.clusters.Get(clusterID)
		require.True(ok, "need cluster to determine rollup configs")
		require.NotNil(cluster.cfgset, "need a full config set")
		require.NoError(cluster.cfgset.CheckChains(), "config set must be valid")

		// Collect L2 RPC endpoints
		var l2RPCs []string
		for _, l2ELID := range l2ELIDs {
			l2EL, ok := orch.GetL2EL(l2ELID)
			require.True(ok, "need L2 EL node %s", l2ELID)
			l2RPCs = append(l2RPCs, l2EL.UserRPC())
		}

		cfg := &filter.Config{
			L2RPCs:              l2RPCs,
			DataDir:             orch.p.TempDir(),
			BackfillDuration:    30 * time.Second, // Short for tests
			MessageExpiryWindow: uint64((7 * 24 * time.Hour).Seconds()),
			Version:             "dev",
			LogConfig: oplog.CLIConfig{
				Level:  log.LevelDebug,
				Format: oplog.FormatText,
			},
			MetricsConfig: metrics.CLIConfig{
				Enabled: false,
			},
			PprofConfig: oppprof.CLIConfig{
				ListenEnabled: false,
			},
			RPC: oprpc.CLIConfig{
				ListenAddr:  "127.0.0.1",
				ListenPort:  0, // Allocate dynamically
				EnableAdmin: false,
			},
		}

		plog := p.Logger()
		filterNode := &InteropFilter{
			id:      filterID,
			userRPC: "", // set on start
			cfg:     cfg,
			p:       p,
			logger:  plog,
			service: nil, // set on start
		}
		orch.interopFilters.Set(filterID, filterNode)
		filterNode.Start()
		orch.p.Cleanup(filterNode.Stop)
	})
}
