package types

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum/go-ethereum/common"
)

// BlockSize is the size in bytes required for leaf data.
const BlockSize = 136

// Leaf is the keccak state matrix added to the large preimage merkle tree.
type Leaf struct {
	// Input is the data absorbed for the block, exactly 136 bytes
	Input [BlockSize]byte
	// Index of the block in the absorption process
	Index *big.Int
	// StateCommitment is the hash of the internal state after absorbing the input.
	StateCommitment common.Hash
}

// InputData is a contiguous segment of preimage data.
type InputData struct {
	// Input is the preimage data.
	// When Finalize is false, len(Input) must equal len(Commitments)*BlockSize
	// When Finalize is true, len(Input) must be between len(Commitments - 1)*BlockSize and len(Commitments)*BlockSize
	Input []byte
	// Commitments are the keccak commitments for each leaf in the chunk.
	Commitments []common.Hash
	// Finalize indicates whether the chunk is the final chunk.
	Finalize bool
}

type LargePreimageIdent struct {
	Claimant common.Address
	UUID     *big.Int
}

type LargePreimageMetaData struct {
	LargePreimageIdent

	// Timestamp is the time at which the proposal first became fully available.
	// 0 when not all data is available yet
	Timestamp       uint64
	PartOffset      uint32
	ClaimedSize     uint32
	BlocksProcessed uint32
	BytesProcessed  uint32
	Countered       bool
}

// ShouldVerify returns true if the preimage upload is complete and has not yet been countered.
// Note that the challenge period for the preimage may have expired but the image not yet been finalized.
func (m LargePreimageMetaData) ShouldVerify() bool {
	return m.Timestamp > 0 && !m.Countered
}

type LargePreimageOracle interface {
	Addr() common.Address
	GetActivePreimages(ctx context.Context, blockHash common.Hash) ([]LargePreimageMetaData, error)
	GetInputDataBlocks(ctx context.Context, block batching.Block, ident LargePreimageIdent) ([]uint64, error)
	DecodeInputData(data []byte) (*big.Int, InputData, error)
}
