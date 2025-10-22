package supernode

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	opnodecfg "github.com/ethereum-optimism/optimism/op-node/config"
	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/resources"
	gethlog "github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supernode/config"
)

type Supernode struct {
	log          gethlog.Logger
	version      string
	requestStop  context.CancelCauseFunc
	stopped      bool
	cfg          *config.CLIConfig
	chains       map[eth.ChainID]cc.ChainContainer
	wg           sync.WaitGroup
	l1Client     *sources.L1Client
	beaconClient *sources.L1BeaconClient
	httpServer   *httputil.HTTPServer
	rpcRouter    *resources.Router
	// Metrics router/server for per-chain metrics
	metrics       *resources.MetricsService
	metricsRouter *resources.MetricsRouter
	// cached address when available
	rpcAddr string
}

func New(ctx context.Context, log gethlog.Logger, version string, requestStop context.CancelCauseFunc, cfg *config.CLIConfig, vnCfgs map[eth.ChainID]*opnodecfg.Config) (*Supernode, error) {
	s := &Supernode{log: log, version: version, requestStop: requestStop, cfg: cfg, chains: make(map[eth.ChainID]cc.ChainContainer)}

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
		chainID := eth.ChainIDFromUInt64(id)
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
	s.httpServer = httputil.NewHTTPServer(addr, s.rpcRouter)

	// Optionally build metrics service
	if cfg.MetricsConfig.Enabled {
		s.metrics = resources.NewMetricsService(log, cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort, s.metricsRouter)
	}
	return s, nil
}

func (s *Supernode) Start(ctx context.Context) error {
	s.log.Info("supernode starting", "version", s.version)
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
		s.metrics.Start(func(err error) {
			defer s.wg.Done()
			if s.requestStop != nil {
				s.requestStop(err)
			}
		})
	}
	for chainID, chain := range s.chains {
		s.wg.Add(1)
		go func(chainID eth.ChainID, chain cc.ChainContainer) {
			defer s.wg.Done()
			if err := chain.Start(ctx); err != nil {
				s.log.Error("error starting chain", "chain_id", chainID.String(), "error", err)
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
	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(ctx, 0)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
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
			s.log.Error("error stopping chain container", "chain_id", chainID.String(), "error", err)
		}
	}

	s.wg.Wait()

	if s.l1Client != nil {
		s.l1Client.Close()
	}

	return nil
}

func (s *Supernode) Stopped() bool { return s.stopped }

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
