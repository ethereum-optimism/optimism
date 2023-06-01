package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	gorm *gorm.DB

	Blocks         BlocksDB
	ContractEvents ContractEventsDB
	Bridge         BridgeDB
}

func NewDB(dsn string) (*DB, error) {
	gorm, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		// The indexer will explictly manage the transaction
		// flow processing blocks
		SkipDefaultTransaction: true,
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
