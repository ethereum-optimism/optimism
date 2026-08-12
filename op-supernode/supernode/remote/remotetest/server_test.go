package remotetest_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote/remotetest"
)

// TestServerServesProtocol drives the test server through a real remote.HTTPAdapter and
// confirms the round trip: contiguous, linked blocks; blockTime is reported; and a
// bounded Head produces the "nothing new yet" (ok=false) response.
func TestServerServesProtocol(t *testing.T) {
	chain := remotetest.New(remotetest.Config{
		ChainID:        eth.ChainIDFromUInt64(8453),
		BlockTime:      2,
		MsgsPerBlock:   3,
		StartTimestamp: 1000,
		Head:           2,
	})
	srv := httptest.NewServer(chain.Handler())
	defer srv.Close()

	adapter := remote.NewHTTPAdapter(eth.ChainIDFromUInt64(8453), srv.URL, srv.Client())

	b1, ok, err := adapter.NextFinalized(context.Background(), 0)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, uint64(1), b1.Number)
	require.Equal(t, uint64(1002), b1.Timestamp)
	require.Len(t, b1.Messages, 3)
	require.Equal(t, uint64(2), adapter.BlockTime(), "blockTime should be cached from the response")

	b2, ok, err := adapter.NextFinalized(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, b1.Hash, b2.ParentHash, "block 2 must link to block 1")

	// Head == 2, so there is no block 3 yet.
	_, ok, err = adapter.NextFinalized(context.Background(), 2)
	require.NoError(t, err)
	require.False(t, ok)
}

// TestServerAnchorsAtFirstBlock covers the anchor request against a chain whose history
// starts far past genesis, the way a live public chain's does: after=0 gets FirstBlock,
// and everything from there is strictly contiguous.
func TestServerAnchorsAtFirstBlock(t *testing.T) {
	const first = uint64(47_300_000)
	chainID := eth.ChainIDFromUInt64(11155420)
	chain := remotetest.New(remotetest.Config{
		ChainID:        chainID,
		BlockTime:      2,
		MsgsPerBlock:   1,
		StartTimestamp: 1000,
		FirstBlock:     first,
	})
	srv := httptest.NewServer(chain.Handler())
	defer srv.Close()

	adapter := remote.NewHTTPAdapter(chainID, srv.URL, srv.Client())

	anchor, ok, err := adapter.NextFinalized(context.Background(), 0)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, first, anchor.Number, "after=0 must be answered with the anchor")
	require.Equal(t, uint64(1002), anchor.Timestamp, "the anchor sits one block past StartTimestamp")
	require.NotEqual(t, common.Hash{}, anchor.ParentHash)

	next, ok, err := adapter.NextFinalized(context.Background(), first)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, first+1, next.Number)
	require.Equal(t, anchor.Hash, next.ParentHash, "blocks after the anchor stay linked")

	// Nothing below the anchor is served.
	_, ok, err = adapter.NextFinalized(context.Background(), first-10)
	require.NoError(t, err)
	require.False(t, ok)
}
