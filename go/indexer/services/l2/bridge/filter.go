package bridge

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l2bridge"
	"github.com/ethereum-optimism/optimism/l2geth/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterWithdrawalInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterWithdrawalInitiatedWithRetry(filterer *l2bridge.L2StandardBridgeFilterer, opts *bind.FilterOpts) (*l2bridge.L2StandardBridgeWithdrawalInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(opts.Context, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterWithdrawalInitiated(opts, nil, nil, nil)
		switch err {
		case nil:
			cancel()
			return res, err
		default:
			logger.Error("Error fetching filter", "err", err)
		}
		time.Sleep(clientRetryInterval)
	}
}
