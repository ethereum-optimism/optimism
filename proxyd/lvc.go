package proxyd

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
)

const cacheSyncRate = 1 * time.Second

type lvcUpdateFn func(context.Context, *ethclient.Client) (string, error)

type EthLastValueCache struct {
	client  *ethclient.Client
	cache   Cache
	key     string
	updater lvcUpdateFn
	quit    chan struct{}
}

func newLVC(client *ethclient.Client, cache Cache, cacheKey string, updater lvcUpdateFn) *EthLastValueCache {
	return &EthLastValueCache{
		client:  client,
		cache:   cache,
		key:     cacheKey,
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
				lvcPollTimeGauge.WithLabelValues(h.key).SetToCurrentTime()

				value, err := h.getUpdate()
				if err != nil {
					log.Error("error retrieving latest value", "key", h.key, "error", err)
					continue
				}
				log.Trace("polling latest value", "value", value)

				if err := h.cache.Put(context.Background(), h.key, value); err != nil {
					log.Error("error writing last value to cache", "key", h.key, "error", err)
				}

			case <-h.quit:
				return
			}
		}
	}()
}

func (h *EthLastValueCache) getUpdate() (string, error) {
	const maxRetries = 5
	var err error

	for i := 0; i <= maxRetries; i++ {
		var value string
		value, err = h.updater(context.Background(), h.client)
		if err != nil {
			backoff := calcBackoff(i)
			log.Warn("http operation failed. retrying...", "error", err, "backoff", backoff)
			lvcErrorsTotal.WithLabelValues(h.key).Inc()
			time.Sleep(backoff)
			continue
		}
		return value, nil
	}

	return "", wrapErr(err, "exceeded retries")
}

func (h *EthLastValueCache) Stop() {
	close(h.quit)
}

func (h *EthLastValueCache) Read(ctx context.Context) (string, error) {
	return h.cache.Get(ctx, h.key)
}
