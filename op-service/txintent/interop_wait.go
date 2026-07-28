package txintent

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// interopWaitDisabled flags a tx that must not wait for the messages it executes.
type interopWaitDisabled struct{}

// WaitForTime blocks until the chain has built a block with a timestamp of at least minTime.
type WaitForTime func(ctx context.Context, minTime uint64) error

// WithInteropDependencyWait holds back planning a tx until the chain it is sent to has built past
// every initiating message the tx executes. An executing message is only valid in a block that is
// strictly newer than the messages it references, so without this the tx is filtered out of the
// mempool, or its block is replaced during consolidation, whenever the destination chain lags the
// source chain.
//
// The messages are read from the tx access-list, so this covers every executing message however it
// was built: ExecuteIndexed, ExecuteIndexeds, a hand-written ExecTrigger, an ExecTrigger embedded
// in another trigger, or an access-list set directly on the tx. Transactions that do not execute
// any message are unaffected.
//
// This wraps the AgainstBlock stage, so it must be applied after the Option that resolves
// AgainstBlock (i.e. txplan.WithAgainstLatestBlock). Planning the tx against a block that already
// contains the dependency also means gas estimation, fee and nonce lookups all see it.
//
// Waiting is meaningless for a tx that references a message which does not exist yet, e.g. one
// built for a block a test sequencer has not sequenced. Those must be combined with
// WithoutInteropDependencyWait.
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

// WithoutInteropDependencyWait opts a tx out of WithInteropDependencyWait. Use it for a tx that
// references a message which is not on the source chain yet, and so can never be waited for: the
// wait would block until the tx is submitted, which is what would produce the message.
func WithoutInteropDependencyWait() txplan.Option {
	return txplan.WithFlag(interopWaitDisabled{})
}

// executedMessagesMinTime returns the timestamp an executing tx's own block must reach for the
// messages in the given access-list to be valid, or 0 if it does not execute any message.
func executedMessagesMinTime(accessList types.AccessList) (uint64, error) {
	var latest uint64
	for _, tuple := range accessList {
		if tuple.Address != predeploys.CrossL2InboxAddr {
			continue
		}
		entries := tuple.StorageKeys
		for len(entries) > 0 {
			remaining, access, err := messages.ParseAccess(entries)
			if err != nil {
				return 0, fmt.Errorf("parsing CrossL2Inbox access-list: %w", err)
			}
			entries = remaining
			latest = max(latest, access.Timestamp)
		}
	}
	if latest == 0 {
		return 0, nil
	}
	// Strictly newer, hence the +1: op-geth filters pending executing messages against the unsafe
	// head timestamp rather than the pending block's.
	// See https://github.com/ethereum-optimism/op-geth/issues/603.
	return latest + 1, nil
}
