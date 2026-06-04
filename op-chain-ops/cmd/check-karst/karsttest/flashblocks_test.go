package karsttest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/sources"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// flashblockFor builds a minimal flashblock for the given block whose receipts
// reference the supplied tx hashes — the only fields the tracker reads.
func flashblockFor(block int, hashes ...common.Hash) *sources.Flashblock {
	fb := &sources.Flashblock{}
	fb.Metadata.BlockNumber = block
	fb.Metadata.Receipts = make(map[string]interface{}, len(hashes))
	for _, h := range hashes {
		fb.Metadata.Receipts[h.Hex()] = nil
	}
	return fb
}

func TestFlashblockTracker_VerifyMatch(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	h := common.HexToHash("0x1234")

	ft.recordIncluded(h, 5)
	require.Len(t, ft.unmatched(), 1, "tx is unmatched until its flashblock arrives")

	ft.recordFlashblock(flashblockFor(5, h))
	require.Empty(t, ft.unmatched(), "tx matches once a flashblock for its block carries it")
	require.NoError(t, ft.Verify(context.Background(), time.Second))
}

func TestFlashblockTracker_VerifyNoTracked(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	// A check that includes no txs (e.g. EIP-7825 rejection) tracks nothing.
	require.NoError(t, ft.Verify(context.Background(), 50*time.Millisecond))
}

func TestFlashblockTracker_VerifyTimeoutNoFlashblocks(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	ft.recordIncluded(common.HexToHash("0xabc"), 9)

	err := ft.Verify(context.Background(), 50*time.Millisecond)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no flashblocks observed")
}

func TestFlashblockTracker_VerifySupersededDiagnostic(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	h := common.HexToHash("0x55")
	ft.recordIncluded(h, 10)

	// The tx surfaced in a flashblock for block 9 (speculative, later superseded)
	// and block 10 produced a flashblock without it.
	ft.recordFlashblock(flashblockFor(9, h))
	ft.recordFlashblock(flashblockFor(10, common.HexToHash("0xother")))

	err := ft.Verify(context.Background(), 50*time.Millisecond)
	require.Error(t, err)
	require.Contains(t, err.Error(), "superseded")
	require.Contains(t, err.Error(), "[9]", "diagnostic names the block that actually carried the tx")
}

func TestFlashblockTracker_ConsumeUntilMatch(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	h := common.HexToHash("0x99")
	ft.recordIncluded(h, 3)

	ch := make(chan *sources.Flashblock, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go ft.Consume(ctx, ch)

	ch <- flashblockFor(3, h)
	require.NoError(t, ft.Verify(context.Background(), time.Second))
}

func TestFlashblockTracker_RecordIncludedDeduplicates(t *testing.T) {
	ft := NewFlashblockTracker(testlog.Logger(t, log.LevelError))
	h := common.HexToHash("0x77")
	ft.recordIncluded(h, 1)
	ft.recordIncluded(h, 2) // ignored: first inclusion block wins
	require.Equal(t, 1, ft.numTracked())
	require.Len(t, ft.includedOrder, 1)
	require.Equal(t, uint64(1), ft.included[h])
}
