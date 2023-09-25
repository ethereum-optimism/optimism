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

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// CallDataSource fetches call data (EVM inputs embedded in transactions) for a given block,
// filtered to a single batch submitter.
type CallDataSource struct {
	open     bool
	callData []eth.Data

	ref         eth.L1BlockRef
	batcherAddr common.Address

	cfg     *rollup.Config
	fetcher L1TransactionFetcher
	log     log.Logger
}

// NewCallDataSource creates a new call-data source.
func NewCallDataSource(ctx context.Context, log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher, ref eth.L1BlockRef, batcherAddr common.Address) DataIter {
	return &CallDataSource{
		open:        false,
		ref:         ref,
		cfg:         cfg,
		fetcher:     fetcher,
		log:         log.New("origin", ref),
		batcherAddr: batcherAddr,
	}
}

// Next returns the next piece of data if any remains. It returns ResetError if it cannot find the
// referenced block, or TemporaryError for any other failure to fetch the block.
func (ds *CallDataSource) Next(ctx context.Context) (eth.Data, error) {
	if !ds.open {
		if _, txs, err := ds.fetcher.InfoAndTxsByHash(ctx, ds.ref.Hash); err == nil {
			ds.open = true
			ds.callData = CallDataFromEVMTransactions(ds.cfg, ds.batcherAddr, txs, ds.log)
		} else if errors.Is(err, ethereum.NotFound) {
			return nil, NewResetError(fmt.Errorf("failed to open call-data source: %w", err))
		} else {
			return nil, NewTemporaryError(fmt.Errorf("failed to open call-data source: %w", err))
		}
	}
	if len(ds.callData) == 0 {
		return nil, io.EOF
	} else {
		data := ds.callData[0]
		ds.callData = ds.callData[1:]
		return data, nil
	}
}

// CallDataFromEVMTransactions filters all of the transactions and returns the call-data from transactions
// that are sent to the batch inbox address from the batch sender address.
// This will return an empty array if no valid transactions are found.
func CallDataFromEVMTransactions(config *rollup.Config, batcherAddr common.Address, txs types.Transactions, log log.Logger) []eth.Data {
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
			out = append(out, tx.Data())
		}
	}
	return out
}
