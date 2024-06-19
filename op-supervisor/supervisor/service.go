package supervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/metrics"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/frontend"
)

type Backend interface {
	frontend.Backend
	io.Closer
}

// SupervisorService implements the full-environment bells and whistles around the Supervisor.
// This includes the setup and teardown of metrics, pprof, admin RPC, regular RPC etc.
type SupervisorService struct {
	closing atomic.Bool

	log log.Logger

	metrics metrics.Metricer

	backend Backend

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server
}

var _ cliapp.Lifecycle = (*SupervisorService)(nil)

func SupervisorFromCLIConfig(ctx context.Context, cfg *CLIConfig, logger log.Logger) (*SupervisorService, error) {
	su := &SupervisorService{log: logger}
	if err := su.initFromCLIConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, su.Stop(ctx)) // try to clean up our failed initialization attempt
	}
	return su, nil
}

func (su *SupervisorService) initFromCLIConfig(ctx context.Context, cfg *CLIConfig) error {
	su.initMetrics(cfg)
	if err := su.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to start PProf server: %w", err)
	}
	if err := su.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to start Metrics server: %w", err)
	}
	su.initBackend(cfg)
	if err := su.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}
	return nil
}

func (su *SupervisorService) initBackend(cfg *CLIConfig) {
	if cfg.MockRun {
		su.backend = backend.NewMockBackend()
	} else {
		su.backend = backend.NewSupervisorBackend()
	}
}

func (su *SupervisorService) initMetrics(cfg *CLIConfig) {
	if cfg.MetricsConfig.Enabled {
		procName := "default"
		su.metrics = metrics.NewMetrics(procName)
		su.metrics.RecordInfo(cfg.Version)
	} else {
		su.metrics = metrics.NoopMetrics
	}
}

func (su *SupervisorService) initPProf(cfg *CLIConfig) error {
	su.pprofService = oppprof.New(
		cfg.PprofConfig.ListenEnabled,
		cfg.PprofConfig.ListenAddr,
		cfg.PprofConfig.ListenPort,
		cfg.PprofConfig.ProfileType,
		cfg.PprofConfig.ProfileDir,
		cfg.PprofConfig.ProfileFilename,
	)

	if err := su.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof service: %w", err)
	}

	return nil
}

func (su *SupervisorService) initMetricsServer(cfg *CLIConfig) error {
	if !cfg.MetricsConfig.Enabled {
		su.log.Info("Metrics disabled")
		return nil
	}
	m, ok := su.metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server", su.metrics)
	}
	su.log.Debug("Starting metrics server", "addr", cfg.MetricsConfig.ListenAddr, "port", cfg.MetricsConfig.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.MetricsConfig.ListenAddr, cfg.MetricsConfig.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	su.log.Info("Started metrics server", "addr", metricsSrv.Addr())
	su.metricsSrv = metricsSrv
	return nil
}

func (su *SupervisorService) initRPCServer(cfg *CLIConfig) error {
	server := oprpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		cfg.Version,
		oprpc.WithLogger(su.log),
		//oprpc.WithHTTPRecorder(su.metrics), // TODO(protocol-quest#286) hook up metrics to RPC server
	)
	if cfg.RPC.EnableAdmin {
		su.log.Info("Admin RPC enabled")
		server.AddAPI(rpc.API{
			Namespace:     "admin",
			Service:       &frontend.AdminFrontend{Supervisor: su.backend},
			Authenticated: true, // TODO(protocol-quest#286): enforce auth on this or not?
		})
	}
	server.AddAPI(rpc.API{
		Namespace:     "supervisor",
		Service:       &frontend.QueryFrontend{Supervisor: su.backend},
		Authenticated: false,
	})
	su.rpcServer = server
	return nil
}

func (su *SupervisorService) Start(ctx context.Context) error {
	su.log.Info("Starting JSON-RPC server")
	if err := su.rpcServer.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}

	su.metrics.RecordUp()
	return nil
}

func (su *SupervisorService) Stop(ctx context.Context) error {
	if !su.closing.CompareAndSwap(false, true) {
		return nil // already closing
	}

	var result error
	if su.rpcServer != nil {
		if err := su.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	if su.backend != nil {
		if err := su.backend.Close(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close supervisor backend: %w", err))
		}
	}
	if su.pprofService != nil {
		if err := su.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
	}
	if su.metricsSrv != nil {
		if err := su.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}
	return result
}

func (su *SupervisorService) Stopped() bool {
	return su.closing.Load()
}
