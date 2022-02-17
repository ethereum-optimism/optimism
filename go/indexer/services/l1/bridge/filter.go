package bridge

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/go/indexer/bindings/l1bridge"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterETHDepositInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterETHDepositInitiatedWithRetry(filterer *l1bridge.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*l1bridge.L1StandardBridgeETHDepositInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(opts.Context, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterETHDepositInitiated(opts, nil, nil)
		switch err {
		case nil:
			cancel()
			return res, err
		default:
			logger.Error("Error fetching filter", "err", err)
			break
		}
		time.Sleep(clientRetryInterval)
	}
}

// FilterERC20DepositInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterERC20DepositInitiatedWithRetry(filterer *l1bridge.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*l1bridge.L1StandardBridgeERC20DepositInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(opts.Context, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterERC20DepositInitiated(opts, nil, nil, nil)
		switch err {
		case nil:
			cancel()
			return res, err
		default:
			logger.Error("Error fetching filter", "err", err)
			break
		}
		time.Sleep(clientRetryInterval)
	}
}
