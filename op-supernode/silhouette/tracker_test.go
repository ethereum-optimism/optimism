package silhouette

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestTrackerRewindsWhenL1ShortensBehindCursor(t *testing.T) {
	l1 := newFakeL1(l1GenesisNum + 5)
	tracker := NewProvenHeadTracker(log.New(), nil, l1, l1GenesisNum, 0)
	tracker.next = l1GenesisNum + 6
	for n := tracker.start; n < tracker.next; n++ {
		tracker.processed[n] = l1.hashOf(n)
	}

	// The replacement L1 is shorter and forks after genesis+1. The newest common block is +1,
	// so replay starts at +2 even though that block has not been produced on the new fork yet.
	l1.head = l1GenesisNum + 3
	l1.reorgAbove = l1GenesisNum + 1
	rewound, err := tracker.reconcileShortenedL1(context.Background())
	require.NoError(t, err)
	require.True(t, rewound)
	require.Equal(t, l1GenesisNum+2, tracker.Cursor())
	require.NotContains(t, tracker.processed, l1GenesisNum+2)
}

func TestTrackerDoesNotRewindAtOrdinaryTip(t *testing.T) {
	l1 := newFakeL1(l1GenesisNum + 2)
	tracker := NewProvenHeadTracker(log.New(), nil, l1, l1GenesisNum, 0)
	tracker.next = l1GenesisNum + 3
	for n := tracker.start; n < tracker.next; n++ {
		tracker.processed[n] = l1.hashOf(n)
	}

	rewound, err := tracker.reconcileShortenedL1(context.Background())
	require.NoError(t, err)
	require.False(t, rewound)
	require.Equal(t, l1GenesisNum+3, tracker.Cursor())
}
