package metered

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1FetcherMetrics interface {
	RecordL1RequestTime(method string, duration time.Duration)
}

type L1Fetcher interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
	L1BlockRefByHash(context.Context, common.Hash) (eth.L1BlockRef, error)
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, optypes.Receipts, error)
	InfoAndRawTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, []hexutil.Bytes, error)
}

type MeteredL1Fetcher struct {
	inner   L1Fetcher
	metrics L1FetcherMetrics
	now     func() time.Time
}

var _ L1Fetcher = (*MeteredL1Fetcher)(nil)

func NewMeteredL1Fetcher(inner L1Fetcher, metrics L1FetcherMetrics) *MeteredL1Fetcher {
	return &MeteredL1Fetcher{
		inner:   inner,
		metrics: metrics,
		now:     time.Now,
	}
}
func (m *MeteredL1Fetcher) L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error) {
	defer m.recordTime("L1BlockRefByLabel")()
	return m.inner.L1BlockRefByLabel(ctx, label)
}

func (m *MeteredL1Fetcher) L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error) {
	defer m.recordTime("L1BlockRefByNumber")()
	return m.inner.L1BlockRefByNumber(ctx, num)
}

func (m *MeteredL1Fetcher) L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error) {
	defer m.recordTime("L1BlockRefByHash")()
	return m.inner.L1BlockRefByHash(ctx, hash)
}

func (m *MeteredL1Fetcher) InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	defer m.recordTime("InfoByHash")()
	return m.inner.InfoByHash(ctx, hash)
}

func (m *MeteredL1Fetcher) InfoAndRawTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, []hexutil.Bytes, error) {
	defer m.recordTime("InfoAndRawTxsByHash")()
	return m.inner.InfoAndRawTxsByHash(ctx, hash)
}

func (m *MeteredL1Fetcher) FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, optypes.Receipts, error) {
	defer m.recordTime("FetchReceipts")()
	return m.inner.FetchReceipts(ctx, blockHash)
}

func (m *MeteredL1Fetcher) recordTime(method string) func() {
	start := m.now()
	return func() {
		end := m.now()
		m.metrics.RecordL1RequestTime(method, end.Sub(start))
	}
}
