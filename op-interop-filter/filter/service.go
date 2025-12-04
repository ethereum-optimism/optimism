package filter

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	opservice "github.com/ethereum-optimism/optimism/op-service"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
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

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server

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
		s.log.Info("Metrics disabled")
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
	s.log.Info("Started metrics server", "addr", metricsSrv.Addr())
	s.metricsSrv = metricsSrv
	return nil
}

func (s *Service) initBackend(ctx context.Context, cfg *Config) error {
	backend, err := NewBackend(ctx, s.log, s.metrics, cfg)
	if err != nil {
		return err
	}
	s.backend = backend
	return nil
}

func (s *Service) initRPCServer(cfg *Config) error {
	opts := []oprpc.Option{
		oprpc.WithLogger(s.log),
	}

	// Load JWT secret if path is provided (generates new secret if file is empty)
	if cfg.JWTSecretPath != "" {
		secret, err := oprpc.ObtainJWTSecret(s.log, cfg.JWTSecretPath, true)
		if err != nil {
			return fmt.Errorf("failed to obtain JWT secret: %w", err)
		}
		opts = append(opts, oprpc.WithJWTSecret(secret[:]))
	}

	server := oprpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		s.version,
		opts...,
	)

	// Register supervisor query API
	server.AddAPI(rpc.API{
		Namespace:     "supervisor",
		Service:       &QueryFrontend{backend: s.backend},
		Authenticated: false,
	})

	// Register admin API (opt-in)
	if cfg.RPC.EnableAdmin {
		s.log.Info("Admin RPC enabled")
		server.AddAPI(rpc.API{
			Namespace:     "admin",
			Service:       &AdminFrontend{backend: s.backend},
			Authenticated: true,
		})
	}

	s.rpcServer = server
	return nil
}

// Start starts the service
func (s *Service) Start(ctx context.Context) error {
	s.log.Info("Starting op-interop-filter")

	// Start backend (begins block ingestion)
	if err := s.backend.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}

	// Start RPC server
	if err := s.rpcServer.Start(); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}
	s.log.Info("RPC server started", "endpoint", s.rpcServer.Endpoint())

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
