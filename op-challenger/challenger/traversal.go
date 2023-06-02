package challenger

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

var (
	// ErrFailedToFetchReceipts is returned when the receipts for a block could not be fetched.
	ErrFailedToFetchReceipts = errors.New("failed to fetch receipts")
	// ErrAlreadyStarted is returned when the log traversal has already been started.
	ErrAlreadyStarted = errors.New("already started")
	// ExponentialBackoff is the default backoff strategy.
	ExponentialBackoff = backoff.Exponential()
)

// logTraversal implements LogTraversal.
type logTraversal struct {
	log             log.Logger
	client          MinimalEthClient
	query           *ethereum.FilterQuery
	quit            chan struct{}
	mutex           sync.RWMutex
	lastBlockNumber *big.Int
	started         bool
}

//go:generate mockery --name MinimalEthClient --output ./mocks/
type MinimalEthClient interface {
	InfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error)
	InfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
}

// NewLogTraversal creates a new log traversal.
// The [MinimalEthClient] passed into this function should be built using an op-service backoff client as the
// underlying [client.RPC] passed into the op-node EthClient.
func NewLogTraversal(client MinimalEthClient, query *ethereum.FilterQuery, log log.Logger, lastBlockNumber *big.Int) *logTraversal {
	return &logTraversal{
		client:          client,
		query:           query,
		quit:            make(chan struct{}),
		log:             log,
		mutex:           sync.RWMutex{},
		lastBlockNumber: lastBlockNumber,
	}
}

// LastBlockNumber returns the last block number that was traversed.
func (l *logTraversal) LastBlockNumber() *big.Int {
	return l.lastBlockNumber
}

// Quit quits the log traversal.
func (l *logTraversal) Quit() {
	l.quit <- struct{}{}
	l.started = false
}

// Start starts the log traversal.
func (l *logTraversal) Start(ctx context.Context, handleLog func(*types.Log) error) error {
	if l.started {
		return ErrAlreadyStarted
	}
	headers := make(chan *types.Header)
	sub, err := l.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		l.log.Error("Failed to subscribe to new heads", "err", err)
		return err
	}
	l.started = true
	go l.onNewHead(ctx, headers, sub, handleLog)
	return nil
}

// Started returns true if the log traversal has started.
func (l *logTraversal) Started() bool {
	return l.started
}

// onNewHead handles a new [types.Header].
func (l *logTraversal) onNewHead(ctx context.Context, headers chan *types.Header, sub ethereum.Subscription, handleLog func(*types.Log) error) {
	for {
		select {
		case <-l.quit:
			l.log.Info("Stopping log traversal: received quit signal")
			sub.Unsubscribe()
			return
		case header := <-headers:
			l.log.Info("Received new head", "number", header.Number)
			l.dispatchNewHead(ctx, header, handleLog)
		}
	}
}

// fetchAndProcessLogs fetches and processes logs given a [eth.BlockInfo].
func (l *logTraversal) fetchAndProcessLogs(ctx context.Context, info eth.BlockInfo, handleLog func(*types.Log) error) error {
	_, receipts, err := l.client.FetchReceipts(ctx, info.Hash())
	if err != nil {
		l.log.Error("Failed to fetch receipts", "err", err)
		return ErrFailedToFetchReceipts
	}
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			err := handleLog(log)
			if err != nil {
				l.log.Error("Failed to handle log", "err", err)
				return err
			}
		}
	}
	return nil
}

// spawnCatchup spawns a new goroutine to "catchup" for missed/skipped blocks.
func (l *logTraversal) spawnCatchup(ctx context.Context, start *big.Int, end *big.Int, handleLog func(*types.Log) error) {
	for {
		if start.Cmp(end) == 1 {
			l.log.Info("Traversal caught up", "start", start, "end", end)
			return
		}

		info, err := l.client.InfoByNumber(ctx, start.Uint64())
		if err != nil {
			l.log.Error("Failed to fetch block", "err", err)
			return
		}

		if err := l.fetchAndProcessLogs(ctx, info, handleLog); err != nil {
			return
		}

		start = start.Add(start, big.NewInt(1))
	}
}

// updateBlockNumber updates the last block number.
func (l *logTraversal) updateBlockNumber(blockNumber *big.Int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.lastBlockNumber = blockNumber
}

// accessBlockNumber retrieves the last block number.
func (l *logTraversal) accessBlockNumber() *big.Int {
	l.mutex.RLock()
	defer l.mutex.RUnlock()
	return l.lastBlockNumber
}

// dispatchNewHead dispatches a new head.
func (l *logTraversal) dispatchNewHead(ctx context.Context, header *types.Header, handleLog func(*types.Log) error) {
	info, err := l.client.InfoByHash(ctx, header.Hash())
	if err != nil {
		l.log.Error("Failed to fetch block", "err", err)
		return
	}

	expectedBlockNumber := l.accessBlockNumber()
	expectedBlockNumber = expectedBlockNumber.Add(expectedBlockNumber, big.NewInt(1))
	currentBlockNumber := big.NewInt(int64(info.NumberU64()))
	if currentBlockNumber.Cmp(expectedBlockNumber) == 1 {
		l.log.Warn("Detected skipped block", "expectedBlockNumber", expectedBlockNumber, "currentBlockNumber", currentBlockNumber)
		endBlockNum := currentBlockNumber.Sub(currentBlockNumber, big.NewInt(1))
		l.log.Warn("Spawning catchup thread", "start", expectedBlockNumber, "end", endBlockNum)
		go l.spawnCatchup(ctx, expectedBlockNumber, endBlockNum, handleLog)

	}
	l.updateBlockNumber(currentBlockNumber)

	_ = l.fetchAndProcessLogs(ctx, info, handleLog)
}
