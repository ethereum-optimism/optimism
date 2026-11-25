package resources

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var testInbox = common.HexToAddress("0xff00000000000000000000000000000000000901")

type fakeL1 struct {
	rollupNode.L1Source
	headers map[uint64]*types.Header
	txs     map[common.Hash]types.Transactions
}

func (f *fakeL1) L1BlockRefByNumber(_ context.Context, n uint64) (eth.L1BlockRef, error) {
	h, ok := f.headers[n]
	if !ok {
		return eth.L1BlockRef{}, context.DeadlineExceeded
	}
	return eth.InfoToL1BlockRef(eth.HeaderBlockInfo(h)), nil
}

func (f *fakeL1) InfoAndTxsByHash(_ context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	for _, h := range f.headers {
		if h.Hash() == hash {
			return eth.HeaderBlockInfo(h), f.txs[hash], nil
		}
	}
	return nil, nil, context.DeadlineExceeded
}

type fakeBeacon struct {
	mu    sync.Mutex
	calls map[uint64]int
}

func (b *fakeBeacon) GetBlobsByHash(_ context.Context, t uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	b.mu.Lock()
	b.calls[t]++
	b.mu.Unlock()
	out := make([]*eth.Blob, len(hashes))
	for i, h := range hashes {
		out[i] = &eth.Blob{}
		copy(out[i][:], h[:])
	}
	return out, nil
}

func (b *fakeBeacon) count(t uint64) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.calls[t]
}

// newFakeChain builds L1 blocks 0..n-1. Every block has a blob tx to another address,
// which must be ignored; blobBlocks also get one blob tx to the batch inbox.
func newFakeChain(n uint64, blobBlocks ...uint64) (*fakeL1, map[uint64][]common.Hash) {
	l1 := &fakeL1{headers: map[uint64]*types.Header{}, txs: map[common.Hash]types.Transactions{}}
	want := map[uint64][]common.Hash{}
	other := common.HexToAddress("0x1234")
	for i := uint64(0); i < n; i++ {
		h := &types.Header{Number: new(big.Int).SetUint64(i), Time: 1000 + 12*i, Difficulty: common.Big0}
		txs := types.Transactions{types.NewTx(&types.BlobTx{To: other, BlobHashes: []common.Hash{{0xee, byte(i)}}})}
		for _, b := range blobBlocks {
			if b == i {
				bh := common.Hash{0x01, byte(i)}
				txs = append(txs, types.NewTx(&types.BlobTx{To: testInbox, BlobHashes: []common.Hash{bh}}))
				want[i] = []common.Hash{bh}
			}
		}
		l1.headers[i] = h
		l1.txs[h.Hash()] = txs
	}
	return l1, want
}

func TestBlobPrefetcherServesReadAhead(t *testing.T) {
	l1, want := newFakeChain(10, 3)
	beacon := &fakeBeacon{calls: map[uint64]int{}}
	m := NewSupernodeMetrics()
	p := NewBlobPrefetcher(testlog.Logger(t, log.LevelInfo), l1, beacon, testInbox, 4, m)

	// Derivation opens block 1: blocks 2..5 are prefetched, including block 3's blobs.
	_, _, err := p.InfoAndTxsByHash(context.Background(), l1.headers[1].Hash())
	require.NoError(t, err)
	t3 := l1.headers[3].Time
	require.Eventually(t, func() bool { return beacon.count(t3) == 1 }, 5*time.Second, 10*time.Millisecond)

	blobs, err := p.GetBlobsByHash(context.Background(), t3, want[3])
	require.NoError(t, err)
	require.Len(t, blobs, 1)
	require.Equal(t, want[3][0][:], blobs[0][:32])
	require.Equal(t, 1, beacon.count(t3), "served from the read-ahead cache, not fetched again")
	require.Equal(t, 1.0, counterValue(t, m.BlobPrefetchLookups.WithLabelValues("hit")))

	// Blocks without inbox blobs are never fetched from the beacon.
	for _, n := range []uint64{2, 4, 5} {
		require.Zero(t, beacon.count(l1.headers[n].Time))
	}
}

func TestBlobPrefetcherMissFallsThrough(t *testing.T) {
	l1, _ := newFakeChain(4)
	beacon := &fakeBeacon{calls: map[uint64]int{}}
	m := NewSupernodeMetrics()
	p := NewBlobPrefetcher(testlog.Logger(t, log.LevelInfo), l1, beacon, testInbox, 2, m)

	_, err := p.GetBlobsByHash(context.Background(), 5000, []common.Hash{{0xab}})
	require.NoError(t, err)
	require.Equal(t, 1, beacon.count(5000))
	require.Equal(t, 1.0, counterValue(t, m.BlobPrefetchLookups.WithLabelValues("miss")))
}

func TestBlobPrefetcherRestartsWindowAfterReset(t *testing.T) {
	l1, _ := newFakeChain(40, 2)
	beacon := &fakeBeacon{calls: map[uint64]int{}}
	p := NewBlobPrefetcher(testlog.Logger(t, log.LevelInfo), l1, beacon, testInbox, 3, nil)

	_, _, err := p.InfoAndTxsByHash(context.Background(), l1.headers[30].Hash())
	require.NoError(t, err)
	// Derivation resets back to block 1: block 2's blobs must be prefetched again.
	_, _, err = p.InfoAndTxsByHash(context.Background(), l1.headers[1].Hash())
	require.NoError(t, err)
	require.Eventually(t, func() bool { return beacon.count(l1.headers[2].Time) == 1 }, 5*time.Second, 10*time.Millisecond)
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	var m dto.Metric
	require.NoError(t, c.Write(&m))
	return m.GetCounter().GetValue()
}
