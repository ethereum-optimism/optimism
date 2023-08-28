package alphabet

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	ErrIndexTooLarge = errors.New("index is larger than the maximum index")
)

// AlphabetTraceProvider is a [TraceProvider] that provides claims for specific
// indices in the given trace.
type AlphabetTraceProvider struct {
	state  []string
	maxLen uint64
}

// NewTraceProvider returns a new [AlphabetProvider].
func NewTraceProvider(state string, depth uint64) *AlphabetTraceProvider {
	return &AlphabetTraceProvider{
		state:  strings.Split(state, ""),
		maxLen: uint64(1 << depth),
	}
}

func (ap *AlphabetTraceProvider) GetStepData(ctx context.Context, i uint64) ([]byte, []byte, *types.PreimageOracleData, error) {
	if i == 0 {
		prestate, err := ap.AbsolutePreState(ctx)
		if err != nil {
			return nil, nil, nil, err
		}
		return prestate, []byte{}, nil, nil
	}
	// We want the pre-state which is the value prior to the one requested
	i--
	// The index cannot be larger than the maximum index as computed by the depth.
	if i >= ap.maxLen {
		return nil, nil, nil, ErrIndexTooLarge
	}
	// We extend the deepest hash to the maximum depth if the trace is not expansive.
	if i >= uint64(len(ap.state)) {
		return ap.GetStepData(ctx, uint64(len(ap.state)))
	}
	return BuildAlphabetPreimage(i, ap.state[i]), []byte{}, nil, nil
}

// Get returns the claim value at the given index in the trace.
func (ap *AlphabetTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	// Step data returns the pre-state, so add 1 to get the state for index i
	claimBytes, _, _, err := ap.GetStepData(ctx, i+1)
	if err != nil {
		return common.Hash{}, err
	}
	return crypto.Keccak256Hash(claimBytes), nil
}

// AbsolutePreState returns the absolute pre-state for the alphabet trace.
func (ap *AlphabetTraceProvider) AbsolutePreState(ctx context.Context) ([]byte, error) {
	return common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000060"), nil
}

// BuildAlphabetPreimage constructs the claim bytes for the index and state item.
func BuildAlphabetPreimage(i uint64, letter string) []byte {
	return append(IndexToBytes(i), LetterToBytes(letter)...)
}

// IndexToBytes converts an index to a byte slice big endian
func IndexToBytes(i uint64) []byte {
	big := new(big.Int)
	big.SetUint64(i)
	out := make([]byte, 32)
	return big.FillBytes(out)
}

// LetterToBytes converts a letter to a 32 byte array
func LetterToBytes(letter string) []byte {
	out := make([]byte, 32)
	out[31] = byte(letter[0])
	return out
}
