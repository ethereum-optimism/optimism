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
	Hash       common.Hash `gorm:"primaryKey;serializer:json"`
	ParentHash common.Hash `gorm:"serializer:json"`
	Number     U256
	Timestamp  uint64
}

func BlockHeaderFromGethHeader(header *types.Header) BlockHeader {
	return BlockHeader{
		Hash:       header.Hash(),
		ParentHash: header.ParentHash,
		Number:     U256{Int: header.Number},
		Timestamp:  header.Time,
	}
}

type L1BlockHeader struct {
	BlockHeader
}

type L2BlockHeader struct {
	BlockHeader
}

type LegacyStateBatch struct {
	// `default:0` is added since gorm would interepret 0 as NULL
	// violating the primary key constraint.
	Index uint64 `gorm:"primaryKey;default:0"`

	Root                common.Hash `gorm:"serializer:json"`
	Size                uint64
	PrevTotal           uint64
	L1ContractEventGUID uuid.UUID
}

type OutputProposal struct {
	OutputRoot    common.Hash `gorm:"primaryKey;serializer:json"`
	L2OutputIndex U256
	L2BlockNumber U256

	L1ContractEventGUID uuid.UUID
}

type BlocksView interface {
	L1BlockHeader(*big.Int) (*L1BlockHeader, error)
	LatestL1BlockHeader() (*L1BlockHeader, error)

	LatestCheckpointedOutput() (*OutputProposal, error)
	OutputProposal(index *big.Int) (*OutputProposal, error)

	L2BlockHeader(*big.Int) (*L2BlockHeader, error)
	LatestL2BlockHeader() (*L2BlockHeader, error)
}

type BlocksDB interface {
	BlocksView

	StoreL1BlockHeaders([]*L1BlockHeader) error
	StoreL2BlockHeaders([]*L2BlockHeader) error

	StoreLegacyStateBatches([]*LegacyStateBatch) error
	StoreOutputProposals([]*OutputProposal) error
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

func (db *blocksDB) StoreLegacyStateBatches(stateBatches []*LegacyStateBatch) error {
	result := db.gorm.Create(stateBatches)
	return result.Error
}

func (db *blocksDB) StoreOutputProposals(outputs []*OutputProposal) error {
	result := db.gorm.Create(outputs)
	return result.Error
}

func (db *blocksDB) L1BlockHeader(height *big.Int) (*L1BlockHeader, error) {
	var l1Header L1BlockHeader
	result := db.gorm.Where(&BlockHeader{Number: U256{Int: height}}).Take(&l1Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l1Header, nil
}

func (db *blocksDB) LatestL1BlockHeader() (*L1BlockHeader, error) {
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

func (db *blocksDB) StoreL2BlockHeaders(headers []*L2BlockHeader) error {
	result := db.gorm.Create(&headers)
	return result.Error
}

func (db *blocksDB) L2BlockHeader(height *big.Int) (*L2BlockHeader, error) {
	var l2Header L2BlockHeader
	result := db.gorm.Where(&BlockHeader{Number: U256{Int: height}}).Take(&l2Header)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, result.Error
	}

	return &l2Header, nil
}

func (db *blocksDB) LatestL2BlockHeader() (*L2BlockHeader, error) {
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
