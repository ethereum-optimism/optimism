package bridge

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/indexer/bindings/l1bridge"
	"github.com/ethereum-optimism/optimism/indexer/bindings/scc"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterStateBatchAppendedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterStateBatchAppendedWithRetry(ctx context.Context, filterer *scc.StateCommitmentChainFilterer, opts *bind.FilterOpts) (*scc.StateCommitmentChainStateBatchAppendedIterator, error) {
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

// FilterETHDepositInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterETHDepositInitiatedWithRetry(ctx context.Context, filterer *l1bridge.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*l1bridge.L1StandardBridgeETHDepositInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterETHDepositInitiated(opts, nil, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}

// FilterERC20DepositInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterERC20DepositInitiatedWithRetry(ctx context.Context, filterer *l1bridge.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*l1bridge.L1StandardBridgeERC20DepositInitiatedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterERC20DepositInitiated(opts, nil, nil, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}
