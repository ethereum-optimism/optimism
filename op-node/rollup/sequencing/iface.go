package sequencing

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/event"
)

type SequencerIface interface {
	event.Deriver
	// NextAction returns when the sequencer needs to do the next change, and iff it should do so.
	NextAction() (t time.Time, ok bool)
	// NextActionChanged returns a channel that is signalled when the schedule reported by
	// NextAction changed outside of event processing, i.e. from an RPC-driven Start or Stop.
	// The driver event loop selects on it so that it re-plans promptly.
	NextActionChanged() <-chan struct{}
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
