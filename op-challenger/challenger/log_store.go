package challenger

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

// logStore manages log subscriptions.
type logStore struct {
	// The log filter query
	query ethereum.FilterQuery

	// core sync mutex for log store
	// this locks the entire log store
	mu      sync.Mutex
	logList []types.Log
	logMap  map[common.Hash][]types.Log

	// Log subscription
	subscription *Subscription

	// Client to query for logs
	client ethereum.LogFilterer

	// Logger
	log log.Logger
}

// NewLogStore creates a new log store.
func NewLogStore(query ethereum.FilterQuery, client ethereum.LogFilterer, log log.Logger) *logStore {
	return &logStore{
		query:        query,
		mu:           sync.Mutex{},
		logList:      make([]types.Log, 0),
		logMap:       make(map[common.Hash][]types.Log),
		subscription: NewSubscription(query, client, log),
		client:       client,
		log:          log,
	}
}

// Subscribed returns true if the subscription has started.
func (l *logStore) Subscribed() bool {
	return l.subscription.Started()
}

// Query returns the log filter query.
func (l *logStore) Query() ethereum.FilterQuery {
	return l.query
}

// Client returns the log filter client.
func (l *logStore) Client() ethereum.LogFilterer {
	return l.client
}

// GetLogs returns all logs in the log store.
func (l *logStore) GetLogs() []types.Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	logs := make([]types.Log, len(l.logList))
	copy(logs, l.logList)
	return logs
}

// GetLogByBlockHash returns all logs in the log store for a given block hash.
func (l *logStore) GetLogByBlockHash(blockHash common.Hash) []types.Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	logs := make([]types.Log, len(l.logMap[blockHash]))
	copy(logs, l.logMap[blockHash])
	return logs
}

// Subscribe starts the subscription.
// This function spawns a new goroutine.
func (l *logStore) Subscribe(ctx context.Context) error {
	err := l.subscription.Subscribe()
	if err != nil {
		l.log.Error("failed to subscribe", "err", err)
		return err
	}
	go l.dispatchLogs(ctx)
	return nil
}

// Quit stops all log store asynchronous tasks.
func (l *logStore) Quit() {
	l.subscription.Quit()
}

// buildBackoffStrategy builds a [backoff.Strategy].
func (l *logStore) buildBackoffStrategy() backoff.Strategy {
	return &backoff.ExponentialStrategy{
		Min:       1000,
		Max:       20_000,
		MaxJitter: 250,
	}
}

// resubscribe attempts to re-establish the log store internal
// subscription with a backoff strategy.
func (l *logStore) resubscribe(ctx context.Context) error {
	l.log.Info("log store resubscribing with backoff")
	backoffStrategy := l.buildBackoffStrategy()
	return backoff.DoCtx(ctx, 10, backoffStrategy, func() error {
		if l.subscription == nil {
			l.log.Error("subscription zeroed out")
			return nil
		}
		err := l.subscription.Subscribe()
		if err == nil {
			l.log.Info("subscription reconnected", "id", l.subscription.ID())
		}
		return err
	})
}

// insertLog inserts a log into the log store.
func (l *logStore) insertLog(log types.Log) {
	l.mu.Lock()
	l.logList = append(l.logList, log)
	l.logMap[log.BlockHash] = append(l.logMap[log.BlockHash], log)
	l.mu.Unlock()
}

// dispatchLogs dispatches logs to the log store.
// This function is intended to be run as a goroutine.
func (l *logStore) dispatchLogs(ctx context.Context) {
	for {
		select {
		case err := <-l.subscription.sub.Err():
			l.log.Error("log subscription error", "err", err)
			for {
				err = l.resubscribe(ctx)
				if err == nil {
					break
				}
			}
		case log := <-l.subscription.logs:
			l.insertLog(log)
		case <-l.subscription.quit:
			l.log.Info("received quit signal from subscription", "id", l.subscription.ID())
			return
		}
	}
}
