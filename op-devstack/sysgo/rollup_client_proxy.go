package sysgo

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/dial"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// RollupClient is the interface we wrap. The proxy implements dial.RollupClientInterface
// by wrapping a real client and only intercepting SyncStatus calls.
type RollupClient interface {
	dial.RollupClientInterface
}

// L2Client is the minimal interface needed to query L2 blocks.
// This matches the interface defined in op-batcher/batcher/driver.go
type L2Client interface {
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
}

// rollupClientProxy wraps a RollupClient and allows test control over responses.
// This is used to control what L2 blocks the batcher sees, enabling precise
// pause control without modifying the batcher itself.
//
// When pauseAtBlockNum is set to a non-zero value, the proxy will cap the
// UnsafeL2 head in SyncStatus responses to pauseAtBlockNum. This causes
// the batcher to process up to and including pauseAtBlockNum, then stop
// because it won't see any blocks beyond that point.
type rollupClientProxy struct {
	inner    RollupClient
	l2Client L2Client // to query actual block data for hash consistency

	mu              sync.RWMutex
	pauseAtBlockNum uint64 // 0 = no pause
}

// newRollupClientProxy creates a new proxy wrapping the given rollup client.
// The l2Client is used to query actual block data when capping the unsafe head,
// ensuring the returned block hash matches the block number.
func newRollupClientProxy(inner RollupClient, l2Client L2Client) *rollupClientProxy {
	return &rollupClientProxy{
		inner:    inner,
		l2Client: l2Client,
	}
}

// SyncStatus intercepts and potentially modifies the sync status response.
// If a pause is active (pauseAtBlockNum > 0), it caps the UnsafeL2 head
// to pauseAtBlockNum, allowing the batcher to process up to and including
// that block, but preventing it from seeing any blocks beyond.
func (p *rollupClientProxy) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	status, err := p.inner.SyncStatus(ctx)
	if err != nil || status == nil {
		return status, err
	}

	p.mu.RLock()
	pauseNum := p.pauseAtBlockNum
	p.mu.RUnlock()

	// If not paused or unsafe head is already at or below pause point, return as-is
	if pauseNum == 0 || status.UnsafeL2.Number <= pauseNum {
		return status, nil
	}

	// Make a copy of the status to avoid modifying the original
	statusCopy := *status

	// Cap the unsafe head at pauseNum (the last block the batcher should see)
	cappedBlockNum := pauseNum

	// Query the actual block to get the correct hash for consistency
	block, err := p.l2Client.BlockByNumber(ctx, big.NewInt(int64(cappedBlockNum)))
	if err != nil {
		// If we can't query the block, return the error rather than
		// returning inconsistent data
		return nil, err
	}

	// Update the unsafe head to the capped value with correct hash
	statusCopy.UnsafeL2 = eth.L2BlockRef{
		Hash:           block.Hash(),
		Number:         cappedBlockNum,
		ParentHash:     block.ParentHash(),
		Time:           block.Time(),
		L1Origin:       status.UnsafeL2.L1Origin, // Keep original L1 origin
		SequenceNumber: status.UnsafeL2.SequenceNumber,
	}

	// Also cap SafeL2 if it's somehow ahead (defensive)
	if statusCopy.SafeL2.Number >= pauseNum {
		statusCopy.SafeL2 = statusCopy.UnsafeL2
	}

	// And FinalizedL2 (also defensive)
	if statusCopy.FinalizedL2.Number >= pauseNum {
		statusCopy.FinalizedL2 = statusCopy.UnsafeL2
	}

	return &statusCopy, nil
}

// setPauseAtBlock sets the block number to pause at.
// When the batcher queries sync status and the unsafe head is beyond
// this block, the proxy will cap it to blockNum (inclusive).
// The batcher will process up to and including blockNum, but won't see
// any blocks after it. Set to 0 to clear the pause.
func (p *rollupClientProxy) setPauseAtBlock(blockNum uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pauseAtBlockNum = blockNum
}

// clearPause removes any active pause, allowing the batcher to see
// all available blocks again.
func (p *rollupClientProxy) clearPause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pauseAtBlockNum = 0
}

// getPause returns the current pause block number (0 = no pause).
func (p *rollupClientProxy) getPause() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pauseAtBlockNum
}

// Close is a no-op for the proxy since it doesn't own the underlying client.
func (p *rollupClientProxy) Close() {
	// No-op: we don't own the inner client
}

// OutputAtBlock forwards to the inner client.
func (p *rollupClientProxy) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	if p.inner == nil {
		return nil, nil
	}
	return p.inner.OutputAtBlock(ctx, blockNum)
}

// RollupConfig forwards to the inner client.
func (p *rollupClientProxy) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	if p.inner == nil {
		return nil, nil
	}
	return p.inner.RollupConfig(ctx)
}

// StartSequencer forwards to the inner client.
func (p *rollupClientProxy) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	if p.inner == nil {
		return nil
	}
	return p.inner.StartSequencer(ctx, unsafeHead)
}

// SequencerActive forwards to the inner client.
func (p *rollupClientProxy) SequencerActive(ctx context.Context) (bool, error) {
	if p.inner == nil {
		return false, nil
	}
	return p.inner.SequencerActive(ctx)
}
