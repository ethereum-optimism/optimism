package derive

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (L1Info, types.Transactions, error)
}

type CalldataSource struct {
	log     log.Logger
	cfg     *rollup.Config
	fetcher L1TransactionFetcher
}

func NewCalldataSource(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher) *CalldataSource {
	return &CalldataSource{log: log, cfg: cfg, fetcher: fetcher}
}

func (cs *CalldataSource) Fetch(ctx context.Context, id eth.BlockID) (eth.L1BlockRef, []eth.Data, error) {
	l1Info, txs, err := cs.fetcher.InfoAndTxsByHash(ctx, id.Hash)
	if err != nil {
		return eth.L1BlockRef{}, nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	data := DataFromEVMTransactions(cs.cfg, txs, cs.log.New("origin", l1Info.ID()))
	return l1Info.BlockRef(), data, nil
}

func DataFromEVMTransactions(config *rollup.Config, txs types.Transactions, log log.Logger) []eth.Data {
	var out []eth.Data
	l1Signer := config.L1Signer()
	for j, tx := range txs {
		if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
			seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				log.Debug("tx in inbox with invalid signature", "index", j, "err", err)
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != config.BatchSenderAddress {
				log.Debug("tx in inbox with unauthorized submitter", "index", j, "err", err)
				continue // not an authorized batch submitter, ignore
			}
			out = append(out, tx.Data())
		}
	}
	return out
}
