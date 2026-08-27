package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

type L1BlobsFetcher interface {
	// GetBlobsByHash fetches blobs that were confirmed at the given timestamp with the given versioned hashes.
	GetBlobsByHash(ctx context.Context, time uint64, hashes []common.Hash) ([]*eth.Blob, error)
}

// DataSourceFactory reads raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log          log.Logger
	dsCfg        DataSourceConfig
	fetcher      L1Fetcher
	blobsFetcher L1BlobsFetcher
	ecotoneTime  *uint64
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, fetcher L1Fetcher, blobsFetcher L1BlobsFetcher) *DataSourceFactory {
	config := DataSourceConfig{
		l1Signer:          cfg.L1Signer(),
		batchInboxAddress: cfg.BatchInboxAddress,
	}
	return &DataSourceFactory{
		log:          log,
		dsCfg:        config,
		fetcher:      fetcher,
		blobsFetcher: blobsFetcher,
		ecotoneTime:  cfg.EcotoneTime,
	}
}

// OpenData returns the appropriate data source for the L1 block `ref`.
func (ds *DataSourceFactory) OpenData(ctx context.Context, ref eth.L1BlockRef, batcherAddr common.Address) (DataIter, error) {
	if ds.ecotoneTime != nil && ref.Time >= *ds.ecotoneTime {
		if ds.blobsFetcher == nil {
			return nil, NewCriticalError(fmt.Errorf("ecotone upgrade active but beacon endpoint not configured"))
		}
		return NewBlobDataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ds.blobsFetcher, ref, batcherAddr), nil
	}
	return NewCalldataSource(ctx, ds.log, ds.dsCfg, ds.fetcher, ref, batcherAddr), nil
}

// DataSourceConfig regroups the mandatory rollup.Config fields needed for DataFromEVMTransactions.
type DataSourceConfig struct {
	l1Signer          types.Signer
	batchInboxAddress common.Address
}

// isValidBatchTx returns true if:
//  1. the transaction type is any of Legacy, ACL, DynamicFee, Blob, or Deposit (for L3s).
//  2. the transaction has a To() address that matches the batch inbox address, and
//  3. the transaction has a valid signature from the batcher address
func isValidBatchTx(tx *types.Transaction, l1Signer types.Signer, batchInboxAddr, batcherAddr common.Address, logger log.Logger) bool {
	// For now, we want to disallow the SetCodeTx type or any future types.
	if tx.Type() > types.BlobTxType && tx.Type() != optypes.DepositTxType {
		return false
	}

	to := tx.To()
	if to == nil || *to != batchInboxAddr {
		return false
	}
	seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
	if err != nil {
		logger.Warn("tx in inbox with invalid signature", "hash", tx.Hash(), "err", err)
		return false
	}
	// some random L1 user might have sent a transaction to our batch inbox, ignore them
	if seqDataSubmitter != batcherAddr {
		logger.Warn("tx in inbox with unauthorized submitter", "addr", seqDataSubmitter, "hash", tx.Hash(), "err", err)
		return false
	}
	return true
}
