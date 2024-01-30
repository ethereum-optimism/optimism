package types

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// BlockSize is the size in bytes required for leaf data.
const BlockSize = 136

// Leaf is the keccak state matrix added to the large preimage merkle tree.
type Leaf struct {
	// Input is the data absorbed for the block, exactly 136 bytes
	Input [BlockSize]byte
	// Index of the block in the absorption process
	Index uint64
	// StateCommitment is the hash of the internal state after absorbing the input.
	StateCommitment common.Hash
}

// Hash returns the hash of the leaf data. That is the
// bytewise concatenation of the input, index, and state commitment.
func (l Leaf) Hash() common.Hash {
	concatted := make([]byte, 0, 136+32+32)
	concatted = append(concatted, l.Input[:]...)
	concatted = append(concatted, new(big.Int).SetUint64(l.Index).Bytes()...)
	concatted = append(concatted, l.StateCommitment.Bytes()...)
	return crypto.Keccak256Hash(concatted)
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

type Challenge struct {
	// StateMatrix is the packed state matrix preimage of the StateCommitment in Prestate
	StateMatrix []byte // TODO(client-pod#480): Need a better representation of this

	// Prestate is the valid leaf immediately prior to the first invalid leaf
	Prestate      Leaf
	PrestateProof merkle.Proof

	// Poststate is the first invalid leaf in the preimage. The challenge claims that this leaf is invalid.
	Poststate      Leaf
	PoststateProof merkle.Proof
}

type LargePreimageOracle interface {
	Addr() common.Address
	GetActivePreimages(ctx context.Context, blockHash common.Hash) ([]LargePreimageMetaData, error)
	GetInputDataBlocks(ctx context.Context, block batching.Block, ident LargePreimageIdent) ([]uint64, error)
	DecodeInputData(data []byte) (*big.Int, InputData, error)
	ChallengeTx(ident LargePreimageIdent, challenge Challenge) (txmgr.TxCandidate, error)
}
