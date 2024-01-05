package alphabet

import (
	"bytes"
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

var _ types.TraceProvider = (*AlphabetTraceProvider)(nil)

// AlphabetTraceProvider is a [TraceProvider] that monotonically increments
// the starting l2 block number as the claim value.
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

func (ap *AlphabetTraceProvider) GetStepData(ctx context.Context, pos types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	posIndex := pos.TraceIndex(ap.depth)
	key := preimage.LocalIndexKey(L2ClaimBlockNumberLocalIndex).PreimageKey()
	preimageData := types.NewPreimageOracleData(key[:], ap.startingBlockNumber.Bytes(), 0)
	if posIndex.Cmp(common.Big0) == 0 {
		return absolutePrestate, []byte{}, preimageData, nil
	}
	// We want the pre-state which is the value prior to the one requested
	prestateTraceIndex := new(big.Int).Sub(posIndex, big.NewInt(1))
	if prestateTraceIndex.Cmp(new(big.Int).SetUint64(ap.maxLen)) >= 0 {
		return nil, nil, nil, fmt.Errorf("%w depth: %v index: %v max: %v", ErrIndexTooLarge, ap.depth, posIndex, ap.maxLen)
	}
	// First step expands the absolute preimage to its full form. Weird but it's how AlphabetVM works.
	claim := ap.step(absolutePrestate)
	for i := big.NewInt(0); i.Cmp(prestateTraceIndex) <= 0; i = i.Add(i, big.NewInt(1)) {
		claim = ap.step(claim)
	}
	return claim, []byte{}, preimageData, nil
}

// step accepts the trace index and claim and returns the stepped trace index and claim.
func (ap *AlphabetTraceProvider) step(stateData []byte) []byte {
	// Decode the stateData into the trace index and claim
	traceIndex := new(big.Int).SetBytes(stateData[:32])
	claim := stateData[32:]
	if bytes.Equal(stateData, absolutePrestate) {
		initTraceIndex := new(big.Int).Lsh(ap.startingBlockNumber, 4)
		initClaim := new(big.Int).Add(absolutePrestateInt, initTraceIndex)
		return BuildAlphabetPreimage(initTraceIndex, initClaim)
	}
	stepTraceIndex := new(big.Int).Add(traceIndex, big.NewInt(1))
	stepClaim := new(big.Int).Add(new(big.Int).SetBytes(claim), big.NewInt(1))
	return BuildAlphabetPreimage(stepTraceIndex, stepClaim)
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
func BuildAlphabetPreimage(traceIndex *big.Int, claim *big.Int) []byte {
	return append(traceIndex.FillBytes(make([]byte, 32)), claim.FillBytes(make([]byte, 32))...)
}

func alphabetStateHash(state []byte) common.Hash {
	h := crypto.Keccak256Hash(state)
	h[0] = mipsevm.VMStatusInvalid
	return h
}
