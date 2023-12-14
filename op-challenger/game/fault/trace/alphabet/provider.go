package alphabet

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	L2ClaimBlockNumberLocalIndex = 4
)

var (
	ErrIndexTooLarge = errors.New("index is larger than the maximum index")
)

// AlphabetTraceProvider is a [TraceProvider] that provides claims for specific
// indices in the given trace.
type AlphabetTraceProvider struct {
	AlphabetPrestateProvider
	state  []string
	depth  uint64
	maxLen uint64
}

// NewTraceProvider returns a new [AlphabetProvider].
func NewTraceProvider(state string, depth uint64) *AlphabetTraceProvider {
	return &AlphabetTraceProvider{
		state:  strings.Split(state, ""),
		depth:  depth,
		maxLen: uint64(1 << depth),
	}
}

func (ap *AlphabetTraceProvider) GetStepData(ctx context.Context, i types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	traceIndex := i.TraceIndex(int(ap.depth))
	if traceIndex.Cmp(common.Big0) == 0 {
		return absolutePrestate, []byte{}, nil, nil
	}
	// We want the pre-state which is the value prior to the one requested
	traceIndex = traceIndex.Sub(traceIndex, big.NewInt(1))
	// The index cannot be larger than the maximum index as computed by the depth.
	if traceIndex.Cmp(big.NewInt(int64(ap.maxLen))) >= 0 {
		return nil, nil, nil, fmt.Errorf("%w traceIndex: %v max: %v pos: %v", ErrIndexTooLarge, traceIndex, ap.maxLen, i)
	}
	// We extend the deepest hash to the maximum depth if the trace is not expansive.
	if traceIndex.Cmp(big.NewInt(int64(len(ap.state)))) >= 0 {
		return ap.GetStepData(ctx, types.NewPosition(int(ap.depth), big.NewInt(int64(len(ap.state)))))
	}
	key := preimage.LocalIndexKey(L2ClaimBlockNumberLocalIndex).PreimageKey()
	preimageData := types.NewPreimageOracleData(key[:], nil, 0)
	return BuildAlphabetPreimage(traceIndex, ap.state[traceIndex.Uint64()]), []byte{}, preimageData, nil
}

// Get returns the claim value at the given index in the trace.
func (ap *AlphabetTraceProvider) Get(ctx context.Context, i types.Position) (common.Hash, error) {
	if uint64(i.Depth()) > ap.depth {
		return common.Hash{}, fmt.Errorf("%w depth: %v max: %v", ErrIndexTooLarge, i.Depth(), ap.depth)
	}
	// Step data returns the pre-state, so add 1 to get the state for index i
	ti := i.TraceIndex(int(ap.depth))
	postPosition := types.NewPosition(int(ap.depth), new(big.Int).Add(ti, big.NewInt(1)))
	claimBytes, _, _, err := ap.GetStepData(ctx, postPosition)
	if err != nil {
		return common.Hash{}, err
	}
	return alphabetStateHash(claimBytes), nil
}

// BuildAlphabetPreimage constructs the claim bytes for the index and state item.
func BuildAlphabetPreimage(i *big.Int, letter string) []byte {
	return append(i.FillBytes(make([]byte, 32)), LetterToBytes(letter)...)
}

func alphabetStateHash(state []byte) common.Hash {
	h := crypto.Keccak256Hash(state)
	h[0] = mipsevm.VMStatusInvalid
	return h
}

// LetterToBytes converts a letter to a 32 byte array
func LetterToBytes(letter string) []byte {
	out := make([]byte, 32)
	out[31] = byte(letter[0])
	return out
}
