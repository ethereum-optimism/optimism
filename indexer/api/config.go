package api

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/config"
	"github.com/ethereum-optimism/optimism/indexer/database"
)

// DB represents the abstract DB access the API has.
type DB struct {
	BridgeTransfers database.BridgeTransfersView
	Closer          func() error
}

// DBConfigConnector implements a fully config based DBConnector
type DBConfigConnector struct {
	config.DBConfig
}

func (cfg *DBConfigConnector) OpenDB(ctx context.Context, log log.Logger) (*DB, error) {
	db, err := database.NewDB(ctx, log, cfg.DBConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DB{
		BridgeTransfers: db.BridgeTransfers,
		Closer:          db.Close,
	}, nil
}

type TestDBConnector struct {
	BridgeTransfers database.BridgeTransfersView
}

func (tdb *TestDBConnector) OpenDB(ctx context.Context, log log.Logger) (*DB, error) {
	return &DB{
		BridgeTransfers: tdb.BridgeTransfers,
		Closer: func() error {
			log.Info("API service closed test DB view")
			return nil
		},
	}, nil
}

// DBConnector is an interface: the config may provide different ways to access the DB.
// This is implemented in tests to provide custom DB views, or share the DB with other services.
type DBConnector interface {
	OpenDB(ctx context.Context, log log.Logger) (*DB, error)
}

// Config for the API service
type Config struct {
	DB            DBConnector
	HTTPServer    config.ServerConfig
	MetricsServer config.ServerConfig
}
