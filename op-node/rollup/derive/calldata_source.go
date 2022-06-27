package derive

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.L1Info, types.Transactions, error)
}

type DataSlice []eth.Data

func (ds *DataSlice) Next(ctx context.Context) (eth.Data, error) {
	if len(*ds) == 0 {
		return nil, io.EOF
	}
	out := (*ds)[0]
	*ds = (*ds)[1:]
	return out, nil
}

type CalldataSource struct {
	log     log.Logger
	cfg     *rollup.Config
	fetcher L1TransactionFetcher
}

func NewCalldataSource(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher) *CalldataSource {
	return &CalldataSource{log: log, cfg: cfg, fetcher: fetcher}
}

func (cs *CalldataSource) OpenData(ctx context.Context, id eth.BlockID) (DataIter, error) {
	_, txs, err := cs.fetcher.InfoAndTxsByHash(ctx, id.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	data := DataFromEVMTransactions(cs.cfg, txs, cs.log.New("origin", id))
	return (*DataSlice)(&data), nil
}

func DataFromEVMTransactions(config *rollup.Config, txs types.Transactions, log log.Logger) []eth.Data {
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
			if seqDataSubmitter != config.BatchSenderAddress {
				log.Warn("tx in inbox with unauthorized submitter", "index", j, "err", err)
				continue // not an authorized batch submitter, ignore
			}
			out = append(out, tx.Data())
		}
	}
	return out
}
