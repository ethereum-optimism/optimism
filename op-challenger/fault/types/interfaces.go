package types

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

// OracleUpdater is a generic interface for updating oracles.
type OracleUpdater interface {
	// UpdateOracle updates the oracle with the given data.
	UpdateOracle(ctx context.Context, data *PreimageOracleData) error
}

// TraceProvider is a generic way to get a claim value at a specific step in the trace.
type TraceProvider interface {
	// Get returns the claim value at the requested index.
	// Get(i) = Keccak256(GetPreimage(i))
	Get(ctx context.Context, i uint64) (common.Hash, error)

	// GetStepData returns the data required to execute the step at the specified trace index.
	// This includes the pre-state of the step (not hashed), the proof data required during step execution
	// and any pre-image data that needs to be loaded into the oracle prior to execution (may be nil)
	// The prestate returned from GetStepData for trace 10 should be the pre-image of the claim from trace 9
	GetStepData(ctx context.Context, i uint64) (prestate []byte, proofData []byte, preimageData *PreimageOracleData, err error)

	// AbsolutePreState is the pre-image value of the trace that transitions to the trace value at index 0
	AbsolutePreState(ctx context.Context) (preimage []byte, err error)
}

// Responder takes a response action & executes.
// For full op-challenger this means executing the transaction on chain.
type Responder interface {
	CanResolve(ctx context.Context) bool
	Resolve(ctx context.Context) error
	Respond(ctx context.Context, response Claim) error
	Step(ctx context.Context, stepData StepCallData) error
}
