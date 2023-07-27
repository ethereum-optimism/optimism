package database

import (
	"errors"

	"gorm.io/gorm"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

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

func ContractEventFromGethLog(log *types.Log, timestamp uint64) ContractEvent {
	return ContractEvent{
		GUID: uuid.New(),

		BlockHash:       log.BlockHash,
		TransactionHash: log.TxHash,

		EventSignature: log.Topics[0],
		LogIndex:       uint64(log.Index),

		Timestamp: timestamp,
	}
}

type L1ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type L2ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type ContractEventsView interface {
	L1ContractEvent(uuid.UUID) (*L1ContractEvent, error)
	L1ContractEventByTxLogIndex(common.Hash, uint64) (*L1ContractEvent, error)

	L2ContractEvent(uuid.UUID) (*L2ContractEvent, error)
	L2ContractEventByTxLogIndex(common.Hash, uint64) (*L2ContractEvent, error)
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

func (db *contractEventsDB) L1ContractEvent(uuid uuid.UUID) (*L1ContractEvent, error) {
	var l1ContractEvent L1ContractEvent
	result := db.gorm.Where(&ContractEvent{GUID: uuid}).Take(&l1ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l1ContractEvent, nil
}

func (db *contractEventsDB) L1ContractEventByTxLogIndex(txHash common.Hash, logIndex uint64) (*L1ContractEvent, error) {
	var l1ContractEvent L1ContractEvent
	result := db.gorm.Where(&ContractEvent{TransactionHash: txHash, LogIndex: logIndex}).Take(&l1ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l1ContractEvent, nil
}

// L2

func (db *contractEventsDB) StoreL2ContractEvents(events []*L2ContractEvent) error {
	result := db.gorm.Create(&events)
	return result.Error
}

func (db *contractEventsDB) L2ContractEvent(uuid uuid.UUID) (*L2ContractEvent, error) {
	var l2ContractEvent L2ContractEvent
	result := db.gorm.Where(&ContractEvent{GUID: uuid}).Take(&l2ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l2ContractEvent, nil
}

func (db *contractEventsDB) L2ContractEventByTxLogIndex(txHash common.Hash, logIndex uint64) (*L2ContractEvent, error) {
	var l2ContractEvent L2ContractEvent
	result := db.gorm.Where(&ContractEvent{TransactionHash: txHash, LogIndex: logIndex}).Take(&l2ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l2ContractEvent, nil
}
