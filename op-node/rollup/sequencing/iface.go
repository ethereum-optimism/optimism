package sequencing

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/event"
)

type SequencerIface interface {
	event.Deriver
	// RunAction performs one sequencer action: start, seal, or (re)process a
	// block. Engine calls inside it use the sequencer's own lifetime context, so
	// it takes none. Called from the sequencer goroutine, or directly by tests.
	RunAction()
	// RunLoop runs the sequencer scheduling loop, and blocks until ctx is canceled.
	RunLoop(ctx context.Context)
	Active() bool
	Init(ctx context.Context, active bool) error
	Start(ctx context.Context, head common.Hash) error
	Stop(ctx context.Context) (hash common.Hash, err error)
	SetMaxSafeLag(ctx context.Context, v uint64) error
	OverrideLeader(ctx context.Context) error
	ConductorEnabled(ctx context.Context) bool
	SetRecoverMode(mode bool)
	Close()
}
