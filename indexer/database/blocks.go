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

	LatestObservedEpoch(*big.Int, uint64) (*Epoch, error)
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

// LatestObservedEpoch return the marker for latest epoch, observed on  L1 & L2, within
// the specified bounds. In other words this returns the latest indexed L1 block that has
// a corresponding indexed L2 block with a matching L1Origin (equal timestamps).
//
// If `fromL1Height` (inclusive) is not specified, the search will start from genesis and
// continue all the way to latest indexed heights if `maxL1Range == 0`.
//
// For more, see the protocol spec:
//   - https://github.com/ethereum-optimism/optimism/blob/develop/specs/derivation.md
func (db *blocksDB) LatestObservedEpoch(fromL1Height *big.Int, maxL1Range uint64) (*Epoch, error) {
	// We use timestamps since that translates to both L1 & L2
	var fromTimestamp, toTimestamp uint64

	// Lower Bound (the default `fromTimestamp = l1_starting_height` (default=0) suffices genesis representation)
	var header L1BlockHeader
	if fromL1Height != nil {
		result := db.gorm.Where("number = ?", fromL1Height).Take(&header)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, result.Error
		}

		fromTimestamp = header.Timestamp
	} else {
		// Take the lowest indexed L1 block to compute the lower bound
		result := db.gorm.Order("number ASC").Take(&header)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, result.Error
		}

		fromL1Height = header.Number
		fromTimestamp = header.Timestamp
	}

	// Upper Bound (lowest timestamp indexed between L1/L2 bounded by `maxL1Range`)
	{
		l1QueryFilter := fmt.Sprintf("timestamp >= %d", fromTimestamp)
		if maxL1Range > 0 {
			maxHeight := new(big.Int).Add(fromL1Height, big.NewInt(int64(maxL1Range)))
			l1QueryFilter = fmt.Sprintf("%s AND number <= %d", l1QueryFilter, maxHeight)
		}

		// Fetch most recent header from l1_block_headers table
		var l1Header L1BlockHeader
		result := db.gorm.Where(l1QueryFilter).Order("timestamp DESC").Take(&l1Header)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, result.Error
		}

		toTimestamp = l1Header.Timestamp

		// Fetch most recent header from l2_block_headers table
		var l2Header L2BlockHeader
		result = db.gorm.Where("timestamp <= ?", toTimestamp).Order("timestamp DESC").Take(&l2Header)
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, result.Error
		}

		if l2Header.Timestamp < toTimestamp {
			toTimestamp = l2Header.Timestamp
		}
	}

	// Search for the latest indexed epoch within range. This is a faster query than doing an INNER JOIN between
	// l1_block_headers and l2_block_headers which requires a full table scan to compute the resulting table.
	l1Query := db.gorm.Table("l1_block_headers").Where("timestamp >= ? AND timestamp <= ?", fromTimestamp, toTimestamp)
	l2Query := db.gorm.Table("l2_block_headers").Where("timestamp >= ? AND timestamp <= ?", fromTimestamp, toTimestamp)
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
