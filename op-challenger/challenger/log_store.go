package challenger

import (
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// logStore manages log subscriptions.
type logStore struct {
	// The log filter query
	query ethereum.FilterQuery

	// core sync mutex for log store
	// this locks the entire log store
	mu      sync.Mutex
	logList []types.Log
	logMap  map[common.Hash]types.Log

	// Log sbscriptions
	currentSubId SubscriptionId
	subMap       map[SubscriptionId]Subscription
	subEscapes   map[SubscriptionId]chan struct{}

	// Client to query for logs
	client ethereum.LogFilterer

	// Logger
	log log.Logger
}

// NewLogStore creates a new log store.
func NewLogStore(query ethereum.FilterQuery) *logStore {
	return &logStore{
		query:        query,
		mu:           sync.Mutex{},
		logList:      make([]types.Log, 0),
		logMap:       make(map[common.Hash]types.Log),
		currentSubId: 0,
		subMap:       make(map[SubscriptionId]Subscription),
		subEscapes:   make(map[SubscriptionId]chan struct{}),
	}
}

// newSubscription creates a new subscription.
func (l *logStore) newSubscription(query ethereum.FilterQuery) (SubscriptionId, error) {
	id := l.currentSubId.Increment()
	subscription := Subscription{
		id:     id,
		query:  query,
		client: l.client,
	}
	err := subscription.Subscribe()
	if err != nil {
		return SubscriptionId(0), err
	}
	l.subMap[id] = subscription
	l.subEscapes[id] = make(chan struct{})
	return id, nil
}

// Spawn constructs a new log subscription and listens for logs.
// This function spawns a new goroutine.
func (l *logStore) Spawn() error {
	subId, err := l.newSubscription(l.query)
	if err != nil {
		return err
	}
	go l.dispatchLogs(subId)
	return nil
}

// Quit stops all log store asynchronous tasks.
func (l *logStore) Quit() {
	for _, channel := range l.subEscapes {
		channel <- struct{}{}
		close(channel)
	}
}

// dispatchLogs dispatches logs to the log store.
// This function is intended to be run as a goroutine.
func (l *logStore) dispatchLogs(subId SubscriptionId) {
	subscription := l.subMap[subId]
	for {
		select {
		case err := <-subscription.sub.Err():
			l.log.Error("log subscription error", "err", err)
			for {
				l.log.Info("resubscribing to subscription", "id", subId)
				err := subscription.Subscribe()
				if err == nil {
					break
				}
			}
		case log := <-subscription.logs:
			l.mu.Lock()
			l.logList = append(l.logList, log)
			l.logMap[log.BlockHash] = log
			l.mu.Unlock()
		case <-l.subEscapes[subId]:
			l.log.Info("subscription received shutoff signal", "id", subId)
			return
		}
	}
}
