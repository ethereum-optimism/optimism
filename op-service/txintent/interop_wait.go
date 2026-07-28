package txintent

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// interopWaitDisabled flags a tx that must not wait for the messages it executes.
type interopWaitDisabled struct{}

// WaitForTime blocks until the chain has built a block with a timestamp of at least minTime.
type WaitForTime func(ctx context.Context, minTime uint64) error

// WithInteropDependencyWait holds back planning a tx until the chain it is sent to has reached
// every message the tx executes, read from its access-list. A block executing a message newer than
// itself is invalid, and is replaced during consolidation.
//
// Wraps the AgainstBlock stage, so it must be applied after the Option that resolves it.
// For a tx referencing a message that does not exist yet, see WithoutInteropDependencyWait.
func WithInteropDependencyWait(wait WaitForTime) txplan.Option {
	return func(tx *txplan.PlannedTx) {
		tx.AgainstBlock.DependOn(&tx.AccessList)
		tx.AgainstBlock.Wrap(func(fn plan.Fn[eth.BlockInfo]) plan.Fn[eth.BlockInfo] {
			return func(ctx context.Context) (eth.BlockInfo, error) {
				if tx.HasFlag(interopWaitDisabled{}) {
					return fn(ctx)
				}
				minTime, err := executedMessagesMinTime(tx.AccessList.Value())
				if err != nil {
					return nil, err
				}
				if minTime > 0 {
					if err := wait(ctx, minTime); err != nil {
						return nil, fmt.Errorf("waiting for chain to reach timestamp %d: %w", minTime, err)
					}
				}
				return fn(ctx)
			}
		})
	}
}

// WithoutInteropDependencyWait opts out of WithInteropDependencyWait, for a tx referencing a
// message that does not exist yet: the wait would block on the tx that produces it.
func WithoutInteropDependencyWait() txplan.Option {
	return txplan.WithFlag(interopWaitDisabled{})
}

// executedMessagesMinTime returns the timestamp the chain must reach before the tx can be planned,
// or 0 if the tx executes no message. An equal timestamp is valid, and the tx cannot land before
// the next block anyway, so the head only has to reach the newest message, not pass it.
func executedMessagesMinTime(accessList types.AccessList) (uint64, error) {
	executed, err := messages.DecodeAccessList(accessList)
	if err != nil {
		return 0, fmt.Errorf("parsing CrossL2Inbox access-list: %w", err)
	}
	var latest uint64
	for _, msg := range executed {
		latest = max(latest, msg.Timestamp)
	}
	return latest, nil
}
