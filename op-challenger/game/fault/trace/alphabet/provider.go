package alphabet

import (
	"context"
	"errors"
	"fmt"
	"math/big"

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
	startingBlockNumber *big.Int
	depth               types.Depth
	maxLen              uint64
}

// NewTraceProvider returns a new [AlphabetProvider].
func NewTraceProvider(startingBlockNumber *big.Int, depth types.Depth) *AlphabetTraceProvider {
	return &AlphabetTraceProvider{
		startingBlockNumber: startingBlockNumber,
		depth:               depth,
		maxLen:              1 << depth,
	}
}

func (ap *AlphabetTraceProvider) GetStepData(ctx context.Context, i types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	traceIndex := i.TraceIndex(ap.depth)
	key := preimage.LocalIndexKey(L2ClaimBlockNumberLocalIndex).PreimageKey()
	preimageData := types.NewPreimageOracleData(key[:], ap.startingBlockNumber.Bytes(), 0)
	if traceIndex.Cmp(common.Big0) == 0 {
		return absolutePrestate, []byte{}, preimageData, nil
	}
	// We want the pre-state which is the value prior to the one requested
	prestateTraceIndex := traceIndex.Sub(traceIndex, big.NewInt(1))
	// The index cannot be larger than the maximum index as computed by the depth.
	if prestateTraceIndex.Cmp(big.NewInt(int64(ap.maxLen))) >= 0 {
		return nil, nil, nil, fmt.Errorf("%w traceIndex: %v max: %v pos: %v", ErrIndexTooLarge, prestateTraceIndex, ap.maxLen, i)
	}
	claim := new(big.Int).Add(absolutePrestateHash.Big(), prestateTraceIndex)
	return BuildAlphabetPreimage(prestateTraceIndex, claim), []byte{}, preimageData, nil
}

// Get returns the claim value at the given index in the trace.
func (ap *AlphabetTraceProvider) Get(ctx context.Context, i types.Position) (common.Hash, error) {
	if i.Depth() > ap.depth {
		return common.Hash{}, fmt.Errorf("%w depth: %v max: %v", ErrIndexTooLarge, i.Depth(), ap.depth)
	}
	// Step data returns the pre-state, so add 1 to get the state for index i
	ti := i.TraceIndex(ap.depth)
	postPosition := types.NewPosition(ap.depth, new(big.Int).Add(ti, big.NewInt(1)))
	claimBytes, _, _, err := ap.GetStepData(ctx, postPosition)
	if err != nil {
		return common.Hash{}, err
	}
	return alphabetStateHash(claimBytes), nil
}

// BuildAlphabetPreimage constructs the claim bytes for the index and claim.
func BuildAlphabetPreimage(i *big.Int, blockNumber *big.Int) []byte {
	return append(i.FillBytes(make([]byte, 32)), blockNumber.Bytes()...)
}

func alphabetStateHash(state []byte) common.Hash {
	h := crypto.Keccak256Hash(state)
	h[0] = mipsevm.VMStatusInvalid
	return h
}
