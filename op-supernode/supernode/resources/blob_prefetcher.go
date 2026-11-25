package resources

import (
	"context"
	"sync"
	"time"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	lru "github.com/hashicorp/golang-lru/v2"

	rollupNode "github.com/ethereum-optimism/optimism/op-node/node"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const (
	blobPrefetchCacheSize   = 512 // blobs, 128 KiB each
	blobPrefetchConcurrency = 4
	blobPrefetchTimeout     = time.Minute
)

// BlobPrefetcher reads ahead the batch-inbox blobs of the L1 blocks that one chain's
// derivation will open next. Fetching blobs is latency-bound (about a second per request
// on a PeerDAS beacon node), and derivation fetches them one L1 block at a time.
//
// It changes no derivation input: prefetched blobs come from the same beacon client, which
// checks each blob against its versioned hash, and are only served for an exact versioned
// hash match. A prefetch of a block that later reorgs out only wastes the fetch.
type BlobPrefetcher struct {
	rollupNode.L1Source

	beacon  rollupNode.L1Beacon
	inbox   common.Address
	depth   uint64
	log     oplog.Logger
	metrics *SupernodeMetrics

	blobs *lru.Cache[common.Hash, *eth.Blob]
	sem   chan struct{}

	mu        sync.Mutex
	scheduled uint64                   // highest L1 block number scheduled for prefetch
	inflight  map[uint64]chan struct{} // L1 block time -> closed when its prefetch ends
}

var (
	_ rollupNode.L1Source = (*BlobPrefetcher)(nil)
	_ rollupNode.L1Beacon = (*BlobPrefetcher)(nil)
)

// NewBlobPrefetcher wraps a chain's L1 source and beacon client. inbox is the chain's
// batch inbox address; depth is how many L1 blocks ahead of derivation to prefetch.
func NewBlobPrefetcher(logger oplog.Logger, l1 rollupNode.L1Source, beacon rollupNode.L1Beacon, inbox common.Address, depth uint64, m *SupernodeMetrics) *BlobPrefetcher {
	cache, _ := lru.New[common.Hash, *eth.Blob](blobPrefetchCacheSize)
	if m == nil {
		m = NewSupernodeMetrics()
	}
	return &BlobPrefetcher{
		L1Source: l1,
		beacon:   beacon,
		inbox:    inbox,
		depth:    depth,
		log:      logger,
		metrics:  m,
		blobs:    cache,
		sem:      make(chan struct{}, blobPrefetchConcurrency),
		inflight: make(map[uint64]chan struct{}),
	}
}

// InfoAndTxsByHash is how blob derivation opens an L1 block. Opening block n schedules
// blocks n+1..n+depth for prefetch.
func (p *BlobPrefetcher) InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error) {
	info, txs, err := p.L1Source.InfoAndTxsByHash(ctx, hash)
	if err == nil {
		p.schedule(info.NumberU64())
	}
	return info, txs, err
}

// GetBlobsByHash serves blobs from the prefetch cache when every requested hash is
// present, waiting for an in-flight prefetch of the same block first. Otherwise it
// fetches from the beacon client as before.
func (p *BlobPrefetcher) GetBlobsByHash(ctx context.Context, blockTime uint64, hashes []common.Hash) ([]*eth.Blob, error) {
	p.mu.Lock()
	ch, waiting := p.inflight[blockTime]
	p.mu.Unlock()
	if waiting {
		select {
		case <-ch:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if blobs, ok := p.cached(hashes); ok {
		p.metrics.BlobPrefetchLookups.WithLabelValues("hit").Inc()
		return blobs, nil
	}
	p.metrics.BlobPrefetchLookups.WithLabelValues("miss").Inc()
	return p.beacon.GetBlobsByHash(ctx, blockTime, hashes)
}

func (p *BlobPrefetcher) cached(hashes []common.Hash) ([]*eth.Blob, bool) {
	out := make([]*eth.Blob, len(hashes))
	for i, h := range hashes {
		b, ok := p.blobs.Get(h)
		if !ok {
			return nil, false
		}
		out[i] = b
	}
	return out, true
}

func (p *BlobPrefetcher) schedule(opened uint64) {
	p.mu.Lock()
	// Derivation went back (pipeline reset or reorg): restart the window from here.
	if p.scheduled > opened+p.depth {
		p.scheduled = opened
	}
	from := max(p.scheduled, opened) + 1
	to := opened + p.depth
	if from > to {
		p.mu.Unlock()
		return
	}
	p.scheduled = to
	p.mu.Unlock()
	for num := from; num <= to; num++ {
		go p.prefetch(num)
	}
}

func (p *BlobPrefetcher) prefetch(num uint64) {
	p.sem <- struct{}{}
	defer func() { <-p.sem }()
	ctx, cancel := context.WithTimeout(context.Background(), blobPrefetchTimeout)
	defer cancel()

	ref, err := p.L1Source.L1BlockRefByNumber(ctx, num)
	if err != nil {
		return // not yet on L1, or transient; derivation fetches it itself
	}
	_, txs, err := p.L1Source.InfoAndTxsByHash(ctx, ref.Hash)
	if err != nil {
		return
	}
	var hashes []common.Hash
	for _, tx := range txs {
		if tx.Type() == types.BlobTxType && tx.To() != nil && *tx.To() == p.inbox {
			hashes = append(hashes, tx.BlobHashes()...)
		}
	}
	if len(hashes) == 0 {
		return
	}
	if _, ok := p.cached(hashes); ok {
		return
	}

	done := make(chan struct{})
	p.mu.Lock()
	if _, busy := p.inflight[ref.Time]; busy {
		p.mu.Unlock()
		return
	}
	p.inflight[ref.Time] = done
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.inflight, ref.Time)
		p.mu.Unlock()
		close(done)
	}()

	blobs, err := p.beacon.GetBlobsByHash(ctx, ref.Time, hashes)
	if err != nil {
		p.log.Debug("blob prefetch failed", "l1", ref, "err", err)
		return
	}
	for i, h := range hashes {
		p.blobs.Add(h, blobs[i])
	}
}
