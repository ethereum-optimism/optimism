package supernode

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/claimfollow"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/heartbeat"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/interop"
	supernodeactivity "github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/supernode"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity/superroot"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	gethlog "github.com/ethereum/go-ethereum/log"
	rpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum-optimism/optimism/op-supernode/config"
)

type Supernode struct {
	log         gethlog.Logger
	version     string
	requestStop context.CancelCauseFunc
	stopped     atomic.Bool
	cfg         *config.CLIConfig
	chains      map[eth.ChainID]cc.InteropChain
	// activitiesMu guards reads and writes of the activities slice.
	activitiesMu    sync.RWMutex
	activities      []activity.Activity
	rootRPC         *oprpc.Handler
	wg              sync.WaitGroup
	lifecycleCancel context.CancelFunc // canceled in Stop() to unblock goroutines from Start()
	l1Client        *sources.L1Client
	beaconClient    *sources.L1BeaconClient
	httpServer      *httputil.HTTPServer
	rpcRouter       *resources.Router
	// Metrics router/server for per-chain metrics
	metrics          *resources.MetricsService
	metricsFanIn     *resources.MetricsFanIn
	supernodeMetrics *resources.SupernodeMetrics
	// cached address when available
	rpcAddr string
}

func New(ctx context.Context, log gethlog.Logger, version string, commit string, requestStop context.CancelCauseFunc, cfg *config.CLIConfig, vnCfgs map[eth.ChainID]*opnodecfg.Config) (*Supernode, error) {
	s := &Supernode{log: log, version: version, requestStop: requestStop, cfg: cfg, chains: make(map[eth.ChainID]cc.InteropChain)}

	// Initialize L1 client
	if err := s.initL1Client(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize L1 client: %w", err)
	}

	// Initialize L1 Beacon client (optional)
	if err := s.initBeaconClient(ctx, cfg); err != nil {
		return nil, fmt.Errorf("failed to initialize L1 Beacon client: %w", err)
	}

	// Initialize chain containers for each configured chain ID
	// Pass shared resources via InitializationOverrides to all containers
	// Build RPC router first; chain containers attach handlers and readiness checks at runtime.
	s.rpcRouter = resources.NewRouter(log, resources.RouterConfig{})
	// Root JSON-RPC handler mounted at '/'
	s.rootRPC = oprpc.NewHandler(version, oprpc.WithLogger(log))
	s.rpcRouter.SetRootHandler(s.rootRPC)
	// Build metrics router; attach per-chain registries later
	s.metricsFanIn = resources.NewMetricsFanIn(len(cfg.Chains))
	s.supernodeMetrics = resources.NewSupernodeMetrics()
	s.supernodeMetrics.Info.WithLabelValues(version, commit).Set(1)
	s.metricsFanIn.AddGatherer(s.supernodeMetrics.Registry())

	// The Private Interop follow module. Dormant unless --private-interop.enabled is set: when it
	// is not, claimFollow stays nil, no route is registered, no activity is added, and every chain
	// container is built exactly as it was before this group existed.
	claimFollow, claimFollowRoutes, err := s.initClaimFollow(cfg, vnCfgs)
	if err != nil {
		return nil, err
	}

	for _, id := range cfg.Chains {
		chainID := eth.ChainIDFromUInt64(id)
		initOverrides := &rollupNode.InitializationOverrides{
			L1Source: resources.NewNonCloseableL1Client(s.l1Client),
			Beacon:   resources.NewNonCloseableL1BeaconClient(s.beaconClient),
		}
		// no rpc handler is passed to the chain container, it will create a new one per (re)start
		if vnCfgs[chainID] == nil {
			log.Error("missing virtual node config for chain", "chain", id)
			continue
		}
		container, err := cc.NewChainContainer(chainID, vnCfgs[chainID], log, *cfg, initOverrides, nil, s.rpcRouter, s.metricsFanIn.SetMetricsRegistry, s.supernodeMetrics, version,
			cc.WithExtraRPCRoutes(claimFollowRoutes[chainID]...))
		if err != nil {
			return nil, fmt.Errorf("failed to create chain container for chain %s: %w", chainID, err)
		}
		s.chains[chainID] = container
	}

	if claimFollow != nil {
		// The module reads the chain the supernode already drives, in process: no dial, no
		// credential, no second view of the same blocks.
		claimFollow.Module().Attach(s.chains[claimFollow.ChainID()])
	}

	// Narrow the chain map to ChainContainer for activities that must not invoke
	// reorg-triggering operations. Only interop gets the wider InteropChain view.
	narrowChains := make(map[eth.ChainID]cc.ChainContainer, len(s.chains))
	for id, c := range s.chains {
		narrowChains[id] = c
	}

	// Resolve interop activation before constructing Superroot so the
	// verified-result reader is available. When interop is not configured,
	// the no-op reader routes every call into the pre-interop fallback.
	interopActivationTimestamp, err := resolveInteropActivationTimestamp(cfg.InteropActivationTimestamp, vnCfgs)
	if err != nil {
		return nil, fmt.Errorf("resolve interop activation timestamp: %w", err)
	}

	log.Info("initializing interop activity", "enabled", interopActivationTimestamp != nil)

	var verifiedReader interop.VerifiedResultReader = interop.NoopVerifiedResultReader{}
	var interopActivity *interop.Interop
	if interopActivationTimestamp != nil {
		var msgExpiryWindow uint64
		for _, vnCfg := range vnCfgs {
			if vnCfg.DependencySet != nil {
				msgExpiryWindow = vnCfg.DependencySet.MessageExpiryWindow()
				break
			}
		}
		interopActivity = interop.New(log.New("activity", "interop"), *interopActivationTimestamp, msgExpiryWindow, s.chains, cfg.DataDir, s.l1Client, cfg.InteropLogBackfillDepth, s.supernodeMetrics)
		verifiedReader = interopActivity
	}

	// Order in this slice governs Start/Stop ordering; interop is appended
	// below so it starts after the RPC servers.
	s.activities = []activity.Activity{
		heartbeat.New(log.New("activity", "heartbeat"), 10*time.Second),
		supernodeactivity.New(log.New("activity", "supernode"), narrowChains),
		superroot.New(log.New("activity", "superroot"), narrowChains, verifiedReader),
	}

	if claimFollow != nil {
		s.activities = append(s.activities, claimFollow)
	}

	if interopActivity != nil {
		s.activities = append(s.activities, interopActivity)
		for _, chain := range s.chains {
			chain.RegisterVerifier(interopActivity)
		}
	}

	// Set up reset callbacks on all chain containers
	// When a chain resets, notify all activities
	for _, chain := range s.chains {
		chain.SetResetCallback(s.onChainReset)
	}

	// Set up http server
	addr := net.JoinHostPort(cfg.RPCConfig.ListenAddr, strconv.Itoa(cfg.RPCConfig.ListenPort))
	s.httpServer = httputil.NewHTTPServer(addr, s.rpcRouter)

	// Optionally build metrics service
	if cfg.MetricsConfig.Enabled {
		s.metrics = resources.NewMetricsService(log, cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort, s.metricsFanIn)
	}
	return s, nil
}

// initClaimFollow builds the Private Interop follow module, if the operator asked for it.
//
// It returns (nil, an empty route map, nil) when the flag group is unset, which is the dormant
// path: no state, no goroutine, no route, and chain containers built with zero options. The
// returned map is keyed by chain ID and is indexed unconditionally by the chain loop — a lookup
// that misses yields a nil slice, and cc.WithExtraRPCRoutes of nothing registers nothing.
//
// The two validations here are the ones only the supernode can make: that the configured chain is
// actually one this process runs, and that its rollup config is loaded. Both would otherwise
// surface as a module that scanned nothing and served an error forever.
func (s *Supernode) initClaimFollow(cfg *config.CLIConfig, vnCfgs map[eth.ChainID]*opnodecfg.Config) (*claimfollow.Activity, map[eth.ChainID][]cc.ExtraRPCRoute, error) {
	routes := map[eth.ChainID][]cc.ExtraRPCRoute{}
	cliCfg := claimfollow.ReadCLIConfig(cfg.RawCtx)
	if err := cliCfg.Check(); err != nil {
		return nil, nil, fmt.Errorf("private interop follow module: %w", err)
	}
	if !cliCfg.Enabled {
		return nil, routes, nil
	}
	chainID, modCfg, route, err := cliCfg.Resolve()
	if err != nil {
		return nil, nil, fmt.Errorf("private interop follow module: %w", err)
	}
	vnCfg := vnCfgs[chainID]
	if vnCfg == nil {
		return nil, nil, fmt.Errorf("private interop follow module: chain %s is not one of this supernode's chains", chainID)
	}

	lgr := s.log.New("activity", "claim-follow", "chain_id", chainID.String())
	metricsReg := prometheus.NewRegistry()
	metrics := claimfollow.NewPromMetrics(opmetrics.With(metricsReg), "supernode")
	s.metricsFanIn.AddGatherer(metricsReg)

	mod := claimfollow.New(modCfg, &vnCfg.Rollup, lgr, metrics)
	routes[chainID] = []cc.ExtraRPCRoute{{
		Route: route,
		// The follow protocol is one method in the standard namespace, served at a sibling route
		// rather than added to the chain's own: see claimfollow.API.
		API: rpc.API{Namespace: "optimism", Service: claimfollow.NewAPI(mod)},
	}}
	act := claimfollow.NewActivity(lgr, chainID, mod, time.Duration(vnCfg.Rollup.BlockTime)*time.Second)
	s.log.Info("private interop follow module enabled",
		"chain_id", chainID.String(), "route", route, "registry", modCfg.Registry)
	return act, routes, nil
}

func resolveInteropActivationTimestamp(override *uint64, vnCfgs map[eth.ChainID]*opnodecfg.Config) (*uint64, error) {
	if override != nil {
		return override, nil
	}

	var resolved *uint64
	var resolvedChain eth.ChainID
	var missingChain *eth.ChainID

	for chainID, vnCfg := range vnCfgs {
		if vnCfg == nil {
			continue
		}

		if vnCfg.Rollup.LagoonTime == nil {
			if resolved != nil {
				return nil, fmt.Errorf("chain %s has no Lagoon activation timestamp, but chain %s is configured for timestamp %d", chainID, resolvedChain, *resolved)
			}
			if missingChain == nil {
				missingChain = new(eth.ChainID)
				*missingChain = chainID
			}
			continue
		}

		if missingChain != nil {
			return nil, fmt.Errorf("chain %s is configured for Lagoon activation timestamp %d, but chain %s has no Lagoon activation timestamp", chainID, *vnCfg.Rollup.LagoonTime, *missingChain)
		}

		if resolved == nil {
			ts := *vnCfg.Rollup.LagoonTime
			resolved = &ts
			resolvedChain = chainID
			continue
		}

		if *resolved != *vnCfg.Rollup.LagoonTime {
			return nil, fmt.Errorf("mismatched Lagoon activation timestamps: chain %s=%d, chain %s=%d", resolvedChain, *resolved, chainID, *vnCfg.Rollup.LagoonTime)
		}
	}

	return resolved, nil
}

func (s *Supernode) Start(ctx context.Context) error {
	s.log.Info("supernode starting", "version", s.version)

	// Create a lifecycle context that is canceled in Stop(). This ensures that
	// goroutines spawned below will exit even if Stop() wins the race and runs
	// before the goroutine has been scheduled. Without this, an activity's
	// Start(ctx) could block forever if its Stop() was already called (and
	// found cancel == nil) before Start() had a chance to initialize.
	var lifecycleCtx context.Context
	lifecycleCtx, s.lifecycleCancel = context.WithCancel(ctx)

	if s.httpServer != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := s.httpServer.Start(); err != nil {
				s.log.Error("rpc server error", "error", err)
				if s.requestStop != nil {
					s.requestStop(err)
				}
				return
			}
			// cache bound address for quick reads
			if addr := s.httpServer.Addr(); addr != nil {
				s.rpcAddr = addr.String()
				s.log.Info("starting RPC router server", "addr", s.rpcAddr)
			}
		}()
	}
	// Start metrics service
	if s.metrics != nil {
		s.wg.Add(1)
		s.metrics.Start(s.wg.Done, func(err error) {
			if s.requestStop != nil {
				s.requestStop(err)
			}
		})
	}
	// Register RPC APIs and start only Runnable activities
	for _, a := range s.activities {
		if ra, ok := a.(activity.RPCActivity); ok {
			if err := s.rootRPC.AddAPI(rpc.API{Namespace: ra.RPCNamespace(), Service: ra.RPCService()}); err != nil {
				s.log.Error("failed to register activity RPC API", "namespace", ra.RPCNamespace(), "error", err)
			}
		}
		if run, ok := a.(activity.RunnableActivity); ok {
			s.wg.Add(1)
			go func(run activity.RunnableActivity) {
				defer s.wg.Done()
				err := run.Start(lifecycleCtx)
				activityName := a.Name()
				switch err {
				case nil:
					s.log.Error("activity quit unexpectedly", "name", activityName)
				case context.Canceled:
					// This is the happy path, normal / clean shutdown
					s.log.Info("activity closing due to cancelled context", "name", activityName)
				case context.DeadlineExceeded:
					s.log.Warn("activity quit due to deadline exceeded", "name", activityName)
				default:
					s.log.Error("error starting runnable activity", "name", activityName, "error", err)
				}
			}(run)
		}
	}
	for chainID, chain := range s.chains {
		s.wg.Add(1)
		go func(chainID eth.ChainID, chain cc.InteropChain) {
			defer s.wg.Done()
			if err := chain.Start(lifecycleCtx); err != nil {
				s.log.Error("error starting chain", "chain_id", chainID.String(), "error", err)
			}
		}(chainID, chain)
	}
	return nil
}

func (s *Supernode) Stop(ctx context.Context) error {
	s.log.Info("supernode stopping")
	s.stopped.Store(false)

	// Cancel the lifecycle context before anything else. This guarantees that
	// activity and chain goroutines will observe a canceled context even if
	// they haven't been scheduled yet when the individual Stop() calls below
	// execute. The individual Stop() calls are still made for orderly cleanup,
	// but the lifecycle cancellation is the backstop that prevents hangs.
	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
	}

	// Stop RPC server first, then close router resources
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 0)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.log.Error("error shutting down rpc server", "error", err)
		} else {
			s.log.Info("rpc server stopped")
		}
	}
	if s.metrics != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 0)
		defer cancel()
		if err := s.metrics.Stop(shutdownCtx); err != nil {
			s.log.Error("error shutting down metrics server", "error", err)
		} else {
			s.log.Info("metrics server stopped")
		}
	}
	if s.rpcRouter != nil {
		if err := s.rpcRouter.Close(); err != nil {
			s.log.Error("error closing rpc router", "error", err)
		} else {
			s.log.Info("rpc router closed")
		}
	}

	s.activitiesMu.RLock()
	activities := append([]activity.Activity(nil), s.activities...)
	s.activitiesMu.RUnlock()
	for _, a := range activities {
		activityName := a.Name()
		if run, ok := a.(activity.RunnableActivity); ok {
			if err := run.Stop(ctx); err != nil {
				s.log.Error("error stopping runnable activity", "name", activityName, "error", err)
			} else {
				s.log.Info("runnable activity stopped", "name", activityName)
			}
		}
	}

	for chainID, chain := range s.chains {
		if err := chain.Stop(ctx); err != nil {
			s.log.Error("error stopping chain container", "chain_id", chainID.String(), "error", err)
		} else {
			s.log.Info("chain container stopped", "chain_id", chainID.String())
		}
	}

	s.log.Info("all chain containers stopped, waiting for goroutines to finish")
	wgDone := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(wgDone)
	}()
	select {
	case <-wgDone:
		s.log.Info("goroutines finished, closing l1 client")
		s.stopped.Store(true)
	case <-ctx.Done():
		s.log.Error("context canceled while waiting for chain goroutines to finish, proceeding with cleanup", "err", ctx.Err())
	case <-time.After(60 * time.Second):
		s.log.Error("timed out waiting for chain goroutines to finish after 60s, proceeding with cleanup")
	}

	if s.l1Client != nil {
		s.l1Client.Close()
	}
	s.log.Info("l1 client closed, supernode stopped")

	return nil
}

// onChainReset is called when a chain container resets due to an invalidated block.
// It notifies all activities about the reset so they can clean up cached state.
func (s *Supernode) onChainReset(chainID eth.ChainID, timestamp uint64, invalidatedBlock eth.BlockRef) {
	s.log.Info("chain reset detected, notifying activities",
		"chainID", chainID,
		"timestamp", timestamp,
		"invalidatedBlock", invalidatedBlock,
	)
	s.activitiesMu.RLock()
	activities := append([]activity.Activity(nil), s.activities...)
	s.activitiesMu.RUnlock()
	for _, a := range activities {
		a.Reset(chainID, timestamp, invalidatedBlock)
	}
}

func (s *Supernode) Stopped() bool { return s.stopped.Load() }

// RPCAddr returns the bound RPC address (host:port) if the server is listening.
// ok is false if the listener has not been created yet.
func (s *Supernode) RPCAddr() (addr string, ok bool) {
	if s.httpServer == nil || s.httpServer.Addr() == nil {
		return "", false
	}
	return s.httpServer.Addr().String(), true
}

// WaitRPCAddr blocks until the RPC server has a bound address or the context is done.
func (s *Supernode) WaitRPCAddr(ctx context.Context) (string, error) {
	// Fast-path
	if addr, ok := s.RPCAddr(); ok {
		return addr, nil
	}
	// Poll until listener is set or context done
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if addr, ok := s.RPCAddr(); ok {
				return addr, nil
			}
		}
	}
}

// L1Client returns the L1 client instance
func (s *Supernode) L1Client() *sources.L1Client {
	return s.l1Client
}

// BeaconClient returns the L1 Beacon client instance (may be nil if not configured)
func (s *Supernode) BeaconClient() *sources.L1BeaconClient {
	return s.beaconClient
}

func (s *Supernode) initL1Client(ctx context.Context, cfg *config.CLIConfig) error {
	s.log.Info("initializing shared L1 client", "l1_addr", cfg.L1NodeAddr)

	// Create L1 RPC client with basic configuration
	// Enable HTTP polling for L1 heads to support HTTP-only L1 connections (e.g., in tests)
	s.log.Info("configuring shared L1 HTTP poll interval", "interval", cfg.L1HTTPPollInterval)
	l1RPC, err := client.NewRPC(ctx, s.log, cfg.L1NodeAddr,
		client.WithDialAttempts(10),
		client.WithHttpPollInterval(cfg.L1HTTPPollInterval),
	)
	if err != nil {
		return fmt.Errorf("failed to dial L1 address (%s): %w", cfg.L1NodeAddr, err)
	}

	nonCloseableRPC := resources.NewNonCloseableRPC(l1RPC)

	l1ClientCfg := sources.L1ClientSimpleConfig(false, sources.RPCKindStandard, 100)
	s.l1Client, err = sources.NewL1Client(nonCloseableRPC, s.log, nil, l1ClientCfg)
	if err != nil {
		return fmt.Errorf("failed to create L1 client: %w", err)
	}

	s.log.Info("L1 client initialized successfully")
	return nil
}

func (s *Supernode) initBeaconClient(ctx context.Context, cfg *config.CLIConfig) error {
	if cfg.L1BeaconAddr == "" {
		s.log.Info("L1 Beacon address not configured, skipping beacon client initialization")
		return nil
	}

	s.log.Info("initializing L1 Beacon client", "beacon_addr", cfg.L1BeaconAddr)

	// Create beacon client
	basicClient := client.NewBasicHTTPClient(cfg.L1BeaconAddr, s.log)
	beaconHTTPClient := sources.NewBeaconHTTPClient(basicClient)

	// Create fallback beacon clients (e.g. blob archiver)
	var fallbacks []apis.BeaconClient
	for _, addr := range cfg.L1BeaconFallbackAddrs {
		fb := client.NewBasicHTTPClient(addr, s.log)
		fallbacks = append(fallbacks, sources.NewBeaconHTTPClient(fb))
	}

	// Create L1 Beacon client with default config
	beaconCfg := sources.L1BeaconClientConfig{
		FetchAllSidecars: false,
	}
	s.beaconClient = sources.NewL1BeaconClient(beaconHTTPClient, beaconCfg, fallbacks...)

	s.log.Info("L1 Beacon client initialized successfully")
	return nil
}
