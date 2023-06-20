package fault

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNegativeIndex = errors.New("index cannot be negative")
	ErrIndexTooLarge = errors.New("index is larger than the maximum index")
)

// TraceProvider is a generic way to get a claim value at a specific
// step in the trace.
type TraceProvider interface {
	Get(i int) (common.Hash, error)
}

type Claim struct {
	Value common.Hash
	Position
}

type Response struct {
	Attack bool // note: can we flip this to true == going right / defending??
	Value  common.Hash
}
