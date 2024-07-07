package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

type L1BlobsFetcher interface {
	// GetBlobs fetches blobs that were confirmed in the given L1 block with the given indexed hashes.
	GetBlobs(ctx context.Context, ref eth.L1BlockRef, hashes []eth.IndexedBlobHash) ([]*eth.Blob, error)
}

type PlasmaInputFetcher interface {
	// GetInput fetches the input for the given commitment at the given block number from the DA storage service.
	GetInput(ctx context.Context, l1 plasma.L1Fetcher, c plasma.CommitmentData, blockId eth.L1BlockRef) (eth.Data, error)
	// AdvanceL1Origin advances the L1 origin to the given block number, syncing the DA challenge events.
	AdvanceL1Origin(ctx context.Context, l1 plasma.L1Fetcher, blockId eth.BlockID) error
	// Reset the challenge origin in case of L1 reorg
	Reset(ctx context.Context, base eth.L1BlockRef, baseCfg eth.SystemConfig) error
}

// DataSourceFactory reads raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log           log.Logger
	dsCfg         DataSourceConfig
	fetcher       L1Fetcher
	blobsFetcher  L1BlobsFetcher
	plasmaFetcher PlasmaInputFetcher
	ecotoneTime   *uint64
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, fetcher L1Fetcher, blobsFetcher L1BlobsFetcher, plasmaFetcher PlasmaInputFetcher) *DataSourceFactory {
	config := DataSourceConfig{
		l1Signer:          cfg.L1Signer(),
		batchInboxAddress: cfg.BatchInboxAddress,
		plasmaEnabled:     cfg.PlasmaEnabled(),
	}
	return &DataSourceFactory{
		log:           log,
		dsCfg:         config,
		fetcher:       fetcher,
		blobsFetcher:  blobsFetcher,
		plasmaFetcher: plasmaFetcher,
		ecotoneTime:   cfg.EcotoneTime,
	}
}

// OpenData returns the appropriate data source for the L1 block `ref`.
func (ds *DataSourceFactory) OpenData(ctx context.Context, ref eth.L1BlockRef, batcherAddr common.Address) (DataIter, error) {
	// Creates a data iterator from blob or calldata source so we can forward it to the plasma source
	// if enabled as it still requires an L1 data source for fetching input commmitments.
	var src DataIter
	if ds.ecotoneTime != nil && ref.Time >= *ds.ecotoneTime {
		if ds.blobsFetcher == nil {
			return nil, fmt.Errorf("ecotone upgrade active but beacon endpoint not configured")
		}
		src = NewBlobDataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ds.blobsFetcher, ref, batcherAddr)
	} else {
		src = NewCalldataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ref, batcherAddr)
	}
	if ds.dsCfg.plasmaEnabled {
		// plasma([calldata | blobdata](l1Ref)) -> data
		return NewPlasmaDataSource(ds.log, src, ds.fetcher, ds.plasmaFetcher, ref), nil
	}
	return src, nil
}

// DataSourceConfig regroups the mandatory rollup.Config fields needed for DataFromEVMTransactions.
type DataSourceConfig struct {
	l1Signer          types.Signer
	batchInboxAddress common.Address
	plasmaEnabled     bool
}

// isValidBatchTx returns true if:
//  1. the transaction has a To() address that matches the batch inbox address, and
//  2. the transaction has a valid signature from the batcher address
func isValidBatchTx(tx *types.Transaction, l1Signer types.Signer, batchInboxAddr, batcherAddr common.Address) bool {
	to := tx.To()
	if to == nil || *to != batchInboxAddr {
		return false
	}
	seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
	if err != nil {
		log.Warn("tx in inbox with invalid signature", "hash", tx.Hash(), "err", err)
		return false
	}
	// some random L1 user might have sent a transaction to our batch inbox, ignore them
	if seqDataSubmitter != batcherAddr {
		log.Warn("tx in inbox with unauthorized submitter", "addr", seqDataSubmitter, "hash", tx.Hash(), "err", err)
		return false
	}
	return true
}
