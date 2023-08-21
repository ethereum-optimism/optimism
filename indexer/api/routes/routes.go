package routes

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/log"
	"github.com/go-chi/chi/v5"
)

type Routes struct {
	Logger              log.Logger
	BridgeTransfersView database.BridgeTransfersView
	Router              *chi.Mux
}

func NewRoutes(logger log.Logger, bv database.BridgeTransfersView, r *chi.Mux) Routes {
	return Routes{
		Logger:              logger,
		BridgeTransfersView: bv,
		Router:              r,
	}
}
