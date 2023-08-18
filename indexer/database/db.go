// Database module defines the data DB struct which wraps specific DB interfaces for L1/L2 block headers, contract events, bridging schemas.
package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DB struct {
	gorm *gorm.DB

	Blocks         BlocksDB
	ContractEvents ContractEventsDB
	Bridge         BridgeDB
}

func NewDB(dsn string) (*DB, error) {
	gorm, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// The indexer will explicitly manage the transaction
		// flow processing blocks
		SkipDefaultTransaction: true,

		// We may choose to create an adapter such that the
		// logger emits to the geth logger when on DEBUG mode
		Logger: logger.Default.LogMode(logger.Silent),
	})

	if err != nil {
		return nil, err
	}

	db := &DB{
		gorm:           gorm,
		Blocks:         newBlocksDB(gorm),
		ContractEvents: newContractEventsDB(gorm),
		Bridge:         newBridgeDB(gorm),
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
		gorm:           tx,
		Blocks:         newBlocksDB(tx),
		ContractEvents: newContractEventsDB(tx),
		Bridge:         newBridgeDB(tx),
	}
}
