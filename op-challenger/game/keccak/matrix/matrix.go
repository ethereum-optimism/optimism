package matrix

import (
	"errors"
	"io"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// StateMatrix implements a stateful keccak sponge with the ability to create state commitments after each permutation
type StateMatrix struct {
	s *state
}

var (
	ErrInvalidMaxLen = errors.New("invalid max length to absorb")
	uint256Size      = 32
)

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

func (d *StateMatrix) AbsorbUpTo(in io.Reader, maxLen int) (types.InputData, error) {
	if maxLen < types.BlockSize || maxLen%types.BlockSize != 0 {
		return types.InputData{}, ErrInvalidMaxLen
	}
	input := make([]byte, 0, maxLen)
	commitments := make([]common.Hash, 0, maxLen/types.BlockSize)
	for len(input)+types.BlockSize <= maxLen {
		readData, err := d.absorbNextLeafInput(in)
		if errors.Is(err, io.EOF) {
			input = append(input, readData...)
			commitments = append(commitments, d.StateCommitment())
			return types.InputData{
				Input:       input,
				Commitments: commitments,
				Finalize:    true,
			}, io.EOF
		} else if err != nil {
			return types.InputData{}, err
		}
		input = append(input, readData...)
		commitments = append(commitments, d.StateCommitment())
	}

	return types.InputData{
		Input:       input,
		Commitments: commitments,
		Finalize:    false,
	}, nil
}

// absorbNextLeafInput reads up to [BlockSize] bytes from in and absorbs them into the state matrix.
// If EOF is reached while reading, the state matrix is finalized and [io.EOF] is returned.
func (d *StateMatrix) absorbNextLeafInput(in io.Reader) ([]byte, error) {
	data := make([]byte, types.BlockSize)
	read := 0
	final := false
	for read < types.BlockSize {
		n, err := in.Read(data[read:])
		if errors.Is(err, io.EOF) {
			final = true
			break
		} else if err != nil {
			return nil, err
		}
		read += n
	}
	input := data[:read]
	d.absorbLeafInput(input, final)
	if final {
		return input, io.EOF
	}
	return input, nil
}

// absorbLeafInput absorbs the specified data into the keccak sponge.
// If final is true, the data is padded to the required length, otherwise it must be exactly [types.BlockSize] bytes.
func (d *StateMatrix) absorbLeafInput(data []byte, final bool) {
	if !final && len(data) != types.BlockSize {
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
