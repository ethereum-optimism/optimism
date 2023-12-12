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
	// NOTE - This can be further broken out over time as new version iterations
	// are implemented
	HealthPath      = "/healthz"
	DepositsPath    = "/api/v0/deposits/"
	WithdrawalsPath = "/api/v0/withdrawals/"

	SupplyPath = "/api/v0/supply"
)

// Api ... Indexer API struct
// TODO : Structured error responses
type APIService struct {
	log    log.Logger
	router *chi.Mux

	bv      database.BridgeTransfersView
	dbClose func() error

	metricsRegistry *prometheus.Registry

	apiServer     *httputil.HTTPServer
	metricsServer *httputil.HTTPServer

	stopped atomic.Bool
}

// chiMetricsMiddleware ... Injects a metrics recorder into request processing middleware
func chiMetricsMiddleware(rec metrics.HTTPRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return metrics.NewHTTPRecordingMiddleware(rec, next)
	}
}

// NewApi ... Construct a new api instance
func NewApi(ctx context.Context, log log.Logger, cfg *Config) (*APIService, error) {
	out := &APIService{log: log, metricsRegistry: metrics.NewRegistry()}
	if err := out.initFromConfig(ctx, cfg); err != nil {
		return nil, errors.Join(err, out.Stop(ctx)) // close any resources we may have opened already
	}
	return out, nil
}

func (a *APIService) initFromConfig(ctx context.Context, cfg *Config) error {
	if err := a.initDB(ctx, cfg.DB); err != nil {
		return fmt.Errorf("failed to init DB: %w", err)
	}
	if err := a.startMetricsServer(cfg.MetricsServer); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}
	a.initRouter(cfg.HTTPServer)
	if err := a.startServer(cfg.HTTPServer); err != nil {
		return fmt.Errorf("failed to start API server: %w", err)
	}
	return nil
}

func (a *APIService) Start(ctx context.Context) error {
	// Completed all setup-up jobs at init-time already,
	// and the API service does not have any other special starting routines or background-jobs to start.
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
	if a.dbClose != nil {
		if err := a.dbClose(); err != nil {
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

// Addr ... returns the address that the HTTP server is listening on (excl. http:// prefix, just the host and port)
func (a *APIService) Addr() string {
	if a.apiServer == nil {
		return ""
	}
	return a.apiServer.Addr().String()
}

func (a *APIService) initDB(ctx context.Context, connector DBConnector) error {
	db, err := connector.OpenDB(ctx, a.log)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	a.dbClose = db.Closer
	a.bv = db.BridgeTransfers
	return nil
}

func (a *APIService) initRouter(apiConfig config.ServerConfig) {
	v := new(service.Validator)

	svc := service.New(v, a.bv, a.log)
	apiRouter := chi.NewRouter()
	h := routes.NewRoutes(a.log, apiRouter, svc)

	promRecorder := metrics.NewPromHTTPRecorder(a.metricsRegistry, MetricsNamespace)

	apiRouter.Use(chiMetricsMiddleware(promRecorder))
	apiRouter.Use(middleware.Timeout(time.Duration(apiConfig.WriteTimeout) * time.Second))
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(middleware.Heartbeat(HealthPath))

	apiRouter.Get(fmt.Sprintf(DepositsPath+addressParam, ethereumAddressRegex), h.L1DepositsHandler)
	apiRouter.Get(fmt.Sprintf(WithdrawalsPath+addressParam, ethereumAddressRegex), h.L2WithdrawalsHandler)
	apiRouter.Get(SupplyPath, h.SupplyView)
	a.router = apiRouter
}

// startServer ... Starts the API server
func (a *APIService) startServer(serverConfig config.ServerConfig) error {
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

// startMetricsServer ... Starts the metrics server
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
