package database

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"gorm.io/gorm"
)

/**
 * Types
 */

type BlockHeader struct {
	Hash       common.Hash `gorm:"primaryKey;serializer:json"`
	ParentHash common.Hash `gorm:"serializer:json"`
	Number     U256
	Timestamp  uint64
}

type L1BlockHeader struct {
	BlockHeader
}

type L2BlockHeader struct {
	BlockHeader

	// Marked when the proposed output is finalized on L1.
	// All bedrock blocks will have `LegacyStateBatchIndex ^== NULL`
	L1BlockHash           *common.Hash `gorm:"serializer:json"`
	LegacyStateBatchIndex *uint64
}

type LegacyStateBatch struct {
	// `default:0` is added since gorm would interepret 0 as NULL
	// violating the primary key constraint.
	Index uint64 `gorm:"primaryKey;default:0"`

	Root        common.Hash `gorm:"serializer:json"`
	Size        uint64
	PrevTotal   uint64
	L1BlockHash common.Hash `gorm:"serializer:json"`
}

type BlocksView interface {
	FinalizedL1BlockHeight() (*big.Int, error)
	FinalizedL2BlockHeight() (*big.Int, error)
}

type BlocksDB interface {
	BlocksView

	StoreL1BlockHeaders([]*L1BlockHeader) error
	StoreLegacyStateBatch(*LegacyStateBatch) error

	StoreL2BlockHeaders([]*L2BlockHeader) error
	MarkFinalizedL1RootForL2Block(common.Hash, common.Hash) error
}

/**
 * Implementation
 */

type blocksDB struct {
	gorm *gorm.DB
}

func newBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}

// L1

func (db *blocksDB) StoreL1BlockHeaders(headers []*L1BlockHeader) error {
	result := db.gorm.Create(&headers)
	return result.Error
}

func (db *blocksDB) StoreLegacyStateBatch(stateBatch *LegacyStateBatch) error {
	// Even though transaction control flow is managed, could we benefit
	// from a nested transaction here?

	result := db.gorm.Create(stateBatch)
	if result.Error != nil {
		return result.Error
	}

	// Mark this state batch index & l1 block hash for all applicable l2 blocks
	l2Headers := make([]*L2BlockHeader, stateBatch.Size)

	// [start, end] range is inclusive. Since `PrevTotal` is the index of the prior batch, no
	// need to subtract one when adding the size
	startHeight := U256{Int: big.NewInt(int64(stateBatch.PrevTotal + 1))}
	endHeight := U256{Int: big.NewInt(int64(stateBatch.PrevTotal + stateBatch.Size))}
	result = db.gorm.Where("number BETWEEN ? AND ?", &startHeight, &endHeight).Find(&l2Headers)
	if result.Error != nil {
		return result.Error
	} else if result.RowsAffected != int64(stateBatch.Size) {
		return errors.New("state batch size exceeds number of indexed l2 blocks")
	}

	for _, header := range l2Headers {
		header.LegacyStateBatchIndex = &stateBatch.Index
		header.L1BlockHash = &stateBatch.L1BlockHash
	}

	result = db.gorm.Save(&l2Headers)
	return result.Error
}

func (db *blocksDB) FinalizedL1BlockHeight() (*big.Int, error) {
	var l1Header L1BlockHeader
	result := db.gorm.Order("number DESC").Take(&l1Header)
	if result.Error != nil {
		return nil, result.Error
	}

	return l1Header.Number.Int, nil
}

// L2

func (db *blocksDB) StoreL2BlockHeaders(headers []*L2BlockHeader) error {
	result := db.gorm.Create(&headers)
	return result.Error
}

func (db *blocksDB) FinalizedL2BlockHeight() (*big.Int, error) {
	var l2Header L2BlockHeader
	result := db.gorm.Order("number DESC").Take(&l2Header)
	if result.Error != nil {
		return nil, result.Error
	}

	result.Logger.Info(context.Background(), "number ", l2Header.Number)
	return l2Header.Number.Int, nil
}

func (db *blocksDB) MarkFinalizedL1RootForL2Block(l2Root, l1Root common.Hash) error {
	var l2Header L2BlockHeader
	l2Header.Hash = l2Root // set the primary key

	result := db.gorm.First(&l2Header)
	if result.Error != nil {
		return result.Error
	}

	l2Header.L1BlockHash = &l1Root
	result = db.gorm.Save(&l2Header)
	return result.Error
}
