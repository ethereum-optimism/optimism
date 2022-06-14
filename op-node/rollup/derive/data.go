package derive

import (
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/core/types"
)

func DataFromEVMTransactions(config *rollup.Config, txs types.Transactions) ([]eth.Data, []error) {
	var out []eth.Data
	var errs []error
	l1Signer := config.L1Signer()
	for j, tx := range txs {
		if to := tx.To(); to != nil && *to == config.BatchInboxAddress {
			seqDataSubmitter, err := l1Signer.Sender(tx) // optimization: only derive sender if To is correct
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid signature: tx: %d, err: %w", j, err))
				continue // bad signature, ignore
			}
			// some random L1 user might have sent a transaction to our batch inbox, ignore them
			if seqDataSubmitter != config.BatchSenderAddress {
				errs = append(errs, fmt.Errorf("unauthorized batch submitter: tx: %d", j))
				continue // not an authorized batch submitter, ignore
			}
			out = append(out, tx.Data())
		}
	}
	return out, errs
}
