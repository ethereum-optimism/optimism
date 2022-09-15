package bridge

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

// clientRetryInterval is the interval to wait between retrying client API
// calls.
var clientRetryInterval = 5 * time.Second

// FilterETHDepositInitiatedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterETHDepositInitiatedWithRetry(ctx context.Context, filterer *bindings.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*bindings.L1StandardBridgeETHDepositInitiatedIterator, error) {
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
func FilterERC20DepositInitiatedWithRetry(ctx context.Context, filterer *bindings.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*bindings.L1StandardBridgeERC20DepositInitiatedIterator, error) {
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

// FilterETHWithdrawalFinalizedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterETHWithdrawalFinalizedWithRetry(ctx context.Context, filterer *bindings.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*bindings.L1StandardBridgeETHWithdrawalFinalizedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterETHWithdrawalFinalized(opts, nil, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}

// FilterERC20WithdrawalFinalizedWithRetry retries the given func until it succeeds,
// waiting for clientRetryInterval duration after every call.
func FilterERC20WithdrawalFinalizedWithRetry(ctx context.Context, filterer *bindings.L1StandardBridgeFilterer, opts *bind.FilterOpts) (*bindings.L1StandardBridgeERC20WithdrawalFinalizedIterator, error) {
	for {
		ctxt, cancel := context.WithTimeout(ctx, DefaultConnectionTimeout)
		opts.Context = ctxt
		res, err := filterer.FilterERC20WithdrawalFinalized(opts, nil, nil, nil)
		cancel()
		if err == nil {
			return res, nil
		}
		logger.Error("Error fetching filter", "err", err)
		time.Sleep(clientRetryInterval)
	}
}
