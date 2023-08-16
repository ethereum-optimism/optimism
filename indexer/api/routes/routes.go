package routes

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/log"
)

type Routes struct {
	Logger              log.Logger
	BridgeTransfersView database.BridgeTransfersView
}

func NewRoutes(logger log.Logger, bv database.BridgeTransfersView) Routes {
	return Routes{
		Logger:              logger,
		BridgeTransfersView: bv,
	}
}
