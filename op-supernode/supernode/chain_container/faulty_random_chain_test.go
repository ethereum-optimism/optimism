package chain_container

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
)

// elState is the execution-layer pre-state a FaultyRandomChain presents to the
// EngineController at the start of a RewindToTimestamp, named after the state
// taxonomy in issue-20929.
type elState int

const (
	elAtTarget       elState = iota // A: unsafe == target (already rewound)
	elSyntheticStuck                // B: unsafe at target height, hash != target; target still present
	elAboveTarget                   // C/D: unsafe above target (normal full rewind)
	elBelowTarget                   // E: unsafe below target; target truncated out of the EL DB
)

// FaultyRandomChain wraps a RandomChain as the engine_controller l2Provider and
// models an execution layer whose reported head can diverge from what
// ForkchoiceUpdate requested -- the divergence the rewind path must survive. The
// embedded *RandomChain serves every l2Provider method not overridden here.
type FaultyRandomChain struct {
	*RandomChain

	// elUnsafe/elSafe/elFinalized are what the EL reports for the block labels,
	// independent of FCU. Seeded by newFaultyRandomChain; ForkchoiceUpdate moves
	// them when cooperative.
	elUnsafe, elSafe, elFinalized eth.L2BlockRef
	state                         elState
	targetNum                     uint64 // in elBelowTarget, byNumber >= this is "gone"
	cooperative                   bool   // FCU lands the requested head (vs. ignores it)
	byNumberErr                   error  // when set, L2BlockRefByNumber returns it (transient RPC failure)
	fcuDeadlines                  int    // ForkchoiceUpdate returns context.DeadlineExceeded this many times first

	newPayloadCalls int // synthetic-insert attempts
	fcuCalls        int
}

// newFaultyRandomChain builds a faulty engine for chain rc presenting EL state
// `state` relative to target. elSafe/elFinalized are seeded from the generated
// safe/finalized heads so computeRewindTargets behaves.
func newFaultyRandomChain(rc *RandomChain, state elState, target eth.L2BlockRef) *FaultyRandomChain {
	f := &FaultyRandomChain{
		RandomChain: rc,
		state:       state,
		targetNum:   target.Number,
		cooperative: true,
		elSafe:      rc.l2[rc.safe].Ref,
		elFinalized: rc.l2[rc.finalized].Ref,
	}
	switch state {
	case elAtTarget:
		f.elUnsafe = target
	case elSyntheticStuck:
		stuck := target
		stuck.Hash = flipHash(target.Hash) // synthetic: same height, different hash
		f.elUnsafe = stuck
	case elBelowTarget:
		f.elUnsafe = rc.l2[target.Number-1].Ref
	default: // elAboveTarget
		f.elUnsafe = rc.l2[rc.unsafe].Ref
	}
	return f
}

func flipHash(h common.Hash) common.Hash {
	h[0] ^= 0xff
	return h
}

func (f *FaultyRandomChain) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	switch label {
	case eth.Unsafe:
		return f.elUnsafe, nil
	case eth.Finalized:
		return f.elFinalized, nil
	default:
		return f.elSafe, nil
	}
}

func (f *FaultyRandomChain) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	if f.byNumberErr != nil {
		return eth.L2BlockRef{}, f.byNumberErr
	}
	if f.state == elBelowTarget && num >= f.targetNum {
		return eth.L2BlockRef{}, ethereum.NotFound
	}
	return f.RandomChain.L2BlockRefByNumber(ctx, num)
}

func (f *FaultyRandomChain) PayloadByNumber(ctx context.Context, number uint64) (*eth.ExecutionPayloadEnvelope, error) {
	if f.state == elBelowTarget && number >= f.targetNum {
		return nil, ethereum.NotFound
	}
	return f.RandomChain.PayloadByNumber(ctx, number)
}

func (f *FaultyRandomChain) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	f.newPayloadCalls++
	return &eth.PayloadStatusV1{Status: eth.ExecutionValid}, nil
}

func (f *FaultyRandomChain) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	f.fcuCalls++
	if f.fcuDeadlines > 0 {
		f.fcuDeadlines-- // transient: the EL commits eventually; the CL deadline fired early
		return nil, context.DeadlineExceeded
	}
	if f.cooperative {
		f.elUnsafe = f.refForHash(state.HeadBlockHash)
		f.elSafe = f.refForHash(state.SafeBlockHash)
		f.elFinalized = f.refForHash(state.FinalizedBlockHash)
	}
	return &eth.ForkchoiceUpdatedResult{
		PayloadStatus: eth.PayloadStatusV1{Status: eth.ExecutionValid},
	}, nil
}

// Conformance to the engine_controller l2Provider (unexported there).
var _ = engine_controller.NewEngineControllerWithL2AndRollup((*FaultyRandomChain)(nil), nil)
