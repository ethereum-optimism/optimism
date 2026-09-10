// Package positions resolves private messages to their deterministic public
// projection positions using ordinary execution and rollup RPCs.
package positions

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/render"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txintent"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ExecutionSource supplies canonical block references and complete receipts.
type ExecutionSource interface {
	apis.ReceiptsFetcher
	BlockRefByNumber(context.Context, uint64) (eth.BlockRef, error)
}

// SafetySource supplies the public projection's derived safety frontier.
type SafetySource interface {
	SyncStatus(context.Context) (*eth.SyncStatus, error)
}

// Resolver uses the same export policy as the batcher. Public receipts verify
// the predicted positions; they never redefine the private log ordering.
type Resolver struct {
	private, projection ExecutionSource
	safety              SafetySource
	emitters            render.EmitterSet
	timeout             time.Duration
}

// New constructs a resolver for a private/projection pair. Callers must supply
// matching chains and the same emitter configuration used by the batcher.
func New(private, projection ExecutionSource, safety SafetySource, emitters render.EmitterSet, timeout time.Duration) *Resolver {
	return &Resolver{private: private, projection: projection, safety: safety, emitters: emitters, timeout: timeout}
}

var _ txintent.PositionResolver = (*Resolver)(nil)

func (r *Resolver) Owns(ctx context.Context, block eth.BlockRef) bool {
	got, err := r.private.BlockRefByNumber(ctx, block.Number)
	return err == nil && got == block
}

// ResolvePositions waits for publication and checks the complete projected log
// sequence, including indices and emitters. A reorg or mismatch fails resolution.
func (r *Resolver) ResolvePositions(ctx context.Context, rec *types.Receipt, block eth.BlockRef) ([]txintent.PublicPosition, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if rec.BlockHash != block.Hash || rec.BlockNumber == nil || rec.BlockNumber.Uint64() != block.Number || !r.Owns(ctx, block) {
		return nil, fmt.Errorf("private receipt does not belong to canonical block %s", block)
	}
	privateLogs, err := blockLogs(ctx, r.private, block.Hash)
	if err != nil {
		return nil, err
	}
	expected := render.RenderedLogs(privateLogs, r.emitters)
	byPrivateIndex := make(map[uint]uint32, len(expected))
	for _, log := range expected {
		byPrivateIndex[uint(log.PrivateLogIndex)] = log.RenderedLogIndex
	}
	out := make([]txintent.PublicPosition, len(rec.Logs))
	public := false
	for _, log := range rec.Logs {
		if log.Index >= uint(len(privateLogs)) || !sameLog(log, privateLogs[log.Index]) || log.TxHash != rec.TxHash || privateLogs[log.Index].TxHash != rec.TxHash {
			return nil, fmt.Errorf("receipt log %d does not match its private block", log.Index)
		}
		_, included := byPrivateIndex[log.Index]
		public = public || included
	}
	if !public {
		return out, nil
	}
	if err := r.awaitProjection(ctx, block.Number); err != nil {
		return nil, err
	}
	projected, err := r.projection.BlockRefByNumber(ctx, block.Number)
	if err != nil {
		return nil, err
	}
	if projected.Time != block.Time {
		return nil, fmt.Errorf("private/projection timestamp mismatch at block %d", block.Number)
	}
	logs, err := blockLogs(ctx, r.projection, projected.Hash)
	if err != nil {
		return nil, err
	}
	if len(logs) != len(expected) {
		return nil, fmt.Errorf("projection block %d has %d logs, expected %d", block.Number, len(logs), len(expected))
	}
	for i, log := range logs {
		want := *expected[i].Log
		if want.Address != predeploys.L2toL2CrossDomainMessengerAddr && want.Address != predeploys.CrossL2InboxAddr {
			want.Address = predeploys.EventReplayerAddr
		}
		if log.Index != uint(i) || !sameLog(log, &want) {
			return nil, fmt.Errorf("projection log %d of block %d differs from the private export sequence", i, block.Number)
		}
	}
	// References are checked again after receipt retrieval to reject a mapping
	// assembled across branches. Subsequent reorgs remain ordinary interop risk.
	current, err := r.projection.BlockRefByNumber(ctx, block.Number)
	if err != nil || current != projected || !r.Owns(ctx, block) {
		return nil, fmt.Errorf("private/projection chain changed during message resolution")
	}
	for i, log := range rec.Logs {
		if index, ok := byPrivateIndex[log.Index]; ok {
			out[i] = txintent.PublicPosition{Origin: logs[index].Address, LogIndex: index, Public: true}
		}
	}
	return out, nil
}

func (r *Resolver) awaitProjection(ctx context.Context, number uint64) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		status, err := r.safety.SyncStatus(ctx)
		if err == nil && status != nil && status.SafeL2.Number >= number {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for projection publication of block %d: %w", number, ctx.Err())
		case <-ticker.C:
		}
	}
}

func blockLogs(ctx context.Context, source ExecutionSource, hash common.Hash) ([]*types.Log, error) {
	_, receipts, err := source.FetchReceipts(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("reading receipts for %s: %w", hash, err)
	}
	var logs []*types.Log
	for _, receipt := range receipts.Geth() {
		logs = append(logs, receipt.Logs...)
	}
	return logs, nil
}

func sameLog(a, b *types.Log) bool {
	if a.Address != b.Address || len(a.Topics) != len(b.Topics) || !bytes.Equal(a.Data, b.Data) {
		return false
	}
	for i := range a.Topics {
		if a.Topics[i] != b.Topics[i] {
			return false
		}
	}
	return true
}
