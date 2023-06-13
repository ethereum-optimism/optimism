package bridge

import (
	"context"
	"time"

	legacy_bindings "github.com/ethereum-optimism/optimism/op-bindings/legacy-bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterStateBatchAppendedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterStateBatchAppendedWithRetry(ctx context.Context, filterer *legacy_bindings.StateCommitmentChainFilterer, opts *bind.FilterOpts) (*legacy_bindings.StateCommitmentChainStateBatchAppendedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterStateBatchAppended(opts, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}
