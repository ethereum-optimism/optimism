package dial

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

func WaitRollupSync(
	ctx context.Context,
	lgr log.Logger,
	rollup SyncStatusProvider,
	l1BlockTarget uint64,
	pollInterval time.Duration,
) error {
	timer := time.NewTimer(pollInterval)
	defer timer.Stop()

	for {
		syncst, err := rollup.SyncStatus(ctx)
		if err != nil {
			// don't log assuming caller handles and logs errors
			return fmt.Errorf("getting sync status: %w", err)
		}

		lgr := lgr.With("current_l1", syncst.CurrentL1, "target_l1", l1BlockTarget)
		if syncst.CurrentL1.Number >= l1BlockTarget {
			lgr.Info("rollup current L1 block target reached")
			return nil
		}

		lgr.Info("rollup current L1 block still behind target, retrying")

		timer.Reset(pollInterval)
		select {
		case <-timer.C: // next try
		case <-ctx.Done():
			lgr.Warn("waiting for rollup sync timed out")
			return ctx.Err()
		}
	}
}
