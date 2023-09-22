package routes

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
)

// Routes ... Route handler struct
type Routes struct {
	Logger              log.Logger
	BridgeTransfersView database.BridgeTransfersView
	Router              *chi.Mux
	v                   *Validator
}

// NewRoutes ... Construct a new route handler instance
func NewRoutes(logger log.Logger, bv database.BridgeTransfersView, r *chi.Mux) Routes {
	return Routes{
		Logger:              logger,
		BridgeTransfersView: bv,
		Router:              r,
	}
}
