package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ethereum-optimism/optimism/indexer/api/routes"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum-optimism/optimism/op-service/httputil"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
)

const ethereumAddressRegex = `^0x[a-fA-F0-9]{40}$`

type Api struct {
	log    log.Logger
	Router *chi.Mux
}

func NewApi(logger log.Logger, bv database.BridgeTransfersView) *Api {
	r := chi.NewRouter()
	h := routes.NewRoutes(logger, bv, r)

	r.Get("/healthz", h.HealthzHandler)
	r.Get(fmt.Sprintf("/api/v0/deposits/{address:%s}", ethereumAddressRegex), h.L1DepositsHandler)
	r.Get(fmt.Sprintf("/api/v0/withdrawals/{address:%s}", ethereumAddressRegex), h.L2WithdrawalsHandler)
	return &Api{log: logger, Router: r}
}

func (a *Api) Listen(ctx context.Context, port int) error {
	a.log.Info("starting api server", "port", port)
	server := http.Server{Addr: fmt.Sprintf(":%d", port), Handler: a.Router}
	err := httputil.ListenAndServeContext(ctx, &server)
	if err != nil {
		a.log.Error("api server shutdown", "err", err)
	} else {
		a.log.Info("api server shutdown")
	}

	return err
}
