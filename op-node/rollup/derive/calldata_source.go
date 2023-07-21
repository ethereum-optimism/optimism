package derive

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/rollkit/celestia-openrpc/types/share"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

type DataIter interface {
	Next(ctx context.Context) (eth.Data, error)
}

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

// DataSourceFactory readers raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline
type DataSourceFactory struct {
	log     log.Logger
	cfg     *rollup.Config
	daCfg   *rollup.DAConfig
	fetcher L1TransactionFetcher
}

func NewDataSourceFactory(log log.Logger, cfg *rollup.Config, daCfg *rollup.DAConfig, fetcher L1TransactionFetcher) *DataSourceFactory {
	return &DataSourceFactory{log: log, cfg: cfg, daCfg: daCfg, fetcher: fetcher}
}

// OpenData returns a DataIter. This struct implements the `Next` function.
func (ds *DataSourceFactory) OpenData(ctx context.Context, id eth.BlockID, batcherAddr common.Address) (DataIter, error) {
	return NewDataSource(ctx, ds.log, ds.cfg, ds.daCfg, ds.fetcher, id, batcherAddr)
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
	daCfg   *rollup.DAConfig
	fetcher L1TransactionFetcher
	log     log.Logger

	batcherAddr common.Address
}

// NewDataSource creates a new calldata source. It suppresses errors in fetching the L1 block if they occur.
// If there is an error, it will attempt to fetch the result on the next call to `Next`.
func NewDataSource(ctx context.Context, log log.Logger, cfg *rollup.Config, daCfg *rollup.DAConfig, fetcher L1TransactionFetcher, block eth.BlockID, batcherAddr common.Address) (DataIter, error) {
	_, txs, err := fetcher.InfoAndTxsByHash(ctx, block.Hash)
	if err != nil {
		return &DataSource{
			open:        false,
			id:          block,
			cfg:         cfg,
			fetcher:     fetcher,
			log:         log,
			batcherAddr: batcherAddr,
		}, nil
	} else {
		data, err := DataFromEVMTransactions(cfg, daCfg, batcherAddr, txs, log.New("origin", block))
		if err != nil {
			return &DataSource{
				open:        false,
				id:          block,
				cfg:         cfg,
				fetcher:     fetcher,
				log:         log,
				batcherAddr: batcherAddr,
			}, err
		}
		return &DataSource{
			open: true,
			data: data,
		}, nil
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *DataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if _, txs, err := ds.fetcher.InfoAndTxsByHash(ctx, ds.id.Hash); err == nil {
			ds.open = true
			ds.data, err = DataFromEVMTransactions(ds.cfg, ds.daCfg, ds.batcherAddr, txs, log.New("origin", ds.id))
			if err != nil {
				// already wrapped
				return nil, err
			}
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
func DataFromEVMTransactions(config *rollup.Config, daCfg *rollup.DAConfig, batcherAddr common.Address, txs types.Transactions, log log.Logger) ([]eth.Data, error) {
	var out []eth.Data
	l1Signer := config.L1Signer()
	for j, tx := range txs {
		if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
			seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				log.Warn("tx in inbox with invalid signature", "index", j, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != batcherAddr {
				log.Warn("tx in inbox with unauthorized submitter", "index", j, "err", err)
				continue // not an authorized batch submitter, ignore
			}

			if daCfg != nil {
				height, index, err := decodeETHData(tx.Data())
				if err != nil {
					log.Warn("unable to decode data pointer", "index", j, "err", err)
					return nil, err
				}
				log.Info("requesting data from celestia", "namespace", hex.EncodeToString(daCfg.Namespace.Bytes()), "height", height)
				blobs, err := daCfg.Client.Blob.GetAll(context.Background(), uint64(height), []share.Namespace{daCfg.Namespace.Bytes()})
				data := make([][]byte, len(blobs))
				for i, blob := range blobs {
					data[i] = blob.Data
				}
				if err != nil {
					return nil, NewResetError(fmt.Errorf("failed to retrieve data from celestia: %w", err))
				}
				log.Debug("retrieved data", "data", hex.EncodeToString(data[index]))
				out = append(out, data[index])
			} else {
				out = append(out, tx.Data())
			}
		}
	}
	return out, nil
}

// decodeETHData will decode the data retrieved from the EVM, this data
// was previously posted from op-batcher and contains the block height
// with transaction index of the SubmitPFD transaction to the DA.
func decodeETHData(celestiaData []byte) (int64, uint32, error) {
	buf := bytes.NewBuffer(celestiaData)
	var height int64
	err := binary.Read(buf, binary.BigEndian, &height)
	if err != nil {
		return 0, 0, fmt.Errorf("error deserializing height: %w", err)
	}
	var index uint32
	err = binary.Read(buf, binary.BigEndian, &index)
	if err != nil {
		return 0, 0, fmt.Errorf("error deserializing index: %w", err)
	}
	return height, index, nil
}
