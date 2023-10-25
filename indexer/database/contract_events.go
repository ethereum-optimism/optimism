package database

import (
	"errors"
	"fmt"
	"math/big"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/google/uuid"
)

/**
 * Types
 */

type ContractEvent struct {
	GUID uuid.UUID `gorm:"primaryKey"`

	// Some useful derived fields
	BlockHash       common.Hash    `gorm:"serializer:bytes"`
	ContractAddress common.Address `gorm:"serializer:bytes"`
	TransactionHash common.Hash    `gorm:"serializer:bytes"`
	LogIndex        uint64

	EventSignature common.Hash `gorm:"serializer:bytes"`
	Timestamp      uint64

	// NOTE: NOT ALL THE DERIVED FIELDS ON `types.Log` ARE
	// AVAILABLE. FIELDS LISTED ABOVE ARE FILLED IN
	RLPLog *types.Log `gorm:"serializer:rlp;column:rlp_bytes"`
}

func ContractEventFromLog(log *types.Log, timestamp uint64) ContractEvent {
	eventSig := common.Hash{}
	if len(log.Topics) > 0 {
		eventSig = log.Topics[0]
	}

	return ContractEvent{
		GUID: uuid.New(),

		BlockHash:       log.BlockHash,
		TransactionHash: log.TxHash,
		ContractAddress: log.Address,

		EventSignature: eventSig,
		LogIndex:       uint64(log.Index),

		Timestamp: timestamp,

		RLPLog: log,
	}
}

func (c *ContractEvent) AfterFind(tx *gorm.DB) error {
	// Fill in some of the derived fields that are not
	// populated when decoding the RLPLog from RLP
	c.RLPLog.BlockHash = c.BlockHash
	c.RLPLog.TxHash = c.TransactionHash
	c.RLPLog.Index = uint(c.LogIndex)
	return nil
}

type L1ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type L2ContractEvent struct {
	ContractEvent `gorm:"embedded"`
}

type ContractEventsView interface {
	L1ContractEvent(uuid.UUID) (*L1ContractEvent, error)
	L1ContractEventWithFilter(ContractEvent) (*L1ContractEvent, error)
	L1ContractEventsWithFilter(ContractEvent, *big.Int, *big.Int) ([]L1ContractEvent, error)
	L1LatestContractEventWithFilter(ContractEvent) (*L1ContractEvent, error)

	L2ContractEvent(uuid.UUID) (*L2ContractEvent, error)
	L2ContractEventWithFilter(ContractEvent) (*L2ContractEvent, error)
	L2ContractEventsWithFilter(ContractEvent, *big.Int, *big.Int) ([]L2ContractEvent, error)
	L2LatestContractEventWithFilter(ContractEvent) (*L2ContractEvent, error)

	ContractEventsWithFilter(ContractEvent, string, *big.Int, *big.Int) ([]ContractEvent, error)
}

type ContractEventsDB interface {
	ContractEventsView

	StoreL1ContractEvents([]L1ContractEvent) error
	StoreL2ContractEvents([]L2ContractEvent) error
}

/**
 * Implementation
 */

type contractEventsDB struct {
	log  log.Logger
	gorm *gorm.DB
}

func newContractEventsDB(log log.Logger, db *gorm.DB) ContractEventsDB {
	return &contractEventsDB{log: log.New("table", "events"), gorm: db}
}

// L1

func (db *contractEventsDB) StoreL1ContractEvents(events []L1ContractEvent) error {
	// Since the block hash refers back to L1, we dont necessarily have to check
	// that the RLP bytes match when doing conflict resolution.
	deduped := db.gorm.Clauses(clause.OnConflict{OnConstraint: "l1_contract_events_block_hash_log_index_key", DoNothing: true})
	result := deduped.Create(&events)
	if result.Error == nil && int(result.RowsAffected) < len(events) {
		db.log.Warn("ignored L1 contract event duplicates", "duplicates", len(events)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *contractEventsDB) L1ContractEvent(uuid uuid.UUID) (*L1ContractEvent, error) {
	return db.L1ContractEventWithFilter(ContractEvent{GUID: uuid})
}

func (db *contractEventsDB) L1ContractEventWithFilter(filter ContractEvent) (*L1ContractEvent, error) {
	var l1ContractEvent L1ContractEvent
	result := db.gorm.Where(&filter).Take(&l1ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l1ContractEvent, nil
}

func (db *contractEventsDB) L1ContractEventsWithFilter(filter ContractEvent, fromHeight, toHeight *big.Int) ([]L1ContractEvent, error) {
	if fromHeight == nil {
		fromHeight = big.NewInt(0)
	}
	if toHeight == nil {
		return nil, errors.New("end height unspecified")
	}
	if fromHeight.Cmp(toHeight) > 0 {
		return nil, fmt.Errorf("fromHeight %d is greater than toHeight %d", fromHeight, toHeight)
	}

	query := db.gorm.Table("l1_contract_events").Where(&filter)
	query = query.Joins("INNER JOIN l1_block_headers ON l1_contract_events.block_hash = l1_block_headers.hash")
	query = query.Where("l1_block_headers.number >= ? AND l1_block_headers.number <= ?", fromHeight, toHeight)
	query = query.Order("l1_block_headers.number ASC, l1_contract_events.log_index ASC").Select("l1_contract_events.*")

	// NOTE: We use `Find` here instead of `Scan` since `Scan` doesn't not support
	// model hooks like `ContractEvent#AfterFind`. Functionally they are the same
	var events []L1ContractEvent
	result := query.Find(&events)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return events, nil
}

func (db *contractEventsDB) L1LatestContractEventWithFilter(filter ContractEvent) (*L1ContractEvent, error) {
	var l1ContractEvent L1ContractEvent
	result := db.gorm.Where(&filter).Order("timestamp DESC").Take(&l1ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l1ContractEvent, nil
}

// L2

func (db *contractEventsDB) StoreL2ContractEvents(events []L2ContractEvent) error {
	// Since the block hash refers back to L2, we dont necessarily have to check
	// that the RLP bytes match when doing conflict resolution.
	deduped := db.gorm.Clauses(clause.OnConflict{OnConstraint: "l2_contract_events_block_hash_log_index_key", DoNothing: true})
	result := deduped.Create(&events)
	if result.Error == nil && int(result.RowsAffected) < len(events) {
		db.log.Warn("ignored L2 contract event duplicates", "duplicates", len(events)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *contractEventsDB) L2ContractEvent(uuid uuid.UUID) (*L2ContractEvent, error) {
	return db.L2ContractEventWithFilter(ContractEvent{GUID: uuid})
}

func (db *contractEventsDB) L2ContractEventWithFilter(filter ContractEvent) (*L2ContractEvent, error) {
	var l2ContractEvent L2ContractEvent
	result := db.gorm.Where(&filter).Take(&l2ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l2ContractEvent, nil
}

func (db *contractEventsDB) L2ContractEventsWithFilter(filter ContractEvent, fromHeight, toHeight *big.Int) ([]L2ContractEvent, error) {
	if fromHeight == nil {
		fromHeight = big.NewInt(0)
	}
	if toHeight == nil {
		return nil, errors.New("end height unspecified")
	}
	if fromHeight.Cmp(toHeight) > 0 {
		return nil, fmt.Errorf("fromHeight %d is greater than toHeight %d", fromHeight, toHeight)
	}

	query := db.gorm.Table("l2_contract_events").Where(&filter)
	query = query.Joins("INNER JOIN l2_block_headers ON l2_contract_events.block_hash = l2_block_headers.hash")
	query = query.Where("l2_block_headers.number >= ? AND l2_block_headers.number <= ?", fromHeight, toHeight)
	query = query.Order("l2_block_headers.number ASC, l2_contract_events.log_index ASC").Select("l2_contract_events.*")

	// NOTE: We use `Find` here instead of `Scan` since `Scan` doesn't not support
	// model hooks like `ContractEvent#AfterFind`. Functionally they are the same
	var events []L2ContractEvent
	result := query.Find(&events)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return events, nil
}

func (db *contractEventsDB) L2LatestContractEventWithFilter(filter ContractEvent) (*L2ContractEvent, error) {
	var l2ContractEvent L2ContractEvent
	result := db.gorm.Where(&filter).Order("timestamp DESC").Take(&l2ContractEvent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l2ContractEvent, nil
}

// Auxiliary methods for both L1 and L2

// ContractEventsWithFilter will retrieve contract events within the specified range according to the `chainSelector`.
func (db *contractEventsDB) ContractEventsWithFilter(filter ContractEvent, chainSelector string, fromHeight, toHeight *big.Int) ([]ContractEvent, error) {
	switch chainSelector {
	case "l1":
		l1Events, err := db.L1ContractEventsWithFilter(filter, fromHeight, toHeight)
		if err != nil {
			return nil, err
		}
		events := make([]ContractEvent, len(l1Events))
		for i := range l1Events {
			events[i] = l1Events[i].ContractEvent
		}
		return events, nil
	case "l2":
		l2Events, err := db.L2ContractEventsWithFilter(filter, fromHeight, toHeight)
		if err != nil {
			return nil, err
		}
		events := make([]ContractEvent, len(l2Events))
		for i := range l2Events {
			events[i] = l2Events[i].ContractEvent
		}
		return events, nil
	default:
		return nil, errors.New("expected 'l1' or 'l2' for chain selection")
	}
}
