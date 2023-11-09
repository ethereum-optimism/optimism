// Database module defines the data DB struct which wraps specific DB interfaces for L1/L2 block headers, contract events, bridging schemas.
package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/indexer/config"
	_ "github.com/ethereum-optimism/optimism/indexer/database/serializers"
	"github.com/ethereum-optimism/optimism/op-service/retry"

	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	gorm *gorm.DB
	log  log.Logger

	Blocks             BlocksDB
	ContractEvents     ContractEventsDB
	BridgeTransfers    BridgeTransfersDB
	BridgeMessages     BridgeMessagesDB
	BridgeTransactions BridgeTransactionsDB
}

// NewDB connects to the configured DB, and provides client-bindings to it.
// The initial connection may fail, or the dial may be cancelled with the provided context.
func NewDB(ctx context.Context, log log.Logger, dbConfig config.DBConfig) (*DB, error) {
	log = log.New("module", "db")

	dsn := fmt.Sprintf("host=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Name)
	if dbConfig.Port != 0 {
		dsn += fmt.Sprintf(" port=%d", dbConfig.Port)
	}
	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}

	gormConfig := gorm.Config{
		Logger: newLogger(log),

		// The indexer will explicitly manage the transactions
		SkipDefaultTransaction: true,

		// The postgres parameter counter for a given query is represented with uint16,
		// resulting in a parameter limit of 65535. In order to avoid reaching this limit
		// we'll utilize a batch size of 3k for inserts, well below the limit as long as
		// the number of columns < 20.
		CreateBatchSize: 3_000,
	}

	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}
	gorm, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gorm, err := gorm.Open(postgres.Open(dsn), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		return gorm, nil
	})

	if err != nil {
		return nil, err
	}

	db := &DB{
		gorm:               gorm,
		log:                log,
		Blocks:             newBlocksDB(log, gorm),
		ContractEvents:     newContractEventsDB(log, gorm),
		BridgeTransfers:    newBridgeTransfersDB(log, gorm),
		BridgeMessages:     newBridgeMessagesDB(log, gorm),
		BridgeTransactions: newBridgeTransactionsDB(log, gorm),
	}

	return db, nil
}

// Transaction executes all operations conducted with the supplied database in a single
// transaction. If the supplied function errors, the transaction is rolled back.
func (db *DB) Transaction(fn func(db *DB) error) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		txDB := &DB{
			gorm:               tx,
			Blocks:             newBlocksDB(db.log, tx),
			ContractEvents:     newContractEventsDB(db.log, tx),
			BridgeTransfers:    newBridgeTransfersDB(db.log, tx),
			BridgeMessages:     newBridgeMessagesDB(db.log, tx),
			BridgeTransactions: newBridgeTransactionsDB(db.log, tx),
		}

		return fn(txDB)
	})
}

func (db *DB) Close() error {
	db.log.Info("closing database")
	sql, err := db.gorm.DB()
	if err != nil {
		return err
	}

	return sql.Close()
}

func (db *DB) ExecuteSQLMigration(migrationsFolder string) error {
	err := filepath.Walk(migrationsFolder, func(path string, info os.FileInfo, err error) error {
		// Check for any walking error
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to process migration file: %s", path))
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read the migration file content
		db.log.Info("reading sql file", "path", path)
		fileContent, readErr := os.ReadFile(path)
		if readErr != nil {
			return errors.Wrap(readErr, fmt.Sprintf("Error reading SQL file: %s", path))
		}

		// Execute the migration
		db.log.Info("executing sql file", "path", path)
		execErr := db.gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(execErr, fmt.Sprintf("Error executing SQL script: %s", path))
		}

		return nil
	})

	db.log.Info("finished migrations")
	return err
}
