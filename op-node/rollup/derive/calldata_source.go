package derive

import (
	"context"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// CalldataSource readers raw transactions from a given block & then filters for
// batch submitter transactions.
// This is not a stage in the pipeline, but a wrapper for another stage in the pipeline

type L1TransactionFetcher interface {
	InfoAndTxsByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, types.Transactions, error)
}

// CalldataSourceImpl is a fault tolerant approach to fetching data.
// The constructor will never fail & it will instead re-attempt the fetcher
// at a later point.
// This API greatly simplifies some calling code.
type CalldataSourceImpl struct {
	// Internal state + data
	open bool
	data []eth.Data
	// Required to re-attempt fetching
	id      eth.BlockID
	cfg     *rollup.Config // TODO: `DataFromEVMTransactions` should probably not take the full config
	fetcher L1TransactionFetcher
	log     log.Logger
}

func NewCalldataSourceImpl(ctx context.Context, log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher, block eth.BlockID) *CalldataSourceImpl {
	_, txs, err := fetcher.InfoAndTxsByHash(ctx, block.Hash)
	if err != nil {
		return &CalldataSourceImpl{
			open:    false,
			id:      block,
			cfg:     cfg,
			fetcher: fetcher,
			log:     log,
		}
	} else {
		return &CalldataSourceImpl{
			open: true,
			data: DataFromEVMTransactions(cfg, txs, log.New("origin", block)),
		}
	}
}

func (cs *CalldataSourceImpl) Next(ctx context.Context) (eth.Data, error) {
	if !cs.open {
		if _, txs, err := cs.fetcher.InfoAndTxsByHash(ctx, cs.id.Hash); err == nil {
			cs.open = true
			cs.data = DataFromEVMTransactions(cs.cfg, txs, log.New("origin", cs.id))
		} else {
			return nil, err
		}
	}
	if len(cs.data) == 0 {
		return nil, io.EOF
	} else {
		data := cs.data[0]
		cs.data = cs.data[1:]
		return data, nil
	}
}

type CalldataSource struct {
	log     log.Logger
	cfg     *rollup.Config
	fetcher L1TransactionFetcher
}

func NewCalldataSource(log log.Logger, cfg *rollup.Config, fetcher L1TransactionFetcher) *CalldataSource {
	return &CalldataSource{log: log, cfg: cfg, fetcher: fetcher}
}

func (cs *CalldataSource) OpenData(ctx context.Context, id eth.BlockID) *CalldataSourceImpl {
	return NewCalldataSourceImpl(ctx, cs.log, cs.cfg, cs.fetcher, id)
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
