package database

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gorm.io/gorm"
)

/**
 * Types
 */

type ContractEvent struct {
	GUID            string      `gorm:"primaryKey"`
	BlockHash       common.Hash `gorm:"serializer:json"`
	TransactionHash common.Hash `gorm:"serializer:json"`

	EventSignature hexutil.Bytes `gorm:"serializer:json"`
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
	L1ContractEventByGUID(string) (*L1ContractEvent, error)
	L2ContractEventByGUID(string) (*L2ContractEvent, error)
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

func (db *contractEventsDB) L1ContractEventByGUID(guid string) (*L1ContractEvent, error) {
	var event L1ContractEvent
	result := db.gorm.First(&event, "guid = ?", guid)
	if result.Error != nil {
		return nil, result.Error
	}

	return &event, nil
}

// L2

func (db *contractEventsDB) StoreL2ContractEvents(events []*L2ContractEvent) error {
	result := db.gorm.Create(&events)
	return result.Error
}

func (db *contractEventsDB) L2ContractEventByGUID(guid string) (*L2ContractEvent, error) {
	var event L2ContractEvent
	result := db.gorm.First(&event, "guid = ?", guid)
	if result.Error != nil {
		return nil, result.Error
	}

	return &event, nil
}
