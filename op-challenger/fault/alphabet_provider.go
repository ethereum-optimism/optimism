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
		maxLen: uint64(1 << depth),
	}
}

// GetPreimage returns the preimage for the given hash.
func (ap *AlphabetProvider) GetPreimage(i uint64) ([]byte, error) {
	// The index cannot be larger than the maximum index as computed by the depth.
	if i >= ap.maxLen {
		return []byte{}, ErrIndexTooLarge
	}
	// We extend the deepest hash to the maximum depth if the trace is not expansive.
	if i >= uint64(len(ap.state)) {
		return ap.GetPreimage(uint64(len(ap.state)) - 1)
	}
	return buildAlphabetClaimBytes(i, ap.state[i]), nil
}

// Get returns the claim value at the given index in the trace.
func (ap *AlphabetProvider) Get(i uint64) (common.Hash, error) {
	claimBytes, err := ap.GetPreimage(i)
	if err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(claimBytes), nil
}

// buildAlphabetClaimBytes constructs the claim bytes for the index and state item.
func buildAlphabetClaimBytes(i uint64, letter string) []byte {
	return append(IndexToBytes(i), []byte(letter)...)
}

// IndexToBytes converts an index to a byte slice big endian
func IndexToBytes(i uint64) []byte {
	big := new(big.Int)
	big.SetUint64(i)
	return big.Bytes()
}
