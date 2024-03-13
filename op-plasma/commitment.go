package plasma

import (
	"bytes"
	"errors"

	"github.com/ethereum/go-ethereum/crypto"
)

// ErrInvalidCommitment is returned when the commitment cannot be parsed into a known commitment type.
var ErrInvalidCommitment = errors.New("invalid commitment")

// ErrCommitmentMismatch is returned when the commitment does not match the given input.
var ErrCommitmentMismatch = errors.New("commitment mismatch")

// CommitmentType is the commitment type prefix.
type CommitmentType byte

// KeccakCommitmentType is the default commitment type for the DA storage.
const Keccak256CommitmentType CommitmentType = 0

// Keccak256Commitment is the default commitment type for op-plasma.
type Keccak256Commitment []byte

// Encode adds a commitment type prefix self describing the commitment.
func (c Keccak256Commitment) Encode() []byte {
	return append([]byte{byte(Keccak256CommitmentType)}, c...)
}

// TxData adds an extra version byte to signal it's a commitment.
func (c Keccak256Commitment) TxData() []byte {
	return append([]byte{TxDataVersion1}, c.Encode()...)
}

// Verify checks if the commitment matches the given input.
func (c Keccak256Commitment) Verify(input []byte) error {
	if !bytes.Equal(c, crypto.Keccak256(input)) {
		return ErrCommitmentMismatch
	}
	return nil
}

// Keccak256 creates a new commitment from the given input.
func Keccak256(input []byte) Keccak256Commitment {
	return Keccak256Commitment(crypto.Keccak256(input))
}

// DecodeKeccak256 validates and casts the commitment into a Keccak256Commitment.
func DecodeKeccak256(commitment []byte) (Keccak256Commitment, error) {
	if len(commitment) == 0 {
		return nil, ErrInvalidCommitment
	}
	if commitment[0] != byte(Keccak256CommitmentType) {
		return nil, ErrInvalidCommitment
	}
	c := commitment[1:]
	if len(c) != 32 {
		return nil, ErrInvalidCommitment
	}
	return c, nil
}
