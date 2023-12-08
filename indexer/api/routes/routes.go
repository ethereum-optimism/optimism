package routes

import (
	"github.com/ethereum-optimism/optimism/indexer/api/service"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
)

type Routes struct {
	logger log.Logger
	router *chi.Mux
	svc    service.Service
}

// NewRoutes ... Construct a new route handler instance
func NewRoutes(l log.Logger, r *chi.Mux, svc service.Service) Routes {
	return Routes{
		logger: l,
		router: r,
		svc:    svc,
	}
}
