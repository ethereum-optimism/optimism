package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const ethereumAddressRegex = `^0x[a-fA-F0-9]{40}$`

type Api struct {
	log           log.Logger
	Router        *chi.Mux
	MetricsRouter *chi.Mux
}

const (
	MetricsNamespace = "op_indexer"
)

func NewApi(logger log.Logger, bv database.BridgeTransfersView) *Api {
	apiRouter := chi.NewRouter()
	h := routes.NewRoutes(logger, bv, apiRouter)

	apiRouter.Use(middleware.Heartbeat("/healthz"))

	apiRouter.Get(fmt.Sprintf("/api/v0/deposits/{address:%s}", ethereumAddressRegex), h.L1DepositsHandler)
	apiRouter.Get(fmt.Sprintf("/api/v0/withdrawals/{address:%s}", ethereumAddressRegex), h.L2WithdrawalsHandler)

	metricsRouter := chi.NewRouter()
	registry := metrics.NewRegistry()
	promRecorder := metrics.NewPromHTTPRecorder(registry, MetricsNamespace)
	promHandler := metrics.NewHTTPRecordingMiddleware(promRecorder, apiRouter)

	metricsRouter.Use(middleware.Heartbeat("/healthz"))

	metricsRouter.Get("/metrics", promHandler.ServeHTTP)

	return &Api{log: logger, Router: apiRouter, MetricsRouter: metricsRouter}
}

func (a *Api) Listen(ctx context.Context, port int) error {
	a.log.Info("api server listening...", "port", port)
	server := http.Server{Addr: fmt.Sprintf(":%d", port), Handler: a.Router}
	err := httputil.ListenAndServeContext(ctx, &server)
	if err != nil {
		a.log.Error("api server stopped", "err", err)
	} else {
		a.log.Info("api server stopped")
	}

	return err
}

func (a *Api) ListenMetrics(ctx context.Context, port int) error {
	a.log.Info("metrics server listening...", "port", port)

	server := http.Server{Addr: fmt.Sprintf(":%d", port), Handler: a.MetricsRouter}
	err := httputil.ListenAndServeContext(ctx, &server)
	if err != nil {
		a.log.Error("api server stopped", "err", err)
	} else {
		a.log.Info("api server stopped")
	}

	return err
}
