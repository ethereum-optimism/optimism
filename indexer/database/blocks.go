package database

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

/**
 * Types
 */

type BlockHeader struct {
	Hash       common.Hash `gorm:"primaryKey;serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     *big.Int    `gorm:"serializer:u256"`
	Timestamp  uint64

	RLPHeader *RLPHeader `gorm:"serializer:rlp;column:rlp_bytes"`
}

func BlockHeaderFromHeader(header *types.Header) BlockHeader {
	return BlockHeader{
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Number:     header.Number,
		Timestamp:  header.Time,

		RLPHeader: (*RLPHeader)(header),
	}
}

func (b BlockHeader) String() string {
	return fmt.Sprintf("{Hash: %s, Number: %s}", b.Hash, b.Number)
}

type L1BlockHeader struct {
	BlockHeader `gorm:"embedded"`
}

type L2BlockHeader struct {
	BlockHeader `gorm:"embedded"`
}

type BlocksView interface {
	L1BlockHeader(common.Hash) (*L1BlockHeader, error)
	L1BlockHeaderWithFilter(BlockHeader) (*L1BlockHeader, error)
	L1BlockHeaderWithScope(func(db *gorm.DB) *gorm.DB) (*L1BlockHeader, error)
	L1LatestBlockHeader() (*L1BlockHeader, error)

	L2BlockHeader(common.Hash) (*L2BlockHeader, error)
	L2BlockHeaderWithFilter(BlockHeader) (*L2BlockHeader, error)
	L2BlockHeaderWithScope(func(db *gorm.DB) *gorm.DB) (*L2BlockHeader, error)
	L2LatestBlockHeader() (*L2BlockHeader, error)
}

type BlocksDB interface {
	BlocksView

	StoreL1BlockHeaders([]L1BlockHeader) error
	StoreL2BlockHeaders([]L2BlockHeader) error
}

/**
 * Implementation
 */

type blocksDB struct {
	log  log.Logger
	gorm *gorm.DB
}

func newBlocksDB(log log.Logger, db *gorm.DB) BlocksDB {
	return &blocksDB{log: log.New("table", "blocks"), gorm: db}
}

// L1

func (db *blocksDB) StoreL1BlockHeaders(headers []L1BlockHeader) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "hash"}}, DoNothing: true})
	result := deduped.Create(&headers)
	if result.Error == nil && int(result.RowsAffected) < len(headers) {
		db.log.Warn("ignored L1 block duplicates", "duplicates", len(headers)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *blocksDB) L1BlockHeader(hash common.Hash) (*L1BlockHeader, error) {
	return db.L1BlockHeaderWithFilter(BlockHeader{Hash: hash})
}

func (db *blocksDB) L1BlockHeaderWithFilter(filter BlockHeader) (*L1BlockHeader, error) {
	return db.L1BlockHeaderWithScope(func(gorm *gorm.DB) *gorm.DB { return gorm.Where(&filter) })
}

func (db *blocksDB) L1BlockHeaderWithScope(scope func(*gorm.DB) *gorm.DB) (*L1BlockHeader, error) {
	var l1Header L1BlockHeader
	result := db.gorm.Scopes(scope).Take(&l1Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l1Header, nil
}

func (db *blocksDB) L1LatestBlockHeader() (*L1BlockHeader, error) {
	var l1Header L1BlockHeader
	result := db.gorm.Order("number DESC").Take(&l1Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l1Header, nil
}

// L2

func (db *blocksDB) StoreL2BlockHeaders(headers []L2BlockHeader) error {
	deduped := db.gorm.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "hash"}}, DoNothing: true})
	result := deduped.Create(&headers)
	if result.Error == nil && int(result.RowsAffected) < len(headers) {
		db.log.Warn("ignored L2 block duplicates", "duplicates", len(headers)-int(result.RowsAffected))
	}

	return result.Error
}

func (db *blocksDB) L2BlockHeader(hash common.Hash) (*L2BlockHeader, error) {
	return db.L2BlockHeaderWithFilter(BlockHeader{Hash: hash})
}

func (db *blocksDB) L2BlockHeaderWithFilter(filter BlockHeader) (*L2BlockHeader, error) {
	return db.L2BlockHeaderWithScope(func(gorm *gorm.DB) *gorm.DB { return gorm.Where(&filter) })
}

func (db *blocksDB) L2BlockHeaderWithScope(scope func(*gorm.DB) *gorm.DB) (*L2BlockHeader, error) {
	var l2Header L2BlockHeader
	result := db.gorm.Scopes(scope).Take(&l2Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l2Header, nil
}

func (db *blocksDB) L2LatestBlockHeader() (*L2BlockHeader, error) {
	var l2Header L2BlockHeader
	result := db.gorm.Order("number DESC").Take(&l2Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &l2Header, nil
}
