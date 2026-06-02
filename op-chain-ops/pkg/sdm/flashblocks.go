package sdm

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// FlashblockSummary captures the per-flashblock fields useful when comparing a final block
// against the flashblocks that were streamed while it was being built.
type FlashblockSummary struct {
	BlockNumber     uint64 `json:"block_number"`
	Index           int    `json:"index"`
	TxCount         int    `json:"tx_count"`
	HasPostExecTx   bool   `json:"has_post_exec_tx"`
	PostExecTxBytes int    `json:"post_exec_tx_bytes,omitempty"`
}

// FlashblockCollector records flashblocks observed from an op-rbuilder flashblocks websocket.
type FlashblockCollector struct {
	mu      sync.Mutex
	byBlock map[uint64][]FlashblockSummary
	next    chan FlashblockSummary
	err     error
}

// StartFlashblockCollector dials url and records every flashblock until ctx is canceled.
func StartFlashblockCollector(ctx context.Context, url string) (*FlashblockCollector, error) {
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("dial flashblocks websocket %s: %w", url, err)
	}

	collector := &FlashblockCollector{
		byBlock: make(map[uint64][]FlashblockSummary),
		next:    make(chan FlashblockSummary, 1024),
	}
	go func() {
		defer conn.Close(websocket.StatusNormalClosure, "sdm-devnet done")
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				if ctx.Err() == nil {
					collector.setErr(err)
				}
				return
			}

			var fb sources.Flashblock
			if err := json.Unmarshal(msg, &fb); err != nil {
				collector.setErr(fmt.Errorf("unmarshal flashblock: %w", err))
				continue
			}

			summary := FlashblockSummary{
				BlockNumber:   uint64(fb.Metadata.BlockNumber),
				Index:         fb.Index,
				TxCount:       len(fb.Diff.Transactions),
				HasPostExecTx: fb.Diff.PostExecTx != nil,
			}
			if fb.Diff.PostExecTx != nil {
				summary.PostExecTxBytes = len(*fb.Diff.PostExecTx)
			}

			collector.mu.Lock()
			collector.byBlock[summary.BlockNumber] = append(collector.byBlock[summary.BlockNumber], summary)
			collector.mu.Unlock()
			select {
			case collector.next <- summary:
			default:
			}
		}
	}()
	return collector, nil
}

func (c *FlashblockCollector) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.err = err
}

func (c *FlashblockCollector) Err() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.err
}

func (c *FlashblockCollector) WaitNext(ctx context.Context) (FlashblockSummary, bool) {
	if c == nil {
		return FlashblockSummary{}, false
	}
	select {
	case summary := <-c.next:
		return summary, true
	case <-ctx.Done():
		return FlashblockSummary{}, false
	}
}

func (c *FlashblockCollector) DrainPending() int {
	if c == nil {
		return 0
	}
	drained := 0
	for {
		select {
		case <-c.next:
			drained++
		default:
			return drained
		}
	}
}

func (c *FlashblockCollector) Summaries(blockNum uint64) []FlashblockSummary {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := append([]FlashblockSummary(nil), c.byBlock[blockNum]...)
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

// WaitSummaries waits briefly for at least one flashblock for blockNum. It returns an empty slice
// if none arrives before the timeout, so callers can report "0 observed" without failing the SDM
// validation itself.
func (c *FlashblockCollector) WaitSummaries(ctx context.Context, blockNum uint64, timeout time.Duration) []FlashblockSummary {
	if c == nil {
		return nil
	}
	if summaries := c.Summaries(blockNum); len(summaries) > 0 {
		return summaries
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-waitCtx.Done():
			return c.Summaries(blockNum)
		case <-ticker.C:
			if summaries := c.Summaries(blockNum); len(summaries) > 0 {
				return summaries
			}
		}
	}
}
