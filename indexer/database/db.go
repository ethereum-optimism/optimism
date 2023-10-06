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

var (
	// The postgres parameter counter for a given query is stored via a  uint16,
	// resulting in a parameter limit of 65535. In order to avoid reaching this limit
	// we'll utilize a batch size of 3k for inserts, well below as long as the the number
	// of columns < 20.
	batchInsertSize int = 3_000
)

type DB struct {
	gorm *gorm.DB

	Blocks             BlocksDB
	ContractEvents     ContractEventsDB
	BridgeTransfers    BridgeTransfersDB
	BridgeMessages     BridgeMessagesDB
	BridgeTransactions BridgeTransactionsDB
}

func NewDB(log log.Logger, dbConfig config.DBConfig) (*DB, error) {
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}

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
		// The indexer will explicitly manage the transactions
		SkipDefaultTransaction: true,
		Logger:                 newLogger(log),
	}

	gorm, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gorm, err := gorm.Open(postgres.Open(dsn), &gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}

		return gorm, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database after multiple retries: %w", err)
	}

	db := &DB{
		gorm:               gorm,
		Blocks:             newBlocksDB(gorm),
		ContractEvents:     newContractEventsDB(gorm),
		BridgeTransfers:    newBridgeTransfersDB(gorm),
		BridgeMessages:     newBridgeMessagesDB(gorm),
		BridgeTransactions: newBridgeTransactionsDB(gorm),
	}

	return db, nil
}

// Transaction executes all operations conducted with the supplied database in a single
// transaction. If the supplied function errors, the transaction is rolled back.
func (db *DB) Transaction(fn func(db *DB) error) error {
	return db.gorm.Transaction(func(tx *gorm.DB) error {
		return fn(dbFromGormTx(tx))
	})
}

func (db *DB) Close() error {
	sql, err := db.gorm.DB()
	if err != nil {
		return err
	}

	return sql.Close()
}

func dbFromGormTx(tx *gorm.DB) *DB {
	return &DB{
		gorm:               tx,
		Blocks:             newBlocksDB(tx),
		ContractEvents:     newContractEventsDB(tx),
		BridgeTransfers:    newBridgeTransfersDB(tx),
		BridgeMessages:     newBridgeMessagesDB(tx),
		BridgeTransactions: newBridgeTransactionsDB(tx),
	}
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
		fileContent, readErr := os.ReadFile(path)
		if readErr != nil {
			return errors.Wrap(readErr, fmt.Sprintf("Error reading SQL file: %s", path))
		}

		// Execute the migration
		execErr := db.gorm.Exec(string(fileContent)).Error
		if execErr != nil {
			return errors.Wrap(execErr, fmt.Sprintf("Error executing SQL script: %s", path))
		}

		return nil
	})

	return err
}
