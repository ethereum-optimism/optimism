package api

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

const ethereumAddressRegex = `^0x[a-fA-F0-9]{40}$`

// Api ... Indexer API struct
// TODO : Structured error responses
type API struct {
	log             log.Logger
	router          *chi.Mux
	serverConfig    config.ServerConfig
	metricsConfig   config.ServerConfig
	metricsRegistry *prometheus.Registry
}

const (
	MetricsNamespace = "op_indexer_api"
	addressParam     = "{address:%s}"

	// Endpoint paths
	// NOTE - This can be further broken out over time as new version iterations
	// are implemented
	HealthPath      = "/healthz"
	DepositsPath    = "/api/v0/deposits/"
	WithdrawalsPath = "/api/v0/withdrawals/"
)

// chiMetricsMiddleware ... Injects a metrics recorder into request processing middleware
func chiMetricsMiddleware(rec metrics.HTTPRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return metrics.NewHTTPRecordingMiddleware(rec, next)
	}
}

// NewApi ... Construct a new api instance
func NewApi(logger log.Logger, bv database.BridgeTransfersView, serverConfig config.ServerConfig, metricsConfig config.ServerConfig) *API {
	// (1) Initialize dependencies
	apiRouter := chi.NewRouter()
	h := routes.NewRoutes(logger, bv, apiRouter)

	mr := metrics.NewRegistry()
	promRecorder := metrics.NewPromHTTPRecorder(mr, MetricsNamespace)

	// (2) Inject routing middleware
	apiRouter.Use(chiMetricsMiddleware(promRecorder))
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(middleware.Heartbeat(HealthPath))

	// (3) Set GET routes
	apiRouter.Get(fmt.Sprintf(DepositsPath+addressParam, ethereumAddressRegex), h.L1DepositsHandler)
	apiRouter.Get(fmt.Sprintf(WithdrawalsPath+addressParam, ethereumAddressRegex), h.L2WithdrawalsHandler)

	return &API{log: logger, router: apiRouter, metricsRegistry: mr, serverConfig: serverConfig, metricsConfig: metricsConfig}
}

// Run ... Runs the API server routines
func (a *API) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// (1) Construct an inner function that will start a goroutine
	//    and handle any panics that occur on a shared error channel
	processCtx, processCancel := context.WithCancel(ctx)
	runProcess := func(start func(ctx context.Context) error) {
		wg.Add(1)
		go func() {
			defer func() {
				if err := recover(); err != nil {
					a.log.Error("halting api on panic", "err", err)
					debug.PrintStack()
					errCh <- fmt.Errorf("panic: %v", err)
				}

				processCancel()
				wg.Done()
			}()

			errCh <- start(processCtx)
		}()
	}

	// (2) Start the API and metrics servers
	runProcess(a.startServer)
	runProcess(a.startMetricsServer)

	// (3) Wait for all processes to complete
	wg.Wait()

	err := <-errCh
	if err != nil {
		a.log.Error("api stopped", "err", err)
	} else {
		a.log.Info("api stopped")
	}

	return err
}

// Port ... Returns the the port that server is listening on
func (a *API) Port() int {
	return a.serverConfig.Port
}

// startServer ... Starts the API server
func (a *API) startServer(ctx context.Context) error {
	a.log.Info("api server listening...", "port", a.serverConfig.Port)
	server := http.Server{Addr: fmt.Sprintf(":%d", a.serverConfig.Port), Handler: a.router}

	addr := fmt.Sprintf(":%d", a.serverConfig.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		a.log.Error("Listen:", err)
		return err
	}
	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return fmt.Errorf("failed to get TCP address from network listener")
	}

	// Update the port in the config in case the OS chose a different port
	// than the one we requested (e.g. using port 0 to fetch a random open port)
	a.serverConfig.Port = tcpAddr.Port

	err = http.Serve(listener, server.Handler)
	if err != nil {
		a.log.Error("api server stopped with error", "err", err)
	} else {
		a.log.Info("api server stopped")
	}
	return err
}

// startMetricsServer ... Starts the metrics server
func (a *API) startMetricsServer(ctx context.Context) error {
	a.log.Info("starting metrics server...", "port", a.metricsConfig.Port)
	err := metrics.ListenAndServe(ctx, a.metricsRegistry, a.metricsConfig.Host, a.metricsConfig.Port)
	if err != nil {
		a.log.Error("metrics server stopped", "err", err)
	} else {
		a.log.Info("metrics server stopped")
	}
	return err
}
