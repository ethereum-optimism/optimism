package database

import (
	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"

	"github.com/google/uuid"
)

/**
 * Types
 */

type ContractEvent struct {
	GUID            uuid.UUID   `gorm:"primaryKey"`
	BlockHash       common.Hash `gorm:"serializer:json"`
	TransactionHash common.Hash `gorm:"serializer:json"`

	EventSignature common.Hash `gorm:"serializer:json"`
	LogIndex       uint64
	Timestamp      uint64
}

type L1ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type L2ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type ContractEventsView interface {
}

type ContractEventsDB interface {
	ContractEventsView

	StoreL1ContractEvents([]*L1ContractEvent) error
	StoreL2ContractEvents([]*L2ContractEvent) error
}

/**
 * Implementation
 */

type contractEventsDB struct {
	gorm *gorm.DB
}

func newContractEventsDB(db *gorm.DB) ContractEventsDB {
	return &contractEventsDB{gorm: db}
}

// L1

func (db *contractEventsDB) StoreL1ContractEvents(events []*L1ContractEvent) error {
	result := db.gorm.Create(&events)
	return result.Error
}

// L2

func (db *contractEventsDB) StoreL2ContractEvents(events []*L2ContractEvent) error {
	result := db.gorm.Create(&events)
	return result.Error
}
