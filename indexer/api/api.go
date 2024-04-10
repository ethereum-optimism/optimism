package api

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/api/service"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
)

const ethereumAddressRegex = `^0x[a-fA-F0-9]{40}$`

const (
	MetricsNamespace = "op_indexer_api"
	addressParam     = "{address:%s}"

	// Endpoint paths
	DocsPath        = "/docs"
	HealthPath      = "/healthz"
	DepositsPath    = "/api/v0/deposits/"
	WithdrawalsPath = "/api/v0/withdrawals/"

	SupplyPath = "/api/v0/supply"
)

// APIService handles the overall API functionality, including the HTTP server and metrics.
type APIService struct {
	log    log.Logger
	router *chi.Mux

	db           database.BridgeTransfersView
	dbCloser     func() error
	apiServer    *httputil.HTTPServer
	metricsServer *httputil.HTTPServer

	metricsRegistry *prometheus.Registry
	stopped         atomic.Bool
}

// NewAPIService constructs a new APIService instance.
func NewAPIService(ctx context.Context, log log.Logger, cfg *config.Config) (*APIService, error) {
	out := &APIService{
		log:            log,
		metricsRegistry: metrics.NewRegistry(),
	}

	if err := out.initFromConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, out.Stop(ctx))
	}

	return out, nil
}

func (a *APIService) initFromConfig(ctx context.Context, cfg *config.Config) error {
	if err := a.initDB(ctx, cfg.DB); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}

	if err := a.startMetricsServer(cfg.MetricsServer); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	a.initRouter(cfg.HTTPServer)

	if err := a.startAPIServer(cfg.HTTPServer); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	return nil
}

func (a *APIService) Start(ctx context.Context) error {
	// No additional startup tasks required.
	return nil
}

func (a *APIService) Stop(ctx context.Context) error {
	var result error
	if a.apiServer != nil {
		if err := a.apiServer.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop API server: %w", err))
		}
	}
	if a.metricsServer != nil {
		if err := a.metricsServer.Stop(ctx); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to stop metrics server: %w", err))
		}
	}
	if a.dbCloser != nil {
		if err := a.dbCloser(); err != nil {
			result = errors.Join(result, fmt.Errorf("failed to close DB: %w", err))
		}
	}
	a.stopped.Store(true)
	a.log.Info("API service shutdown complete")
	return result
}

func (a *APIService) Stopped() bool {
	return a.stopped.Load()
}

func (a *APIService) Addr() string {
	if a.apiServer == nil {
		return ""
	}
	return a.apiServer.Addr().String()
}

func (a *APIService) initDB(ctx context.Context, connector database.Connector) error {
	db, err := connector.OpenDB(ctx, a.log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	a.db = db.BridgeTransfers
	a.dbCloser = db.Closer
	return nil
}

func (a *APIService) initRouter(apiConfig config.ServerConfig) {
	v := new(service.Validator)
	svc := service.New(v, a.db, a.log)
	routes := routes.NewRoutes(a.log, svc)

	a.router = chi.NewRouter()
	a.router.Use(middleware.Logger)
	a.router.Use(middleware.Timeout(time.Duration(apiConfig.WriteTimeout) * time.Second))
	a.router.Use(middleware.Recoverer)
	a.router.Use(middleware.Heartbeat(HealthPath))
	a.router.Use(chiMetricsMiddleware(metrics.NewPromHTTPRecorder(a.metricsRegistry, MetricsNamespace)))

	a.router.Get(fmt.Sprintf(DepositsPath+addressParam, ethereumAddressRegex), routes.L1DepositsHandler)
	a.router.Get(fmt.Sprintf(WithdrawalsPath+addressParam, ethereumAddressRegex), routes.L2WithdrawalsHandler)
	a.router.Get(SupplyPath, routes.SupplyView)
	a.router.Get(DocsPath, routes.DocsHandler)
}

func (a *APIService) startAPIServer(serverConfig config.ServerConfig) error {
	a.log.Debug("API server listening...", "port", serverConfig.Port)

	addr := net.JoinHostPort(serverConfig.Host, strconv.Itoa(serverConfig.Port))
	srv, err := httputil.StartHTTPServer(addr, a.router)
	if err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}

	a.log.Info("API server started", "addr", srv.Addr().String())
	a.apiServer = srv
	return nil
}

func (a *APIService) startMetricsServer(metricsConfig config.ServerConfig) error {
	a.log.Debug("starting metrics server...", "port", metricsConfig.Port)
	srv, err := metrics.StartServer(a.metricsRegistry, metricsConfig.Host, metricsConfig.Port)
	if err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	a.log.Info("Metrics server started", "addr", srv.Addr().String())
	a.metricsServer = srv
	return nil
}

func chiMetricsMiddleware(rec metrics.HTTPRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return metrics.NewHTTPRecordingMiddleware(rec, next)
	}
}
