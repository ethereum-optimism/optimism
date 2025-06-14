package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/config"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/metrics"
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend"
	frontend2 "github.com/ethereum-optimism/optimism/op-node/opnv2/service/frontend"
	"github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/oppprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/depset"
)

var errInvalidMetricer = errors.New("invalid metricer")

type Backend interface {
	apis.SupervisorQueryAPI

	DependencySet() depset.DependencySet
	P2P() p2p.Node

	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Service implements the full-environment bells and whistles around the op-node-v2.
// This includes the setup and teardown of metrics, pprof, admin RPC, regular RPC etc.
type Service struct {
	closing atomic.Bool

	log log.Logger

	metrics metrics.Metricer

	backend Backend

	pprofService *oppprof.Service
	metricsSrv   *httputil.HTTPServer
	rpcServer    *oprpc.Server
}

var _ cliapp.Lifecycle = (*Service)(nil)

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
	if err := su.initBackend(ctx, cfg); err != nil {
		return fmt.Errorf("failed to start backend: %w", err)
	}
	if err := su.initRPCServer(cfg); err != nil {
		return fmt.Errorf("failed to start RPC server: %w", err)
	}
	return nil
}

func (su *Service) initBackend(ctx context.Context, cfg *config.Config) error {
	be, err := backend.NewBackend(ctx, su.log, su.metrics, cfg)
	if err != nil {
		return fmt.Errorf("failed to create op-node backend: %w", err)
	}
	su.backend = be
	return nil
}

func (su *Service) initMetrics(cfg *config.Config) {
	if cfg.MetricsConfig.Enabled {
		su.metrics = metrics.NewMetrics()
		su.metrics.RecordInfo(cfg.Version)
	} else {
		su.metrics = metrics.NoopMetrics{}
	}
}

func (su *Service) initPProf(cfg *config.Config) error {
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

func (su *Service) initMetricsServer(cfg *config.Config) error {
	if !cfg.MetricsConfig.Enabled {
		su.log.Info("Metrics disabled")
		return nil
	}
	m, ok := su.metrics.(opmetrics.RegistryMetricer)
	if !ok {
		return fmt.Errorf("metrics were enabled, but metricer %T does not expose registry for metrics-server: %w", su.metrics, errInvalidMetricer)
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

func (su *Service) initRPCServer(cfg *config.Config) error {
	server := oprpc.NewServer(
		cfg.RPC.ListenAddr,
		cfg.RPC.ListenPort,
		cfg.Version,
		oprpc.WithLogger(su.log),
		oprpc.WithRPCRecorder(su.metrics.NewRecorder("main")),
	)
	if err := RegisterRPCs(su.log, cfg, server.Handler, su.backend); err != nil {
		return fmt.Errorf("failed to setup RPC routes: %w", err)
	}
	su.rpcServer = server
	return nil
}

type RpcRouter interface {
	AddAPI(api rpc.API) error
	AddRPC(route string) error
	AddAPIToRPC(route string, api rpc.API) error
}

func RegisterRPCs(logger log.Logger, cfg *config.Config, router RpcRouter, backend Backend) error {
	depSet := backend.DependencySet()

	// If there is a single chain dependency-set,
	// then make those RPCs available at the root route.
	// This is backwards compatible with op-node v1 RPC routes.
	var err error
	if len(depSet.Chains()) <= 1 {
		if cfg.RPC.EnableAdmin {
			err = errors.Join(err, router.AddAPI(rpc.API{
				Namespace: "admin",
				Service:   &frontend2.OpnodeAdminFrontend{},
			}))
		}
		err = errors.Join(err, router.AddAPI(rpc.API{
			Namespace: "optimism",
			Service:   &frontend2.OptimismFrontend{},
		}))
	}

	if cfg.P2P != nil && !cfg.P2P.Disabled() {
		err = errors.Join(err, router.AddAPI(rpc.API{
			Namespace: p2p.NamespaceRPC,
			Service:   p2p.NewP2PAPIBackend(backend.P2P(), logger),
		}))
	}

	err = errors.Join(err, router.AddRPC("/super"))
	err = errors.Join(err, router.AddAPIToRPC("/super", rpc.API{
		Namespace: "supervisor",
		Service:   &frontend2.SupervisorQueryFrontend{Supervisor: backend},
	}))
	if cfg.RPC.EnableAdmin {
		err = errors.Join(err, router.AddAPIToRPC("/super", rpc.API{
			Namespace: "admin",
			Service:   &frontend2.SupervisorAdminFrontend{},
		}))
	}

	// For each chain, register RPC frontends
	for _, chainID := range depSet.Chains() {
		route := "/chain/" + chainID.String()
		err = errors.Join(err, router.AddRPC(route))

		if cfg.RPC.EnableAdmin {
			err = errors.Join(err, router.AddAPIToRPC(route, rpc.API{
				Namespace: "admin",
				Service:   &frontend2.OpnodeAdminFrontend{},
			}))
		}
		err = errors.Join(err, router.AddAPIToRPC(route, rpc.API{
			Namespace: "optimism",
			Service:   &frontend2.OptimismFrontend{},
		}))
	}

	return err
}

func (su *Service) Start(ctx context.Context) error {
	su.log.Info("Starting JSON-RPC server")
	if err := su.rpcServer.Start(); err != nil {
		return fmt.Errorf("unable to start RPC server: %w", err)
	}

	if err := su.backend.Start(ctx); err != nil {
		return fmt.Errorf("unable to start backend: %w", err)
	}

	su.metrics.RecordUp()
	su.log.Info("JSON-RPC Server started", "endpoint", su.rpcServer.Endpoint())
	return nil
}

func (su *Service) Stop(ctx context.Context) error {
	if !su.closing.CompareAndSwap(false, true) {
		su.log.Warn("op-node is already closing")
		return nil // already closing
	}
	su.log.Info("Stopping JSON-RPC server")
	var result error
	if su.rpcServer != nil {
		if err := su.rpcServer.Stop(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop RPC server: %w", err))
		}
	}
	su.log.Info("Stopped RPC Server")
	if su.backend != nil {
		if err := su.backend.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close op-node backend: %w", err))
		}
	}
	su.log.Info("Stopped Backend")
	if su.pprofService != nil {
		if err := su.pprofService.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop PProf server: %w", err))
		}
	}
	su.log.Info("Stopped PProf")
	if su.metricsSrv != nil {
		if err := su.metricsSrv.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}
	su.log.Info("JSON-RPC server stopped")
	return result
}

func (su *Service) Stopped() bool {
	return su.closing.Load()
}

func (su *Service) HttpRPC() string {
	// the RPC endpoint is assumed to be HTTP
	return "http://" + su.rpcServer.Endpoint()
}

func (su *Service) Port() (int, error) {
	return su.rpcServer.Port()
}

func (su *Service) Backend() Backend {
	return su.backend
}
