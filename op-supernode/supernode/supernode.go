package supernode

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/types"
	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supernode/config"
)

type Supernode struct {
	log          gethlog.Logger
	version      string
	requestStop  context.CancelCauseFunc
	stopped      bool
	cfg          *config.CLIConfig
	chains       map[types.ChainID]cc.ChainContainer
	wg           sync.WaitGroup
	l1Client     *sources.L1Client
	beaconClient *sources.L1BeaconClient
	rpcServer    *http.Server
	rpcRouter    *resources.Router
	// Metrics router/server for per-chain metrics
	metrics       *resources.MetricsService
	metricsRouter *resources.MetricsRouter
}

func New(ctx context.Context, log gethlog.Logger, version string, requestStop context.CancelCauseFunc, cfg *config.CLIConfig, vnCfgs map[types.ChainID]*opnodecfg.Config) (*Supernode, error) {
	s := &Supernode{log: log, version: version, requestStop: requestStop, cfg: cfg, chains: make(map[types.ChainID]cc.ChainContainer)}

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
	// Build RPC router first; we'll attach per-chain handlers at runtime via SetHandler
	s.rpcRouter = resources.NewRouter(log, resources.RouterConfig{})
	// Build metrics router; attach per-chain registries later
	s.metricsRouter = resources.NewMetricsRouter(log)
	for _, id := range cfg.Chains {
		chainID := types.ChainID(id)
		initOverrides := &rollupNode.InitializationOverrides{
			L1Source: resources.NewNonCloseableL1Client(s.l1Client),
			Beacon:   resources.NewNonCloseableL1BeaconClient(s.beaconClient),
		}
		// no rpc handler is passed to the chain container, it will create a new one per (re)start using rpcRouter.SetHandler
		if vnCfgs[chainID] == nil {
			log.Error("missing virtual node config for chain", "chain", id)
			continue
		}
		s.chains[chainID] = cc.NewChainContainer(chainID, vnCfgs[chainID], log, *cfg, initOverrides, nil, s.rpcRouter.SetHandler, s.metricsRouter.SetHandler)
	}
	addr := net.JoinHostPort(cfg.RPCConfig.ListenAddr, strconv.Itoa(cfg.RPCConfig.ListenPort))
	s.rpcServer = &http.Server{Addr: addr, Handler: s.rpcRouter}

	// Optionally build metrics service
	if cfg.MetricsConfig.Enabled {
		s.metrics = resources.NewMetricsService(log, cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort, s.metricsRouter)
	}
	return s, nil
}

func (s *Supernode) Start(ctx context.Context) error {
	s.log.Info("supernode starting", "version", s.version)
	// Start RPC server
	if s.rpcServer != nil {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.log.Info("starting RPC router server", "addr", s.rpcServer.Addr)
			if err := s.rpcServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.log.Error("rpc server error", "error", err)
				if s.requestStop != nil {
					s.requestStop(err)
				}
			}
		}()
	}
	// Start metrics service
	if s.metrics != nil {
		s.wg.Add(1)
		s.metrics.Start(func(err error) {
			defer s.wg.Done()
			if s.requestStop != nil {
				s.requestStop(err)
			}
		})
	}
	for chainID, chain := range s.chains {
		s.wg.Add(1)
		go func(chainID types.ChainID, chain cc.ChainContainer) {
			defer s.wg.Done()
			if err := chain.Start(ctx); err != nil {
				s.log.Error("error starting chain", "chain_id", chainID, "error", err)
			}
		}(chainID, chain)
	}
	<-ctx.Done()
	s.log.Info("supernode received stop signal")
	return ctx.Err()
}

func (s *Supernode) Stop(ctx context.Context) error {
	s.log.Info("supernode stopping")
	s.stopped = true

	// Stop RPC server first, then close router resources
	if s.rpcServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 0)
		defer cancel()
		if err := s.rpcServer.Shutdown(shutdownCtx); err != nil {
			s.log.Error("error shutting down rpc server", "error", err)
		}
	}
	if s.metrics != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 0)
		defer cancel()
		if err := s.metrics.Stop(shutdownCtx); err != nil {
			s.log.Error("error shutting down metrics server", "error", err)
		}
	}
	if s.rpcRouter != nil {
		if err := s.rpcRouter.Close(); err != nil {
			s.log.Error("error closing rpc router", "error", err)
		}
	}
	if s.metricsRouter != nil {
		if err := s.metricsRouter.Close(); err != nil {
			s.log.Error("error closing metrics router", "error", err)
		}
	}

	for chainID, chain := range s.chains {
		if err := chain.Stop(ctx); err != nil {
			s.log.Error("error stopping chain container", "chain_id", chainID, "error", err)
		}
	}

	s.wg.Wait()

	if s.l1Client != nil {
		s.l1Client.Close()
	}

	return nil
}

func (s *Supernode) Stopped() bool { return s.stopped }

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
	l1RPC, err := client.NewRPC(ctx, s.log, cfg.L1NodeAddr, client.WithDialAttempts(10))
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

	// Create L1 Beacon client with default config
	beaconCfg := sources.L1BeaconClientConfig{
		FetchAllSidecars: false,
	}
	s.beaconClient = sources.NewL1BeaconClient(beaconHTTPClient, beaconCfg)

	s.log.Info("L1 Beacon client initialized successfully")
	return nil
}
