package types

import (
	"context"
	"math/big"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/merkle"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
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
	concatted = append(concatted, math.U256Bytes(new(big.Int).SetUint64(l.Index))...)
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

// ShouldVerify returns true if the preimage upload is complete, has not yet been countered, and the
// challenge period has not yet elapsed.
func (m LargePreimageMetaData) ShouldVerify(now time.Time, ignoreAfter time.Duration) bool {
	return m.Timestamp > 0 && !m.Countered && m.Timestamp+uint64(ignoreAfter.Seconds()) > uint64(now.Unix())
}

type StateSnapshot [25]uint64

// Pack packs the state in to the solidity ABI encoding required for the state matrix
func (s StateSnapshot) Pack() []byte {
	buf := make([]byte, 0, len(s)*32)
	for _, v := range s {
		buf = append(buf, math.U256Bytes(new(big.Int).SetUint64(v))...)
	}
	return buf
}

type Challenge struct {
	// StateMatrix is the packed state matrix preimage of the StateCommitment in Prestate
	StateMatrix StateSnapshot

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
	GetInputDataBlocks(ctx context.Context, block rpcblock.Block, ident LargePreimageIdent) ([]uint64, error)
	GetProposalTreeRoot(ctx context.Context, block rpcblock.Block, ident LargePreimageIdent) (common.Hash, error)
	DecodeInputData(data []byte) (*big.Int, InputData, error)
	ChallengeTx(ident LargePreimageIdent, challenge Challenge) (txmgr.TxCandidate, error)
	ChallengePeriod(ctx context.Context) (uint64, error)
}
