package raftwallogdb

import (
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"

	"github.com/ethereum-optimism/optimism/op-core/interop"
	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
)

// gatedWAL blocks StoreLog until gate is closed, signaling entry via started.
type gatedWAL struct {
	walStore
	started chan struct{}
	gate    chan struct{}
}

func (g *gatedWAL) StoreLog(l *raft.Log) error {
	close(g.started)
	<-g.gate
	return g.walStore.StoreLog(l)
}

type readResults struct {
	latest      eth.BlockID
	latestOK    bool
	findSeal    messages.BlockSeal
	findErr     error
	containsErr error
	futureErr   error
}

// TestReadsNotBlockedBySealBlock pins that reads answer from the last
// published seal while a SealBlock durable write is in flight.
func TestReadsNotBlockedBySealBlock(t *testing.T) {
	t.Parallel()

	db := tempDB(t)
	blk1 := sealRange(t, db, blockID(0, 0xA0), 1, 1)

	g := &gatedWAL{walStore: db.w, started: make(chan struct{}), gate: make(chan struct{})}
	db.w = g

	sealDone := make(chan error, 1)
	go func() {
		sealDone <- db.SealBlock(blk1.Hash, blockID(2, 0x22), 200)
	}()
	<-g.started

	results := make(chan readResults, 1)
	go func() {
		var r readResults
		r.latest, r.latestOK = db.LatestSealedBlock()
		r.findSeal, r.findErr = db.FindSealedBlock(1)
		_, r.containsErr = db.Contains(messages.ContainsQuery{BlockNum: 1, Timestamp: 100})
		_, r.futureErr = db.Contains(messages.ContainsQuery{BlockNum: 3, Timestamp: 300})
		results <- r
	}()

	var r readResults
	select {
	case r = <-results:
	case <-time.After(10 * time.Second):
		t.Error("reads blocked behind in-flight SealBlock")
	}

	close(g.gate)
	require.NoError(t, <-sealDone)

	if t.Failed() {
		return
	}
	require.True(t, r.latestOK)
	require.Equal(t, blk1, r.latest)
	require.NoError(t, r.findErr)
	require.Equal(t, blk1.Hash, r.findSeal.Hash)
	// Block 1 was sealed with no logs, so LogIdx 0 is out of range.
	require.ErrorIs(t, r.containsErr, interop.ErrConflict)
	// Block 3 is beyond the published seal.
	require.ErrorIs(t, r.futureErr, interop.ErrFuture)

	latest, ok := db.LatestSealedBlock()
	require.True(t, ok)
	require.Equal(t, blockID(2, 0x22), latest)
}
