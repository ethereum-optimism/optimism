// Database module defines the data DB struct which wraps specific DB interfaces for L1/L2 block headers, contract events, bridging schemas.
package database

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/indexer/config"
	_ "github.com/ethereum-optimism/optimism/indexer/database/serializers"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/pkg/errors"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

func NewDB(dbConfig config.DBConfig) (*DB, error) {
	retryStrategy := &retry.ExponentialStrategy{Min: 1000, Max: 20_000, MaxJitter: 250}

	dsn := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Port, dbConfig.Name)
	if dbConfig.User != "" {
		dsn += fmt.Sprintf(" user=%s", dbConfig.User)
	}
	if dbConfig.Password != "" {
		dsn += fmt.Sprintf(" password=%s", dbConfig.Password)
	}

	gormConfig := gorm.Config{
		// The indexer will explicitly manage the transactions
		SkipDefaultTransaction: true,
		Logger:                 logger.Default.LogMode(logger.Silent),
	}

	gorm, err := retry.Do[*gorm.DB](context.Background(), 10, retryStrategy, func() (*gorm.DB, error) {
		gorm, err := gorm.Open(postgres.Open(dsn), &gormConfig)

		if err != nil {
			return nil, errors.Wrap(err, "failed to connect to database")
		}
		return gorm, nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to database after multiple retries")
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
