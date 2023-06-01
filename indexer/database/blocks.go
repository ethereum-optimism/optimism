package database

import (
	"database/sql"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

/**
 * Types
 */

type BlockHeader struct {
	Hash       common.Hash    `gorm:"primaryKey"`
	ParentHash common.Hash    `gorm:"unique"`
	Number     pgtype.Numeric `gorm:"unique"`
	Timestamp  uint64
}

type L1BlockHeader struct {
	*BlockHeader
}

type L2BlockHeader struct {
	*BlockHeader

	// Marked when the proposed output is finalized on L1.
	// All bedrock blocks will have `LegacyStateBatchIndex == NULL`
	L1BlockHash           *common.Hash
	LegacyStateBatchIndex sql.NullInt64
}

type LegacyStateBatch struct {
	Index       uint64      `gorm:"primaryKey"`
	Root        common.Hash `gorm:"unique"`
	Size        uint64
	PrevTotal   uint64
	L1BlockHash common.Hash
}

type BlocksView interface {
	LatestL1BlockHeight() (*big.Int, error)
	LatestL2BlockHeight() (*big.Int, error)
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
	// Event though transaction control flow is managed, could we benefit
	// from a nested transaction here?

	result := db.gorm.Create(stateBatch)
	if result.Error != nil {
		return result.Error
	}

	// Mark this index & l1 block hash for all applicable l2 blocks
	l2Headers := make([]L2BlockHeader, stateBatch.Size)

	// [start, end] range is inclusive. Since `PrevTotal` is the index of the prior batch, no
	// need to substract one when adding the size
	startHeight := pgtype.Numeric{Int: big.NewInt(int64(stateBatch.PrevTotal + 1)), Status: pgtype.Present}
	endHeight := pgtype.Numeric{Int: big.NewInt(int64(stateBatch.PrevTotal + stateBatch.Size)), Status: pgtype.Present}
	result = db.gorm.Where("number BETWEEN ? AND ?", &startHeight, &endHeight).Find(&l2Headers)
	if result.Error != nil {
		return result.Error
	}

	for _, header := range l2Headers {
		header.LegacyStateBatchIndex = sql.NullInt64{Int64: int64(stateBatch.Index), Valid: true}
		header.L1BlockHash = &stateBatch.L1BlockHash
	}

	result = db.gorm.Save(&l2Headers)
	return result.Error
}

func (db *blocksDB) LatestL1BlockHeight() (*big.Int, error) {
	var latestHeader L1BlockHeader
	result := db.gorm.Order("number desc").First(&latestHeader)
	if result.Error != nil {
		return nil, result.Error
	}

	return latestHeader.Number.Int, nil
}

// L2

func (db *blocksDB) StoreL2BlockHeaders(headers []*L2BlockHeader) error {
	result := db.gorm.Create(&headers)
	return result.Error
}

func (db *blocksDB) LatestL2BlockHeight() (*big.Int, error) {
	var latestHeader L2BlockHeader
	result := db.gorm.Order("number desc").First(&latestHeader)
	if result.Error != nil {
		return nil, result.Error
	}

	return latestHeader.Number.Int, nil
}

func (db *blocksDB) MarkFinalizedL1RootForL2Block(l2Root, l1Root common.Hash) error {
	var l2Header L2BlockHeader
	result := db.gorm.First(&l2Header, "hash = ?", l2Root)
	if result.Error == nil {
		l2Header.L1BlockHash = &l1Root
		db.gorm.Save(&l2Header)
	}

	return result.Error
}
