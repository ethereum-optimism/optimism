package backend

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-service/sources/caching"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db"
)

type Metrics interface {
	CacheAdd(chainID *big.Int, label string, cacheSize int, evicted bool)
	CacheGet(chainID *big.Int, label string, hit bool)

	RecordDBEntryCount(chainID *big.Int, count int64)
	RecordDBSearchEntriesRead(chainID *big.Int, count int64)
}

// chainMetrics is an adapter between the metrics API expected by clients that assume there's only a single chain
// and the actual metrics implementation which requires a chain ID to identify the source chain.
type chainMetrics struct {
	chainID  *big.Int
	delegate Metrics
}

func newChainMetrics(chainID *big.Int, delegate Metrics) *chainMetrics {
	return &chainMetrics{
		chainID:  chainID,
		delegate: delegate,
	}
}

func (c *chainMetrics) CacheAdd(label string, cacheSize int, evicted bool) {
	c.delegate.CacheAdd(c.chainID, label, cacheSize, evicted)
}

func (c *chainMetrics) CacheGet(label string, hit bool) {
	c.delegate.CacheGet(c.chainID, label, hit)
}

func (c *chainMetrics) RecordDBEntryCount(count int64) {
	c.delegate.RecordDBEntryCount(c.chainID, count)
}

func (c *chainMetrics) RecordDBSearchEntriesRead(count int64) {
	c.delegate.RecordDBSearchEntriesRead(c.chainID, count)
}

var _ caching.Metrics = (*chainMetrics)(nil)
var _ db.Metrics = (*chainMetrics)(nil)
