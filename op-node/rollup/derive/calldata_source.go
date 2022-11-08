package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
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
	FetchReceiptsFromTxs(ctx context.Context, txs types.Transactions, info eth.BlockInfo, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	GetBlobFromCloud(vh common.Hash) (string, error)
	GetBlobFromRPC(vh common.Hash) (string, error)
}

// DataSourceFactory readers raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log     log.Logger
	cfg     *rollup.Config
	fetcher L1TransactionFetcher
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher) *DataSourceFactory {
	return &DataSourceFactory{log: log, cfg: cfg, fetcher: fetcher}
}

// OpenData returns a CalldataSourceImpl. This struct implements the `Next` function.
func (ds *DataSourceFactory) OpenData(ctx context.Context, id eth.BlockID, batcherAddr common.Address) DataIter {
	return NewDataSource(ctx, ds.log, ds.cfg, ds.fetcher, id, batcherAddr)
}

// DataSource is a fault tolerant approach to fetching data.
// The constructor will never fail & it will instead re-attempt the fetcher
// at a later point.
type DataSource struct {
	// Internal state + data
	open bool
	data []eth.Data
	// Required to re-attempt fetching
	id      eth.BlockID
	cfg     *rollup.Config // TODO: `DataFromEVMTransactions` should probably not take the full config
	fetcher L1TransactionFetcher
	log     log.Logger

	batcherAddr common.Address
}

// NewDataSource creates a new calldata source. It suppresses errors in fetching the L1 block if they occur.
// If there is an error, it will attempt to fetch the result on the next call to `Next`.
func NewDataSource(ctx context.Context, log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher, block eth.BlockID, batcherAddr common.Address) DataIter {
	// SYSCOIN info
	info, txs, err := fetcher.InfoAndTxsByHash(ctx, block.Hash)
	if err != nil {
		return &DataSource{
			open:        false,
			id:          block,
			cfg:         cfg,
			fetcher:     fetcher,
			log:         log,
			batcherAddr: batcherAddr,
		}
	} else {
		return &DataSource{
			open: true,
			data: DataFromEVMTransactions(ctx, fetcher, info, block.Hash, cfg, txs, log.New("origin", block)),
		}
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *DataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if info, txs, err := ds.fetcher.InfoAndTxsByHash(ctx, ds.id.Hash); err == nil {
			ds.open = true
			ds.data = DataFromEVMTransactions(ctx, ds.fetcher, info, ds.id.Hash, ds.cfg, txs, log.New("origin", ds.id))
		} else if errors.Is(err, ethereum.NotFound) {
			return nil, NewResetError(fmt.Errorf("failed to open calldata source: %w", err))
		} else {
			return nil, NewTemporaryError(fmt.Errorf("failed to open calldata source: %w", err))
		}
	}
	if len(ds.data) == 0 {
		return nil, io.EOF
	} else {
		data := ds.data[0]
		ds.data = ds.data[1:]
		return data, nil
	}
}

// DataFromEVMTransactions filters all of the transactions and returns the calldata from transactions
// that are sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
func DataFromEVMTransactions(ctx context.Context, fetcher L1TransactionFetcher, info eth.BlockInfo, blockHash common.Hash, config *rollup.Config, txs types.Transactions, log log.Logger) []eth.Data {
	var out []eth.Data
	var txsToCheck types.Transactions
	for _, tx := range txs {
		if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
			/*seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				log.Warn("tx in inbox with invalid signature", "index", j, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != batcherAddr {
				log.Warn("tx in inbox with unauthorized submitter", "index", j, "err", err)
				continue // not an authorized batch submitter, ignore
			}*/
			txsToCheck = append(txsToCheck, tx)
		}
	}
	_, receipts, err := fetcher.FetchReceiptsFromTxs(ctx, txsToCheck, info, blockHash)
	if err != nil {
		log.Warn("DataFromEVMTransactions", "failed to fetch L1 block info and receipts", err)
		return nil
	}
	for i, receipt := range receipts {
		if(receipt.Status != types.ReceiptStatusSuccessful) {
			log.Warn("DataFromEVMTransactions: transaction was not successful", "index", i, "status", receipt.Status)
			continue // reverted, ignore
		}
		// get version hash from calldata and lookup data via syscoinclient
		// get calldata, break it down into array of VH's
		// 1. get data from syscoin rpc
		// 2. if not get it from archiving service
		// 2a. validate the data against the kzg commitment
		out = append(out, txs[receipt.TransactionIndex].Data())
	}
	return out
}
