package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync/atomic"

	"github.com/ethereum-optimism/optimism/op-node/config"
	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-node/node/runcfg"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum/go-ethereum/log"
)

var errInvalidMetricer = errors.New("invalid metricer")

type Backend interface {
	P2P() p2p.Node
	InteropRPC() (rpcEndpoint string, jwtSecret eth.Bytes32)
	InteropRPCPort() (int, error)
	RuntimeConfig() runcfg.ReadonlyRuntimeConfig

	//DriverAPI() node.DriverClient
	//RollupAPI() engine.RollupAPI
	//PublishAPI() apis.PublishAPI

	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Service implements the full-environment bells and whistles around the op-node.
// This includes the setup and teardown of metrics, pprof, admin RPC, regular RPC etc.
type Service struct {
	closing atomic.Bool

	log log.Logger

	metrics metrics.Metricer

	backend Backend

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcHandler   *oprpc.Handler
	rpcServer    *httputil.HTTPServer
}

var _ cliapp.Lifecycle = (*Service)(nil)

func LifecycleFromConfig(ctx context.Context, cfg *config.Config, logger log.Logger) (cliapp.Lifecycle, error) {
	return FromConfig(ctx, cfg, logger)
}

func FromConfig(ctx context.Context, cfg *config.Config, logger log.Logger) (*Service, error) {
	su := &Service{log: logger}
	if err := su.initFromCLIConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, su.Stop(ctx)) // try to clean up our failed initialization attempt
	}
	return su, nil
}

func (su *Service) initFromCLIConfig(ctx context.Context, cfg *config.Config) error {
	su.initMetrics(cfg)
	if err := su.initPProf(cfg); err != nil {
		return fmt.Errorf("failed to start PProf server: %w", err)
	}
	if err := su.initMetricsServer(cfg); err != nil {
		return fmt.Errorf("failed to start Metrics server: %w", err)
	}
	su.initRPCRouter(cfg)
	if err := su.initBackend(ctx, cfg); err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}
	su.initRPCServer(cfg)
	return nil
}

func (su *Service) initBackend(ctx context.Context, cfg *config.Config) error {
	n, err := node.New(ctx, cfg, su.log, cfg.Version, su.metrics, su.rpcHandler)
	if err != nil {
		return fmt.Errorf("failed to create op-node backend: %w", err)
	}
	su.backend = n
	return nil
}

func (su *Service) initMetrics(cfg *config.Config) {
	if cfg.Metrics.Enabled {
		su.metrics = metrics.NewMetrics("")
		su.metrics.RecordInfo(cfg.Version)
	} else {
		su.metrics = metrics.NoopMetrics
	}
}

func (su *Service) initPProf(cfg *config.Config) error {
	su.pprofService = oppprof.New(
		cfg.Pprof.ListenEnabled,
		cfg.Pprof.ListenAddr,
		cfg.Pprof.ListenPort,
		cfg.Pprof.ProfileType,
		cfg.Pprof.ProfileDir,
		cfg.Pprof.ProfileFilename,
	)

	if err := su.pprofService.Start(); err != nil {
		return fmt.Errorf("failed to start pprof service: %w", err)
	}

	return nil
}

func (su *Service) initMetricsServer(cfg *config.Config) error {
	if !cfg.Metrics.Enabled {
		su.log.Info("Metrics disabled")
		return nil
	}
	m, ok := su.metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server: %w", su.metrics, errInvalidMetricer)
	}
	su.log.Debug("Starting metrics server", "addr", cfg.Metrics.ListenAddr, "port", cfg.Metrics.ListenPort)
	metricsSrv, err := opmetrics.StartServer(m.Registry(), cfg.Metrics.ListenAddr, cfg.Metrics.ListenPort)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	su.log.Info("Started metrics server", "addr", metricsSrv.Addr())
	su.metricsSrv = metricsSrv
	return nil
}

func (su *Service) initRPCRouter(cfg *config.Config) {
	su.rpcHandler = oprpc.NewHandler(cfg.Version,
		oprpc.WithLogger(su.log),
		// CORS is not important on op-node, but we used to do this on the old op-node RPC server, so kept for compatibility.
		oprpc.WithCORSHosts([]string{"*"}),
		oprpc.WithRPCRecorder(su.metrics.NewRecorder("main")))
}

func (su *Service) initRPCServer(cfg *config.Config) {
	endpoint := net.JoinHostPort(cfg.RPC.ListenAddr, strconv.Itoa(cfg.RPC.ListenPort))
	su.rpcServer = httputil.NewHTTPServer(endpoint, su.rpcHandler)
}

func (su *Service) Start(ctx context.Context) error {
	su.log.Info("Starting JSON-RPC server")
	if err := su.backend.Start(ctx); err != nil {
		return fmt.Errorf("unable to start backend: %w", err)
	}

	if err := su.rpcServer.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}

	su.metrics.RecordUp()
	su.log.Info("JSON-RPC Server started", "endpoint", su.rpcServer.HTTPEndpoint())
	return nil
}

func (su *Service) Stop(ctx context.Context) error {
	if !su.closing.CompareAndSwap(false, true) {
		su.log.Warn("op-node is already closing")
		return nil // already closing
	}
	su.log.Info("Stopping op-node service")
	var result error
	if su.rpcServer != nil {
		if err := su.rpcServer.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
		su.log.Info("Stopped JSON-RPC server")
	}
	if su.backend != nil {
		if err := su.backend.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close op-node backend: %w", err))
		}
		su.log.Info("Stopped Backend")
	}
	if su.pprofService != nil {
		if err := su.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
		su.log.Info("Stopped PProf")
	}
	if su.metricsSrv != nil {
		if err := su.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
		su.log.Info("Stopped Metrics")
	}
	su.log.Info("Stopped op-node service")
	return result
}

func (su *Service) Stopped() bool {
	return su.closing.Load()
}

func (su *Service) HttpRPC() string {
	return su.rpcServer.HTTPEndpoint()
}

func (su *Service) Port() (int, error) {
	return su.rpcServer.Port()
}

func (su *Service) Backend() Backend {
	return su.backend
}
