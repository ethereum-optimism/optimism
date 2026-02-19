package filter

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"

	"github.com/ethereum-optimism/optimism/op-interop-filter/flags"
	"github.com/ethereum-optimism/optimism/op-interop-filter/metrics"
)

// Service is the main op-interop-filter service
type Service struct {
	log     log.Logger
	metrics metrics.Metricer
	version string

	pprofService   *oppprof.Service
	metricsSrv     *httputil.HTTPServer
	rpcServer      *oprpc.Server // Main RPC server (public supervisor API)
	adminRPCServer *oprpc.Server // Admin RPC server (JWT-protected, separate port)

	backend *Backend

	stopped atomic.Bool
}

var _ cliapp.Lifecycle = (*Service)(nil)

// Main returns the main entrypoint for the service
func Main(version string) cliapp.LifecycleAction {
	return func(cliCtx *cli.Context, closeApp context.CancelCauseFunc) (cliapp.Lifecycle, error) {
		if err := flags.CheckRequired(cliCtx); err != nil {
			return nil, err
		}

		cfg, err := NewConfig(cliCtx, version)
		if err != nil {
			return nil, fmt.Errorf("failed to parse config: %w", err)
		}
		if err := cfg.Check(); err != nil {
			return nil, fmt.Errorf("invalid config: %w", err)
		}

		l := oplog.NewLogger(oplog.AppOut(cliCtx), cfg.LogConfig)
		oplog.SetGlobalLogHandler(l.Handler())
		opservice.ValidateEnvVars(flags.EnvVarPrefix, flags.Flags, l)

		l.Info("Initializing op-interop-filter", "version", version)

		if !cfg.MessageExpiryWindowExplicit {
			l.Debug("Using default message expiry window", "window", DefaultMessageExpiryWindow)
		} else {
			l.Debug("Message expiry window configured", "window", time.Duration(cfg.MessageExpiryWindow)*time.Second)
		}

		return NewService(cliCtx.Context, cfg, l)
	}
}

// NewService creates a new Service instance
func NewService(ctx context.Context, cfg *Config, logger log.Logger) (*Service, error) {
	s := &Service{
		log:     logger,
		version: cfg.Version,
	}
	if err := s.init(ctx, cfg); err != nil {
		return nil, errors.Join(err, s.Stop(ctx))
	}
	return s, nil
}

func (s *Service) init(ctx context.Context, cfg *Config) error {
	s.initMetrics(cfg)

	if err := s.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to init pprof: %w", err)
	}
	if err := s.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to init metrics server: %w", err)
	}
	if err := s.initBackend(ctx, cfg); err != nil {
		return fmt.Errorf("failed to init backend: %w", err)
	}
	if err := s.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to init RPC server: %w", err)
	}
	if err := s.initAdminRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to init admin RPC server: %w", err)
	}
	return nil
}

func (s *Service) initMetrics(cfg *Config) {
	if cfg.MetricsConfig.Enabled {
		s.metrics = metrics.NewMetrics("default")
		s.metrics.RecordInfo(s.version)
	} else {
		s.metrics = metrics.NoopMetrics
	}
}

func (s *Service) initPProf(cfg *Config) error {
	s.pprofService = oppprof.New(
		cfg.PprofConfig.ListenEnabled,
		cfg.PprofConfig.ListenAddr,
		cfg.PprofConfig.ListenPort,
		cfg.PprofConfig.ProfileType,
		cfg.PprofConfig.ProfileDir,
		cfg.PprofConfig.ProfileFilename,
	)
	if err := s.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof: %w", err)
	}
	return nil
}

func (s *Service) initMetricsServer(cfg *Config) error {
	if !cfg.MetricsConfig.Enabled {
		s.log.Debug("Metrics disabled")
		return nil
	}
	m, ok := s.metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics do not expose registry")
	}
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	s.log.Debug("Started metrics server", "addr", metricsSrv.Addr())
	s.metricsSrv = metricsSrv
	return nil
}

func (s *Service) initBackend(ctx context.Context, cfg *Config) error {
	// Calculate start timestamp once for all components.
	// Chain ingesters will start ingesting from (startTimestamp - backfillDuration)
	// and report Ready() once they reach startTimestamp.
	// Cross-validator initializes to startTimestamp and waits for chains to catch up.
	startTimestamp := uint64(clock.SystemClock.Now().Unix())

	chains := make(map[eth.ChainID]ChainIngester)

	// Create chain ingesters for each L2 RPC
	for _, rpcURL := range cfg.L2RPCs {
		// Query chain ID from the RPC
		ethClient, err := ethclient.Dial(rpcURL)
		if err != nil {
			return fmt.Errorf("failed to connect to %s: %w", rpcURL, err)
		}
		chainIDBig, err := ethClient.ChainID(ctx)
		ethClient.Close()
		if err != nil {
			return fmt.Errorf("failed to query chain ID from %s: %w", rpcURL, err)
		}
		chainID := eth.ChainIDFromBig(chainIDBig)

		// Look up rollup config for this chain ID
		rollupCfg, ok := cfg.RollupConfigs[chainID]
		if !ok {
			return fmt.Errorf("no rollup config found for chain %s from RPC %s (use --networks or --rollup-configs)", chainID, rpcURL)
		}

		if _, exists := chains[chainID]; exists {
			return fmt.Errorf("duplicate chain ID %s: multiple RPCs return the same chain ID", chainID)
		}

		s.log.Info("Creating chain ingester", "chain", chainID, "rpc", rpcURL)

		ingester, err := NewLogsDBChainIngester(
			ctx,
			s.log,
			s.metrics,
			chainID,
			rpcURL,
			cfg.DataDir,
			startTimestamp,
			cfg.BackfillDuration,
			cfg.PollInterval,
			rollupCfg,
		)
		if err != nil {
			return fmt.Errorf("failed to create chain ingester for chain %s: %w", chainID, err)
		}

		chains[chainID] = ingester
	}

	crossValidator := NewLockstepCrossValidator(
		ctx,
		s.log,
		s.metrics,
		cfg.MessageExpiryWindow,
		startTimestamp,
		cfg.ValidationInterval,
		chains,
	)

	s.backend = NewBackend(ctx, BackendParams{
		Logger:         s.log,
		Metrics:        s.metrics,
		Chains:         chains,
		CrossValidator: crossValidator,
	})

	s.log.Info("Created backend", "chains", len(chains))
	return nil
}

func (s *Service) initRPCServer(cfg *Config) error {
	// Create server without JWT - public supervisor API
	server := oprpc.NewServer(
		cfg.RPCAddr,
		cfg.RPCPort,
		s.version,
		oprpc.WithLogger(s.log),
	)

	// Register supervisor query API (public, no auth)
	server.AddAPI(rpc.API{
		Namespace:     "supervisor",
		Service:       &QueryFrontend{backend: s.backend},
		Authenticated: false,
	})

	s.rpcServer = server
	return nil
}

func (s *Service) initAdminRPCServer(cfg *Config) error {
	// Admin RPC is disabled if no address is configured
	if cfg.AdminRPCAddr == "" {
		s.log.Debug("Admin RPC disabled (no admin.rpc.addr configured)")
		return nil
	}

	// Load JWT secret for authentication
	secret, err := oprpc.ObtainJWTSecret(s.log, cfg.JWTSecretPath, true)
	if err != nil {
		return fmt.Errorf("failed to obtain JWT secret: %w", err)
	}

	// Create admin server with server-wide JWT authentication
	server := oprpc.NewServer(
		cfg.AdminRPCAddr,
		cfg.AdminRPCPort,
		s.version,
		oprpc.WithLogger(s.log),
		oprpc.WithJWTSecret(secret[:]),
	)

	// Register admin API (JWT-protected server-wide)
	server.AddAPI(rpc.API{
		Namespace:     "admin",
		Service:       &AdminFrontend{backend: s.backend},
		Authenticated: true,
	})

	s.adminRPCServer = server
	s.log.Info("Admin RPC configured", "addr", cfg.AdminRPCAddr, "port", cfg.AdminRPCPort)
	return nil
}

// Start starts the service
func (s *Service) Start(ctx context.Context) error {
	s.log.Info("Starting op-interop-filter")

	// Start backend (begins block ingestion)
	if err := s.backend.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}

	// Start main RPC server (supervisor API)
	if err := s.rpcServer.Start(); err != nil {
		// Rollback: stop backend if RPC server fails to start
		stopErr := s.backend.Stop(ctx)
		return errors.Join(fmt.Errorf("failed to start RPC server: %w", err), stopErr)
	}
	s.log.Info("RPC server started", "endpoint", s.rpcServer.Endpoint())

	// Start admin RPC server if configured
	if s.adminRPCServer != nil {
		if err := s.adminRPCServer.Start(); err != nil {
			// Rollback: stop main RPC and backend if admin RPC server fails
			rpcErr := s.rpcServer.Stop()
			backendErr := s.backend.Stop(ctx)
			return errors.Join(fmt.Errorf("failed to start admin RPC server: %w", err), rpcErr, backendErr)
		}
		s.log.Info("Admin RPC server started", "endpoint", s.adminRPCServer.Endpoint())
	}

	s.metrics.RecordUp()
	return nil
}

// Stop stops the service
func (s *Service) Stop(ctx context.Context) error {
	if !s.stopped.CompareAndSwap(false, true) {
		return nil
	}
	s.log.Info("Stopping op-interop-filter")

	var result error
	if s.adminRPCServer != nil {
		if err := s.adminRPCServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop admin RPC: %w", err))
		}
	}
	if s.rpcServer != nil {
		if err := s.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC: %w", err))
		}
	}
	if s.backend != nil {
		if err := s.backend.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop backend: %w", err))
		}
	}
	if s.pprofService != nil {
		if err := s.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop pprof: %w", err))
		}
	}
	if s.metricsSrv != nil {
		if err := s.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics: %w", err))
		}
	}
	return result
}

// Stopped returns true if the service has been stopped
func (s *Service) Stopped() bool {
	return s.stopped.Load()
}

// HTTPEndpoint returns the HTTP endpoint of the RPC server, or empty string if not started.
func (s *Service) HTTPEndpoint() string {
	if s.rpcServer == nil {
		return ""
	}
	// Include http:// prefix as expected by ProxyAddr
	return "http://" + s.rpcServer.Endpoint()
}

// AdminHTTPEndpoint returns the HTTP endpoint of the admin RPC server, or empty string if not configured.
func (s *Service) AdminHTTPEndpoint() string {
	if s.adminRPCServer == nil {
		return ""
	}
	return "http://" + s.adminRPCServer.Endpoint()
}
