package fault

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// AlphabetProvider is a [TraceProvider] that provides claims for specific
// indices in the given trace.
type AlphabetProvider struct {
	state  []string
	maxLen uint64
}

// NewAlphabetProvider returns a new [AlphabetProvider].
func NewAlphabetProvider(state string, depth uint64) *AlphabetProvider {
	return &AlphabetProvider{
		state:  strings.Split(state, ""),
		maxLen: (1 << depth),
	}
}

// Get returns the claim value at the given index in the trace.
func (ap *AlphabetProvider) Get(i uint64) (common.Hash, error) {
	// The index cannot be larger than the maximum index as computed by the depth.
	if i >= ap.maxLen {
		return common.Hash{}, ErrIndexTooLarge
	}
	// We extend the deepest hash to the maximum depth if the trace is not expansive.
	if i >= uint64(len(ap.state)) {
		return ap.Get(uint64(len(ap.state)) - 1)
	}
	return ap.ComputeAlphabetClaim(i), nil
}

// ComputeAlphabetClaim computes the claim for the given index in the trace.
func (ap *AlphabetProvider) ComputeAlphabetClaim(i uint64) common.Hash {
	concatenated := append(IndexToBytes(i), []byte(ap.state[i])...)
	return common.BytesToHash(concatenated)
}

// IndexToBytes converts an index to a byte slice big endian
func IndexToBytes(i uint64) []byte {
	big := new(big.Int)
	big.SetUint64(i)
	return big.Bytes()
}
