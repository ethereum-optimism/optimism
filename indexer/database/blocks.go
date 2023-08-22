package database

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/google/uuid"

	"gorm.io/gorm"
)

/**
 * Types
 */

type BlockHeader struct {
	Hash       common.Hash `gorm:"primaryKey;serializer:bytes"`
	ParentHash common.Hash `gorm:"serializer:bytes"`
	Number     U256
	Timestamp  uint64

	RLPHeader *RLPHeader `gorm:"serializer:rlp;column:rlp_bytes"`
}

func BlockHeaderFromHeader(header *types.Header) BlockHeader {
	return BlockHeader{
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Number:     U256{Int: header.Number},
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

type LegacyStateBatch struct {
	// `default:0` is added since gorm would interepret 0 as NULL
	// violating the primary key constraint.
	Index uint64 `gorm:"primaryKey;default:0"`

	Root                common.Hash `gorm:"serializer:bytes"`
	Size                uint64
	PrevTotal           uint64
	L1ContractEventGUID uuid.UUID
}

type OutputProposal struct {
	OutputRoot    common.Hash `gorm:"primaryKey;serializer:bytes"`
	L2OutputIndex U256
	L2BlockNumber U256

	L1ContractEventGUID uuid.UUID
}

type BlocksView interface {
	L1BlockHeader(common.Hash) (*L1BlockHeader, error)
	L1BlockHeaderWithFilter(BlockHeader) (*L1BlockHeader, error)
	L1LatestBlockHeader() (*L1BlockHeader, error)

	L2BlockHeader(common.Hash) (*L2BlockHeader, error)
	L2BlockHeaderWithFilter(BlockHeader) (*L2BlockHeader, error)
	L2LatestBlockHeader() (*L2BlockHeader, error)

	LatestCheckpointedOutput() (*OutputProposal, error)
	OutputProposal(index *big.Int) (*OutputProposal, error)

	LatestEpoch() (*Epoch, error)
}

type BlocksDB interface {
	BlocksView

	StoreL1BlockHeaders([]L1BlockHeader) error
	StoreL2BlockHeaders([]L2BlockHeader) error

	StoreLegacyStateBatches([]LegacyStateBatch) error
	StoreOutputProposals([]OutputProposal) error
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
	result := db.gorm.Create(&headers)
	return result.Error
}

func (db *blocksDB) StoreLegacyStateBatches(stateBatches []LegacyStateBatch) error {
	result := db.gorm.Create(stateBatches)
	return result.Error
}

func (db *blocksDB) StoreOutputProposals(outputs []OutputProposal) error {
	result := db.gorm.Create(outputs)
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

func (db *blocksDB) LatestCheckpointedOutput() (*OutputProposal, error) {
	var outputProposal OutputProposal
	result := db.gorm.Order("l2_output_index DESC").Take(&outputProposal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &outputProposal, nil
}

func (db *blocksDB) OutputProposal(index *big.Int) (*OutputProposal, error) {
	var outputProposal OutputProposal
	result := db.gorm.Where(&OutputProposal{L2OutputIndex: U256{Int: index}}).Take(&outputProposal)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &outputProposal, nil
}

// L2

func (db *blocksDB) StoreL2BlockHeaders(headers []L2BlockHeader) error {
	result := db.gorm.Create(&headers)
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

	result.Logger.Info(context.Background(), "number ", l2Header.Number)
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
	// Since L1 blocks occur less frequently than L2, we do a INNER JOIN from L1 on
	// L2 for a faster query. Per the protocol, the L2 block that starts a new epoch
	// will have a matching timestamp with the L1 origin.
	query := db.gorm.Table("l1_block_headers").Order("l1_block_headers.timestamp DESC")
	query = query.Joins("INNER JOIN l2_block_headers ON l1_block_headers.timestamp = l2_block_headers.timestamp")
	query = query.Select("*")

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
