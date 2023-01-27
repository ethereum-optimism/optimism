package derive

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

type IndexedDataHash struct {
	Index    uint64      // absolute index in the block, a.k.a. index in sidecar
	DataHash common.Hash // hash of the blob, used for consistency checks
	// Might add tx index and/or tx hash here later, depending on blobs API design
}

type L1BlobsFetcher interface {
	// BlobsByRefAndIndexedDatahashes fetches blobs that were confirmed in the given L1 block at the given index.
	BlobsByRefAndIndexedDatahashes(ctx context.Context, ref eth.L1BlockRef, dataHashes []IndexedDataHash) ([]types.Blob, error)
}

// DataSourceFactory readers raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log          log.Logger
	cfg          *rollup.Config
	fetcher      L1TransactionFetcher
	blobsFetcher L1BlobsFetcher
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher, blobsFetcher L1BlobsFetcher) *DataSourceFactory {
	return &DataSourceFactory{log: log, cfg: cfg, fetcher: fetcher, blobsFetcher: blobsFetcher}
}

// OpenData returns a CalldataSourceImpl. This struct implements the `Next` function.
func (ds *DataSourceFactory) OpenData(ctx context.Context, ref eth.L1BlockRef, batcherAddr common.Address) DataIter {
	if n := ds.cfg.BlobsEnabledL1Timestamp; n != nil && *n <= ref.Time {
		return NewBlobDataSource(ctx, ds.log, ds.cfg, ds.fetcher, ds.blobsFetcher, ref, batcherAddr)
	}
	return NewCallDataSource(ctx, ds.log, ds.cfg, ds.fetcher, ref, batcherAddr)
}
