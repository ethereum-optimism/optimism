package challenger

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

var (
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
	mutex           sync.Mutex
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
func NewLogTraversal(client MinimalEthClient, query *ethereum.FilterQuery, log log.Logger) *logTraversal {
	return &logTraversal{
		client:          client,
		query:           query,
		quit:            make(chan struct{}),
		log:             log,
		mutex:           sync.Mutex{},
		lastBlockNumber: big.NewInt(0),
	}
}

// LastBlockNumber returns the last block number that was traversed.
func (l *logTraversal) LastBlockNumber() *big.Int {
	return l.lastBlockNumber
}

// fetchBlockByHash gracefully fetches block info by hash with a backoff.
func (l *logTraversal) fetchBlockInfoByHash(ctx context.Context, hash common.Hash) (eth.BlockInfo, error) {
	var info eth.BlockInfo
	err := backoff.DoCtx(ctx, 5, ExponentialBackoff, func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		var err error
		info, err = l.client.InfoByHash(ctx, hash)
		return err
	})
	return info, err
}

// fetchBlockByNumber gracefully fetches block info by number with a backoff.
func (l *logTraversal) fetchBlockInfoByNumber(ctx context.Context, number uint64) (eth.BlockInfo, error) {
	var info eth.BlockInfo
	err := backoff.DoCtx(ctx, 5, ExponentialBackoff, func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		var err error
		info, err = l.client.InfoByNumber(ctx, number)
		return err
	})
	return info, err
}

// fetchBlockReceipts fetches receipts for a block by hash with a backoff.
func (l *logTraversal) fetchBlockReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	var receipts types.Receipts
	err := backoff.DoCtx(ctx, 5, ExponentialBackoff, func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()
		var err error
		_, receipts, err = l.client.FetchReceipts(ctx, hash)
		return err
	})
	if err != nil {
		return nil, err
	}
	return receipts, nil
}

// subscribeNewHead subscribes to new heads with a backoff.
func (l *logTraversal) subscribeNewHead(ctx context.Context, headers chan *types.Header) (ethereum.Subscription, error) {
	var sub ethereum.Subscription
	err := backoff.DoCtx(ctx, 4, ExponentialBackoff, func() error {
		ctx, cancel := context.WithTimeout(ctx, time.Second*5)
		defer cancel()
		var err error
		sub, err = l.client.SubscribeNewHead(ctx, headers)
		return err
	})
	return sub, err
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
	sub, err := l.subscribeNewHead(ctx, headers)
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
			l.dispatchNewHead(ctx, header, handleLog, true)
		}
	}
}

// spawnCatchup spawns a new goroutine to "catchup" for missed/skipped blocks.
func (l *logTraversal) spawnCatchup(ctx context.Context, start *big.Int, end *big.Int, handleLog func(*types.Log) error) {
	for {
		// Break if we've caught up (start > end).
		if start.Cmp(end) == 1 {
			l.log.Info("Traversal caught up", "start", start, "end", end)
			return
		}

		info, err := l.fetchBlockInfoByNumber(ctx, start.Uint64())
		if err != nil {
			l.log.Error("Failed to fetch block", "err", err)
			return
		}

		receipts, err := l.fetchBlockReceipts(ctx, info.Hash())
		if err != nil {
			l.log.Error("Failed to fetch receipts", "err", err)
			return
		}
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				err := handleLog(log)
				if err != nil {
					l.log.Error("Failed to handle log", "err", err)
					return
				}
			}
		}

		// Increment to the next block to catch up to
		start = start.Add(start, big.NewInt(1))
	}
}

// updateBlockNumber updates the last block number with a mutex lock.
func (l *logTraversal) updateBlockNumber(blockNumber *big.Int) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.lastBlockNumber = blockNumber
}

// dispatchNewHead dispatches a new head.
func (l *logTraversal) dispatchNewHead(ctx context.Context, header *types.Header, handleLog func(*types.Log) error, allowCatchup bool) {
	info, err := l.fetchBlockInfoByHash(ctx, header.Hash())
	if err != nil {
		l.log.Error("Failed to fetch block", "err", err)
		return
	}
	expectedBlockNumber := l.lastBlockNumber.Add(l.lastBlockNumber, big.NewInt(1))
	currentBlockNumber := big.NewInt(int64(info.NumberU64()))
	if l.lastBlockNumber.Cmp(big.NewInt(0)) != 0 && currentBlockNumber.Cmp(expectedBlockNumber) == 1 {
		l.log.Warn("Detected skipped block", "expectedBlockNumber", expectedBlockNumber, "currentBlockNumber", currentBlockNumber)
		if allowCatchup {
			endBlockNum := currentBlockNumber.Sub(currentBlockNumber, big.NewInt(1))
			l.log.Warn("Spawning catchup thread", "start", expectedBlockNumber, "end", endBlockNum)
			go l.spawnCatchup(ctx, expectedBlockNumber, endBlockNum, handleLog)
		} else {
			l.log.Warn("Missed block detected with catchup disabled")
		}
	}
	l.updateBlockNumber(currentBlockNumber)
	receipts, err := l.fetchBlockReceipts(ctx, info.Hash())
	if err != nil {
		l.log.Error("Failed to fetch receipts", "err", err)
		return
	}
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			err := handleLog(log)
			if err != nil {
				l.log.Error("Failed to handle log", "err", err)
				return
			}
		}
	}
}
