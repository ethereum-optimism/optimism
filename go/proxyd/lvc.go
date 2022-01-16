package proxyd

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const cacheSyncRate = 1 * time.Second

type lvcUpdateFn func(context.Context, *ethclient.Client) (interface{}, error)

type EthLastValueCache struct {
	client  *ethclient.Client
	updater lvcUpdateFn
	quit    chan struct{}

	mutex sync.RWMutex
	value interface{}
}

func newLVC(client *ethclient.Client, updater lvcUpdateFn) *EthLastValueCache {
	return &EthLastValueCache{
		client:  client,
		updater: updater,
		quit:    make(chan struct{}),
	}
}

func (h *EthLastValueCache) Start() {
	go func() {
		ticker := time.NewTicker(cacheSyncRate)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				value, err := h.getUpdate()
				if err != nil {
					log.Error("error retrieving latest value", "error", err)
					continue
				}
				log.Trace("polling latest value", "value", value)
				h.mutex.Lock()
				h.value = value
				h.mutex.Unlock()

			case <-h.quit:
				return
			}
		}
	}()
}

func (h *EthLastValueCache) getUpdate() (interface{}, error) {
	const maxRetries = 5
	var err error

	for i := 0; i <= maxRetries; i++ {
		var value interface{}
		value, err = h.updater(context.Background(), h.client)
		if err != nil {
			backoff := calcBackoff(i)
			log.Warn("http operation failed. retrying...", "error", err, "backoff", backoff)
			time.Sleep(backoff)
			continue
		}
		return value, nil
	}

	return 0, wrapErr(err, "exceeded retries")
}

func (h *EthLastValueCache) Stop() {
	close(h.quit)
}

func (h *EthLastValueCache) Read() interface{} {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.value
}
