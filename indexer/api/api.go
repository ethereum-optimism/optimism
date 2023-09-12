package api

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

const ethereumAddressRegex = `^0x[a-fA-F0-9]{40}$`

type Api struct {
	log             log.Logger
	Router          *chi.Mux
	serverConfig    config.ServerConfig
	metricsConfig   config.ServerConfig
	metricsRegistry *prometheus.Registry
}

const (
	MetricsNamespace = "op_indexer"
)

func chiMetricsMiddleware(rec metrics.HTTPRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return metrics.NewHTTPRecordingMiddleware(rec, next)
	}
}

func NewApi(logger log.Logger, bv database.BridgeTransfersView, serverConfig config.ServerConfig, metricsConfig config.ServerConfig) *Api {
	apiRouter := chi.NewRouter()
	h := routes.NewRoutes(logger, bv, apiRouter)

	mr := metrics.NewRegistry()
	promRecorder := metrics.NewPromHTTPRecorder(mr, MetricsNamespace)

	apiRouter.Use(chiMetricsMiddleware(promRecorder))
	apiRouter.Use(middleware.Recoverer)
	apiRouter.Use(middleware.Heartbeat("/healthz"))

	apiRouter.Get(fmt.Sprintf("/api/v0/deposits/{address:%s}", ethereumAddressRegex), h.L1DepositsHandler)
	apiRouter.Get(fmt.Sprintf("/api/v0/withdrawals/{address:%s}", ethereumAddressRegex), h.L2WithdrawalsHandler)

	return &Api{log: logger, Router: apiRouter, metricsRegistry: mr, serverConfig: serverConfig, metricsConfig: metricsConfig}
}

func (a *Api) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

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

	runProcess(a.startServer)
	runProcess(a.startMetricsServer)

	wg.Wait()

	err := <-errCh
	if err != nil {
		a.log.Error("api stopped", "err", err)
	} else {
		a.log.Info("api stopped")
	}

	return err
}

func (a *Api) startServer(ctx context.Context) error {
	a.log.Info("api server listening...", "port", a.serverConfig.Port)
	server := http.Server{Addr: fmt.Sprintf(":%d", a.serverConfig.Port), Handler: a.Router}
	err := httputil.ListenAndServeContext(ctx, &server)
	if err != nil {
		a.log.Error("api server stopped", "err", err)
	} else {
		a.log.Info("api server stopped")
	}
	return err
}

func (a *Api) startMetricsServer(ctx context.Context) error {
	a.log.Info("starting metrics server...", "port", a.metricsConfig.Port)
	err := metrics.ListenAndServe(ctx, a.metricsRegistry, a.metricsConfig.Host, a.metricsConfig.Port)
	if err != nil {
		a.log.Error("metrics server stopped", "err", err)
	} else {
		a.log.Info("metrics server stopped")
	}
	return err
}
