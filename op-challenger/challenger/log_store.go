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

	// Log subscriptions
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

// Subscribe starts the subscription.
// This function spawns a new goroutine.
func (l *logStore) Subscribe() error {
	if l.subscription == nil {
		l.log.Error("subscription zeroed out")
		return nil
	}
	err := l.subscription.Subscribe()
	if err != nil {
		l.log.Error("failed to subscribe", "err", err)
		return err
	}
	return nil
}

// Start starts the log store.
// This function spawns a new goroutine.
func (l *logStore) Start() {
	go l.dispatchLogs()
}

// Quit stops all log store asynchronous tasks.
func (l *logStore) Quit() {
	if l.subscription != nil {
		l.subscription.Quit()
	}
}

// buildBackoffStrategy builds a [backoff.Strategy].
func (l *logStore) buildBackoffStrategy() backoff.Strategy {
	return &backoff.ExponentialStrategy{
		Min:       1000,
		Max:       20_000,
		MaxJitter: 250,
	}
}

// resubscribe resubscribes to the log store subscription with a backoff.
func (l *logStore) resubscribe() error {
	l.log.Info("resubscribing to subscription", "id", l.subscription.ID())
	ctx := context.Background()
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

// GetLogs returns all logs in the log store.
func (l *logStore) GetLogs() []types.Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logList
}

// GetLogByBlockHash returns all logs in the log store for a given block hash.
func (l *logStore) GetLogByBlockHash(blockHash common.Hash) []types.Log {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.logMap[blockHash]
}

// dispatchLogs dispatches logs to the log store.
// This function is intended to be run as a goroutine.
func (l *logStore) dispatchLogs() {
	for {
		select {
		case err := <-l.subscription.sub.Err():
			l.log.Error("log subscription error", "err", err)
			for {
				err = l.resubscribe()
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
