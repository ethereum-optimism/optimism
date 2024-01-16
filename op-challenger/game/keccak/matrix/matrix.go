package matrix

import (
	"errors"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// StateMatrix implements a stateful keccak sponge with the ability to create state commitments after each permutation
type StateMatrix struct {
	s *state
}

// LeafSize is the size in bytes required for leaf data.
const LeafSize = 136

var uint256Size = 32

// NewStateMatrix creates a new state matrix initialized with the initial, zero keccak block.
func NewStateMatrix() *StateMatrix {
	return &StateMatrix{s: newLegacyKeccak256()}
}

// StateCommitment returns the state commitment for the current state matrix.
// Additional data may be absorbed after calling this method.
func (d *StateMatrix) StateCommitment() common.Hash {
	buf := d.PackState()
	return crypto.Keccak256Hash(buf)
}

// PackState packs the state in to the solidity ABI encoding required for the state matrix
func (d *StateMatrix) PackState() []byte {
	buf := make([]byte, 0, len(d.s.a)*uint256Size)
	for _, v := range d.s.a {
		buf = append(buf, math.U256Bytes(new(big.Int).SetUint64(v))...)
	}
	return buf
}

// AbsorbNextLeaf reads up to [LeafSize] bytes from in and absorbs them into the state matrix.
// If EOF is reached while reading, the state matrix is finalized and [io.EOF] is returned.
func (d *StateMatrix) AbsorbNextLeaf(in io.Reader) error {
	data := make([]byte, LeafSize)
	read := 0
	final := false
	for read < LeafSize {
		n, err := in.Read(data[read:])
		if errors.Is(err, io.EOF) {
			final = true
			break
		} else if err != nil {
			return err
		}
		read += n
	}
	d.AbsorbLeaf(data[:read], final)
	if final {
		return io.EOF
	}
	return nil
}

// AbsorbLeaf absorbs the specified data into the keccak sponge.
// If final is true, the data is padded to the required length, otherwise it must be exactly
// LeafSize bytes.
func (d *StateMatrix) AbsorbLeaf(data []byte, final bool) {
	if !final && len(data) != LeafSize {
		panic("sha3: Incorrect leaf data length")
	}
	_, _ = d.s.Write(data[:])
	if final {
		d.s.padAndPermute(d.s.dsbyte)
	}
}

// Hash finalizes the keccak permutation and returns the final hash.
// No further leaves can be absorbed after this is called
func (d *StateMatrix) Hash() (h common.Hash) {
	_, _ = d.s.Read(h[:])
	return h
}
