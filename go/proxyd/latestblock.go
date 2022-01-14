package proxyd

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const blockHeadSyncPeriod = 1 * time.Second

type LatestBlockHead struct {
	url    string
	client *ethclient.Client
	quit   chan struct{}
	done   chan struct{}

	mutex    sync.RWMutex
	blockNum uint64
}

func newLatestBlockHead(url string) (*LatestBlockHead, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}

	return &LatestBlockHead{
		url:    url,
		client: client,
		quit:   make(chan struct{}),
		done:   make(chan struct{}),
	}, nil
}

func (h *LatestBlockHead) Start() {
	go func() {
		ticker := time.NewTicker(blockHeadSyncPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				blockNum, err := h.getBlockNum()
				if err != nil {
					log.Error("error retrieving latest block number", "error", err)
					continue
				}
				log.Trace("polling block number", "blockNum", blockNum)
				h.mutex.Lock()
				h.blockNum = blockNum
				h.mutex.Unlock()

			case <-h.quit:
				close(h.done)
				return
			}
		}
	}()
}

func (h *LatestBlockHead) getBlockNum() (uint64, error) {
	const maxRetries = 5
	var err error

	for i := 0; i <= maxRetries; i++ {
		var blockNum uint64
		blockNum, err = h.client.BlockNumber(context.Background())
		if err != nil {
			backoff := calcBackoff(i)
			log.Warn("http operation failed. retrying...", "error", err, "backoff", backoff)
			time.Sleep(backoff)
			continue
		}
		return blockNum, nil
	}

	return 0, wrapErr(err, "exceeded retries")
}

func (h *LatestBlockHead) Stop() {
	close(h.quit)
	<-h.done
	h.client.Close()
}

func (h *LatestBlockHead) GetBlockNum() uint64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.blockNum
}
