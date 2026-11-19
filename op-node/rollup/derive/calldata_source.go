package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
)

// CalldataSource is a fault tolerant approach to fetching data.
// The constructor will never fail & it will instead re-attempt the fetcher
// at a later point.
type CalldataSource struct {
	// Internal state + data
	open bool
	data []eth.Data
	// Required to re-attempt fetching
	ref     eth.L1BlockRef
	dsCfg   DataSourceConfig
	fetcher L1TransactionFetcher
	log     log.Logger

	batcherAddr common.Address
}

// NewCalldataSource creates a new calldata source. It suppresses errors in fetching the L1 block if they occur.
// If there is an error, it will attempt to fetch the result on the next call to `Next`.
func NewCalldataSource(ctx context.Context, log log.Logger, dsCfg DataSourceConfig, fetcher L1TransactionFetcher, ref eth.L1BlockRef, batcherAddr common.Address) DataIter {
	data, err := fetchCalldata(ctx, fetcher, ref, dsCfg, batcherAddr, log.New("origin", ref))
	if err != nil {
		return &CalldataSource{
			open:        false,
			ref:         ref,
			dsCfg:       dsCfg,
			fetcher:     fetcher,
			log:         log,
			batcherAddr: batcherAddr,
		}
	}
	return &CalldataSource{
		open: true,
		data: data,
	}
}

// Next returns the next piece of data if it has it. If the constructor failed, this
// will attempt to reinitialize itself. If it cannot find the block it returns a ResetError
// otherwise it returns a temporary error if fetching the block returns an error.
func (ds *CalldataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if data, err := fetchCalldata(ctx, ds.fetcher, ds.ref, ds.dsCfg, ds.batcherAddr, ds.log); err == nil {
			ds.open = true
			ds.data = data
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

// fetchCalldata fetches the L1 block ref and returns the calldata of its batcher transactions.
func fetchCalldata(ctx context.Context, fetcher L1TransactionFetcher, ref eth.L1BlockRef, dsCfg DataSourceConfig, batcherAddr common.Address, log log.Logger) ([]eth.Data, error) {
	_, txs, err := fetcher.InfoAndRawTxsByHash(ctx, ref.Hash)
	if err != nil {
		return nil, err
	}
	return DataFromEVMTransactions(dsCfg, batcherAddr, txs, log)
}

// DataFromEVMTransactions filters the transactions of an L1 block, given in their EIP-2718
// encoding, and returns the calldata from transactions that are sent to the batch inbox address
// from the batch sender address. It returns an empty array if no valid transactions are found,
// and an error if a transaction does not decode.
func DataFromEVMTransactions(dsCfg DataSourceConfig, batcherAddr common.Address, txs []hexutil.Bytes, log log.Logger) ([]eth.Data, error) {
	btxs, err := batcherTxs(txs, &dsCfg, batcherAddr, log)
	if err != nil {
		return nil, err
	}
	out := []eth.Data{}
	for _, btx := range btxs {
		out = append(out, btx.calldata)
	}
	return out, nil
}
