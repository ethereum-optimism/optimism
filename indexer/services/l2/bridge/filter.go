package bridge

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/bindings/l2bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterWithdrawalInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterWithdrawalInitiatedWithRetry(ctx context.Context, filterer *l2bridge.L2StandardBridgeFilterer, opts *bind.FilterOpts) (*l2bridge.L2StandardBridgeWithdrawalInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterWithdrawalInitiated(opts, nil, nil, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}
