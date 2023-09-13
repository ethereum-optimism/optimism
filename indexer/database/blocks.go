package database

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"gorm.io/gorm"
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

type L1BlockHeader struct {
	BlockHeader `gorm:"embedded"`
}

type L2BlockHeader struct {
	BlockHeader `gorm:"embedded"`
}

type BlocksView interface {
	L1BlockHeader(common.Hash) (*L1BlockHeader, error)
	L1BlockHeaderWithFilter(BlockHeader) (*L1BlockHeader, error)
	L1LatestBlockHeader() (*L1BlockHeader, error)

	L2BlockHeader(common.Hash) (*L2BlockHeader, error)
	L2BlockHeaderWithFilter(BlockHeader) (*L2BlockHeader, error)
	L2LatestBlockHeader() (*L2BlockHeader, error)

	LatestEpoch() (*Epoch, error)
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
	gorm *gorm.DB
}

func newBlocksDB(db *gorm.DB) BlocksDB {
	return &blocksDB{gorm: db}
}

// L1

func (db *blocksDB) StoreL1BlockHeaders(headers []L1BlockHeader) error {
	result := db.gorm.CreateInBatches(&headers, batchInsertSize)
	return result.Error
}

func (db *blocksDB) L1BlockHeader(hash common.Hash) (*L1BlockHeader, error) {
	return db.L1BlockHeaderWithFilter(BlockHeader{Hash: hash})
}

func (db *blocksDB) L1BlockHeaderWithFilter(filter BlockHeader) (*L1BlockHeader, error) {
	var l1Header L1BlockHeader
	result := db.gorm.Where(&filter).Take(&l1Header)
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
	result := db.gorm.CreateInBatches(&headers, batchInsertSize)
	return result.Error
}

func (db *blocksDB) L2BlockHeader(hash common.Hash) (*L2BlockHeader, error) {
	return db.L2BlockHeaderWithFilter(BlockHeader{Hash: hash})
}

func (db *blocksDB) L2BlockHeaderWithFilter(filter BlockHeader) (*L2BlockHeader, error) {
	var l2Header L2BlockHeader
	result := db.gorm.Where(&filter).Take(&l2Header)
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

// Auxiliary Methods on both L1 & L2

type Epoch struct {
	L1BlockHeader L1BlockHeader `gorm:"embedded"`
	L2BlockHeader L2BlockHeader `gorm:"embedded"`
}

// LatestEpoch return the latest epoch, seen on  L1 & L2. In other words
// this returns the latest indexed L1 block that has a corresponding
// indexed L2 block with a matching L1Origin (equal timestamps).
//
// For more, see the protocol spec:
//   - https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md
func (db *blocksDB) LatestEpoch() (*Epoch, error) {
	latestL1Header, err := db.L1LatestBlockHeader()
	if err != nil {
		return nil, err
	} else if latestL1Header == nil {
		return nil, nil
	}

	latestL2Header, err := db.L2LatestBlockHeader()
	if err != nil {
		return nil, err
	} else if latestL2Header == nil {
		return nil, nil
	}

	minTime := latestL1Header.Timestamp
	if latestL2Header.Timestamp < minTime {
		minTime = latestL2Header.Timestamp
	}

	// This is a faster query than doing an INNER JOIN between l1_block_headers and l2_block_headers
	// which requires a full table scan to compute the resulting table.
	l1Query := db.gorm.Table("l1_block_headers").Where("timestamp <= ?", minTime)
	l2Query := db.gorm.Table("l2_block_headers").Where("timestamp <= ?", minTime)
	query := db.gorm.Raw(`SELECT * FROM (?) AS l1_block_headers, (?) AS l2_block_headers
		WHERE l1_block_headers.timestamp = l2_block_headers.timestamp
		ORDER BY l2_block_headers.number DESC LIMIT 1`, l1Query, l2Query)

	var epoch Epoch
	result := query.Take(&epoch)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}

	return &epoch, nil
}
