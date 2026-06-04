package karsttest

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/plan"
	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// flashblockChannelBuffer bounds the FlashblockClient's internal channel. The
// consumer only does map inserts, so it keeps up easily; the buffer just absorbs
// bursts so the websocket reader never has to drop flashblocks.
const flashblockChannelBuffer uint = 256

// FlashblockTracker verifies that the L2 transactions a check submits are
// included in flashblocks. It is endpoint-agnostic: it consumes any stream that
// conforms to the flashblocks websocket API (the same JSON op-rbuilder and
// rollup-boost emit), recording which tx hashes appear in each block's flashblock
// receipts. TrackingOption hooks txplan so every included tx records the block it
// landed in, and Verify asserts each such tx's hash appears in a flashblock for
// that block.
//
// Transactions that never land on-chain are never tracked: TrackingOption only
// records a tx once its inclusion receipt resolves, so e.g. the EIP-7825 probe
// (rejected at submission, so it has no receipt) is correctly excluded from
// verification.
type FlashblockTracker struct {
	logger log.Logger

	mu sync.Mutex
	// flashblockTxs maps an L2 block number to the set of tx hashes observed in
	// that block's flashblock receipts.
	flashblockTxs map[uint64]map[common.Hash]struct{}
	// included maps a submitted tx hash to the block its inclusion receipt
	// reported. includedOrder preserves first-seen order so diagnostics are
	// stable.
	included      map[common.Hash]uint64
	includedOrder []common.Hash
}

// NewFlashblockTracker returns an empty tracker. Combine TrackingOption onto the
// base plan to record submitted txs, feed Consume a flashblock stream, then call
// Verify.
func NewFlashblockTracker(logger log.Logger) *FlashblockTracker {
	return &FlashblockTracker{
		logger:        logger,
		flashblockTxs: make(map[uint64]map[common.Hash]struct{}),
		included:      make(map[common.Hash]uint64),
	}
}

// TrackingOption returns a txplan.Option that records the inclusion block of
// every tx whose receipt resolves. Combine it onto the base plan AFTER the
// inclusion option (NewBasePlan installs WithRetryInclusion) so it wraps the
// resolved receipt rather than being overwritten. Txs that error before
// producing a receipt are not recorded.
func (ft *FlashblockTracker) TrackingOption() txplan.Option {
	return func(tx *txplan.PlannedTx) {
		tx.Included.Wrap(func(fn plan.Fn[*types.Receipt]) plan.Fn[*types.Receipt] {
			return func(ctx context.Context) (*types.Receipt, error) {
				receipt, err := fn(ctx)
				if err == nil && receipt != nil {
					ft.recordIncluded(receipt.TxHash, receipt.BlockNumber.Uint64())
				}
				return receipt, err
			}
		})
	}
}

func (ft *FlashblockTracker) recordIncluded(hash common.Hash, block uint64) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	if _, ok := ft.included[hash]; ok {
		return
	}
	ft.included[hash] = block
	ft.includedOrder = append(ft.includedOrder, hash)
}

// Consume drains the flashblock stream until the channel closes or ctx is
// cancelled, recording the tx hashes carried by each flashblock's receipts. It
// is meant to run in its own goroutine for the lifetime of the checks.
func (ft *FlashblockTracker) Consume(ctx context.Context, ch <-chan *sources.Flashblock) {
	for {
		select {
		case <-ctx.Done():
			return
		case fb, ok := <-ch:
			if !ok {
				return
			}
			ft.recordFlashblock(fb)
		}
	}
}

func (ft *FlashblockTracker) recordFlashblock(fb *sources.Flashblock) {
	if fb == nil || len(fb.Metadata.Receipts) == 0 {
		return
	}
	block := uint64(fb.Metadata.BlockNumber)
	ft.mu.Lock()
	defer ft.mu.Unlock()
	set := ft.flashblockTxs[block]
	if set == nil {
		set = make(map[common.Hash]struct{})
		ft.flashblockTxs[block] = set
	}
	for hashHex := range fb.Metadata.Receipts {
		set[common.HexToHash(hashHex)] = struct{}{}
	}
}

// Verify blocks until every tracked tx's hash has been observed in a flashblock
// for its inclusion block, or until timeout elapses. It returns nil immediately
// if no txs were tracked (e.g. a check that submits nothing includable). On
// timeout it returns a diagnostic naming the txs that were never matched and the
// blocks for which flashblocks were observed.
func (ft *FlashblockTracker) Verify(ctx context.Context, timeout time.Duration) error {
	if ft.numTracked() == 0 {
		ft.logger.Info("flashblocks: no included L2 txs to verify")
		return nil
	}

	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		if len(ft.unmatched()) == 0 {
			ft.logger.Info("flashblocks: all included L2 txs observed in flashblocks", "count", ft.numTracked())
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return ft.unmatchedErr(timeout)
		case <-ticker.C:
		}
	}
}

func (ft *FlashblockTracker) numTracked() int {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return len(ft.included)
}

// unmatched returns the tracked txs whose hash has not yet appeared in a
// flashblock for their inclusion block, in first-seen order.
func (ft *FlashblockTracker) unmatched() []common.Hash {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	return ft.unmatchedLocked()
}

func (ft *FlashblockTracker) unmatchedLocked() []common.Hash {
	var missing []common.Hash
	for _, hash := range ft.includedOrder {
		if !ft.matchedLocked(hash) {
			missing = append(missing, hash)
		}
	}
	return missing
}

func (ft *FlashblockTracker) matchedLocked(hash common.Hash) bool {
	_, ok := ft.flashblockTxs[ft.included[hash]][hash]
	return ok
}

// blocksContainingLocked returns, sorted, every block whose flashblock receipts
// referenced the hash. Used only for diagnostics — if this is non-empty for an
// unmatched tx but excludes its inclusion block, the tx appeared in a superseded
// speculative flashblock.
func (ft *FlashblockTracker) blocksContainingLocked(hash common.Hash) []uint64 {
	var blocks []uint64
	for block, set := range ft.flashblockTxs {
		if _, ok := set[hash]; ok {
			blocks = append(blocks, block)
		}
	}
	sort.Slice(blocks, func(i, j int) bool { return blocks[i] < blocks[j] })
	return blocks
}

func (ft *FlashblockTracker) unmatchedErr(timeout time.Duration) error {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	observedBlocks := make([]uint64, 0, len(ft.flashblockTxs))
	for block := range ft.flashblockTxs {
		observedBlocks = append(observedBlocks, block)
	}
	sort.Slice(observedBlocks, func(i, j int) bool { return observedBlocks[i] < observedBlocks[j] })

	missing := ft.unmatchedLocked()
	var detail strings.Builder
	for _, hash := range missing {
		fmt.Fprintf(&detail, "\n  - tx %s included in block %d but not in that block's flashblock receipts",
			hash, ft.included[hash])
		if seenIn := ft.blocksContainingLocked(hash); len(seenIn) > 0 {
			fmt.Fprintf(&detail, " (it did appear in flashblocks for blocks %v — likely a superseded speculative flashblock)", seenIn)
		}
	}

	if len(observedBlocks) == 0 {
		return fmt.Errorf("flashblocks: %d tracked tx(s) not confirmed and no flashblocks observed within %s — is --flashblocks-ws pointing at a live flashblocks endpoint?%s",
			len(missing), timeout, detail.String())
	}
	return fmt.Errorf("flashblocks: %d tracked tx(s) not confirmed in flashblocks within %s; flashblocks were observed for blocks %v:%s",
		len(missing), timeout, observedBlocks, detail.String())
}

// StartFlashblockTracking dials the flashblocks websocket at wsURL, begins
// consuming the stream in the background, and returns a tracker, the txplan
// option that must be combined onto the base plan, and a stop function. The stop
// function cancels the stream and closes the connection; call it after Verify.
//
// wsURL may be any endpoint that speaks the flashblocks websocket API (e.g.
// op-rbuilder or rollup-boost) — the tracker only relies on the standard
// flashblock JSON shape, not on which component produced it.
func StartFlashblockTracking(ctx context.Context, logger log.Logger, wsURL string) (*FlashblockTracker, txplan.Option, func(), error) {
	wsCl, err := client.DialWS(ctx, client.WSConfig{
		URL:         wsURL,
		Log:         logger,
		MaxAttempts: 10,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("dial flashblocks ws %q: %w", wsURL, err)
	}

	tracker := NewFlashblockTracker(logger)
	fbClient := sources.NewFlashblockClient(wsCl, logger.With("stream", "flashblocks"), flashblockChannelBuffer)

	streamCtx, streamCancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := fbClient.Start(streamCtx); err != nil {
			logger.Warn("flashblocks: stream ended with error", "err", err)
		}
	}()
	go func() {
		defer wg.Done()
		tracker.Consume(streamCtx, fbClient.Next())
	}()

	stop := func() {
		streamCancel()
		wg.Wait()
		_ = wsCl.Close(websocket.StatusNormalClosure, "check-karst done")
	}
	return tracker, tracker.TrackingOption(), stop, nil
}
