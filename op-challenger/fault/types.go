package fault

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrNegativeIndex = errors.New("index cannot be negative")
	ErrIndexTooLarge = errors.New("index is larger than the maximum index")
)

// TraceProvider is a generic way to get a claim value at a specific
// step in the trace.
// The [AlphabetProvider] is a minimal implementation of this interface.
type TraceProvider interface {
	Get(i uint64) (common.Hash, error)
}

type Claim struct {
	Value common.Hash
	Position
}

type Response struct {
	Attack bool // note: can we flip this to true == going right / defending??
	Value  common.Hash
	Parent Claim
}

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	Respond(ctx context.Context, response Response) error
}
