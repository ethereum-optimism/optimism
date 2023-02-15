package derive

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)
const (
	// SYSCOIN
	appendSequencerBatchMethodName = "appendSequencerBatch()"
)
type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, types.Transactions, error)
	GetBlobFromCloud(vh common.Hash) ([]byte, error)
	GetBlobFromRPC(vh common.Hash) ([]byte, error)
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
	_, receipts, txs, err := fetcher.FetchReceipts(ctx, block.Hash)
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
			// SYSCOIN
			data: DataFromEVMTransactions(ctx, fetcher, cfg, batcherAddr, receipts, txs, log.New("origin", block)),
		}
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *DataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		// SYSCOIN
		if _, receipts, txs, err := ds.fetcher.FetchReceipts(ctx, ds.id.Hash); err == nil {
			ds.open = true
			ds.data = DataFromEVMTransactions(ctx, ds.fetcher, ds.cfg, ds.batcherAddr, receipts, txs, log.New("origin", ds.id))
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

// SYSCOIN DataFromEVMTransactions filters all of the transactions and returns the calldata from transactions
// that are sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
func DataFromEVMTransactions(ctx context.Context, fetcher L1TransactionFetcher, config *rollup.Config, batcherAddr common.Address, receipts types.Receipts, txs types.Transactions, log log.Logger) []eth.Data {
	var out []eth.Data
	l1Signer := config.L1Signer()
	for i, receipt := range receipts {
		if to := txs[i].To(); to == nil || *to != config.BatchInboxAddress {
			continue
		}
		if(receipt.Status != types.ReceiptStatusSuccessful) {
			log.Warn("DataFromEVMTransactions: transaction was not successful", "index", i, "status", receipt.Status)
			continue // reverted, ignore
		}
		seqDataSubmitter, err := l1Signer.Sender(txs[i]) // optimization: only derive sender if To is correct
		if err != nil {
			log.Warn("tx in inbox with invalid signature", "index", i, "err", err)
			continue // bad signature, ignore
		}
		// some random L1 user might have sent a transaction to our batch inbox, ignore them
		if seqDataSubmitter != batcherAddr {
			log.Warn("tx in inbox with unauthorized submitter", "index", i, "err", err)
			continue // not an authorized batch submitter, ignore
		}
		calldata := txs[receipt.TransactionIndex].Data()
		// remove function hash
		sig := crypto.Keccak256([]byte(appendSequencerBatchMethodName))[:4]
		sigToCheck := calldata[:4]
		if (!reflect.DeepEqual(sig, sigToCheck)) {
			log.Warn("DataFromEVMTransactions", "append function not found as method signature")
			continue
		}
		calldata = calldata[4:]
		lenData := len(calldata)
		if (lenData%32) != 0 {
			log.Warn("DataFromEVMTransactions", "Invalid length of calldata, not mod of 32", len(calldata))
			continue
		}
		numVHs := lenData/32
		for i := 0; i < numVHs; i++ {
			// get version hash from calldata and lookup data via syscoinclient
			vhBytes := calldata[i*32:(i+1)*32]
			// 1. get data from syscoin rpc
			vh := common.BytesToHash(vhBytes)
			data, err := fetcher.GetBlobFromRPC(vh)
			if err != nil {
				// 2. if not get it from archiving service
				data, err = fetcher.GetBlobFromCloud(vh)
				if err != nil {
					log.Warn("DataFromEVMTransactions", "failed to fetch L1 block info and receipts", err)
					continue
				}
				// check data is valid locally
				vhData := crypto.Keccak256Hash(data)
				if vh != vhData {
					log.Warn("DataFromEVMTransactions", "blob data hash mismatch want", vh, "have", vhData)
					continue
				}
				log.Warn("GetBlobFromCloud", "len", len(data), "vh", vh)
			} else {
				log.Warn("GetBlobFromRPC", "len", len(data), "vh", vh)
			}
			out = append(out, data)
		}
	}
	return out
}
