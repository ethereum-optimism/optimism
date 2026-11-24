package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ResetWatch observes an L2CL node in the background for a reset of its derivation pipeline
// and engine state. Start one with L2CLNode.WatchForReset and finish it with VerifyNoReset.
type ResetWatch struct {
	cl     *L2CLNode
	cancel context.CancelFunc
	done   chan struct{}
	// err and samples are written by the watch goroutine before it closes done, and read only
	// after done is closed.
	err     error
	samples int
}

// WatchForReset starts polling the node's sync status in the background every 100ms, until
// VerifyNoReset is called. A reset restarts derivation from an L1 block at least a channel timeout
// behind the safe head's L1 origin, and rewinds or clears the node's L2 heads until the engine is
// reset. The watch records a reset when one of its polls observes such a regression of CurrentL1
// or of the unsafe or local-safe head, so a reset the node recovers from between two polls goes
// unnoticed.
//
// Use it only while no legitimate unsafe reorg is expected on the node, such as a consolidation
// conflict that the derived safe chain wins, a sequencer restarted at an older head, or EL sync:
// those regress the unsafe head without a reset.
func (cl *L2CLNode) WatchForReset() *ResetWatch {
	// Derivation must have started, or a reset could not rewind CurrentL1.
	cl.AwaitMinL1Processed(0)
	initial := cl.SyncStatus()
	ctx, cancel := context.WithCancel(cl.ctx)
	w := &ResetWatch{
		cl:     cl,
		cancel: cancel,
		done:   make(chan struct{}),
	}
	cl.t.Cleanup(func() {
		cancel()
		<-w.done
	})
	cl.log.Info("Watching for reset", "node", cl, "currentL1", initial.CurrentL1.ID(),
		"unsafe", initial.UnsafeL2.ID(), "localSafe", initial.LocalSafeL2.ID())
	go w.run(ctx, initial)
	return w
}

func (w *ResetWatch) run(ctx context.Context, last *eth.SyncStatus) {
	defer close(w.done)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		rpcCtx, cancel := context.WithTimeout(ctx, DefaultTimeout)
		status, err := w.cl.inner.RollupAPI().SyncStatus(rpcCtx)
		cancel()
		if err != nil {
			if ctx.Err() == nil {
				w.cl.log.Warn("SyncStatus RPC failed while watching for reset; will retry", "node", w.cl, "err", err)
			}
			continue
		}
		w.samples++
		if err := syncStatusRegression(last, status); err != nil {
			w.err = err
			w.cl.log.Error("Observed reset", "node", w.cl, "err", err)
			return
		}
		last = status
	}
}

// VerifyNoReset stops the watch and fails the test if it observed a reset, or if it could not
// read the node's sync status even once.
func (w *ResetWatch) VerifyNoReset() {
	w.cancel()
	<-w.done
	w.cl.require.NoError(w.err, "%s must not reset", w.cl)
	w.cl.require.Positive(w.samples, "watch must read the sync status of %s at least once", w.cl)
	w.cl.log.Info("No reset observed", "node", w.cl, "samples", w.samples)
}

// syncStatusRegression returns an error if cur moved the derivation origin or the unsafe or
// local-safe head backwards compared to prev.
func syncStatusRegression(prev, cur *eth.SyncStatus) error {
	if cur.CurrentL1.Number < prev.CurrentL1.Number {
		return fmt.Errorf("CurrentL1 regressed from %s to %s", prev.CurrentL1.ID(), cur.CurrentL1.ID())
	}
	if cur.UnsafeL2.Number < prev.UnsafeL2.Number {
		return fmt.Errorf("unsafe head regressed from %s to %s", prev.UnsafeL2.ID(), cur.UnsafeL2.ID())
	}
	if cur.LocalSafeL2.Number < prev.LocalSafeL2.Number {
		return fmt.Errorf("local-safe head regressed from %s to %s", prev.LocalSafeL2.ID(), cur.LocalSafeL2.ID())
	}
	return nil
}
