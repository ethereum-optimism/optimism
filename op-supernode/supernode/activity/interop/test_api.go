package interop

// TestAPI adapts a running Interop activity to the transport-agnostic
// apis.SupernodeInteropTestAPI surface the devstack DSL drives supernodes
// through. It is the in-process implementation of that surface; a supernode in
// another process implements the same interface by answering the same four
// calls over RPC.
//
// The adapter is deliberately thin — it only reshapes what
// interop_test_access.go already exposes into serializable form — so that the
// in-process path and any future RPC path cannot drift in behaviour.
//
// Test-only, like everything it calls. Production code paths must not use it.

import (
	"context"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type TestAPI struct {
	i *Interop
}

var _ apis.SupernodeInteropTestAPI = (*TestAPI)(nil)

// NewTestAPI wraps a non-nil Interop activity. Callers that may not have an
// activity must check for that themselves rather than passing nil: a non-nil
// interface holding a nil pointer would defeat the caller's own nil check.
func NewTestAPI(i *Interop) *TestAPI {
	return &TestAPI{i: i}
}

func (a *TestAPI) PauseInterop(_ context.Context, ts uint64) error {
	a.i.PauseAt(ts)
	return nil
}

func (a *TestAPI) ResumeInterop(_ context.Context) error {
	a.i.Resume()
	return nil
}

func (a *TestAPI) InteropStatus(_ context.Context) (eth.SupernodeInteropStatus, error) {
	return eth.SupernodeInteropStatus{
		BackfillAttempts:           a.i.BackfillAttempts(),
		BackfillCompleted:          a.i.BackfillCompleted(),
		ActivationTimestamp:        a.i.ActivationTimestamp(),
		VerificationStartTimestamp: a.i.VerificationStartTimestamp(),
		FirstVerifiableTimestamp:   a.i.FirstVerifiableTimestamp(),
	}, nil
}

func (a *TestAPI) InteropSealedBlocks(_ context.Context, chainID eth.ChainID) (eth.SupernodeSealedBlocks, error) {
	latest, hasLatest, err := a.i.LatestSealedBlock(chainID)
	if err != nil {
		return eth.SupernodeSealedBlocks{}, err
	}
	if !hasLatest {
		// No sealed blocks at all, so there is no first block to look up
		// either: FirstSealedBlock would fail on the empty DB rather than
		// report emptiness, and emptiness is not an error here.
		return eth.SupernodeSealedBlocks{}, nil
	}
	first, err := a.i.FirstSealedBlock(chainID)
	if err != nil {
		return eth.SupernodeSealedBlocks{}, err
	}
	return eth.SupernodeSealedBlocks{
		First:     sealedBlock(first),
		Latest:    sealedBlock(latest),
		HasBlocks: true,
	}, nil
}

func sealedBlock(seal messages.BlockSeal) eth.SupernodeSealedBlock {
	return eth.SupernodeSealedBlock{ID: seal.ID(), Timestamp: seal.Timestamp}
}
