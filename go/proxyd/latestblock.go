package proxyd

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type LatestBlockHead struct {
	url  string
	quit chan struct{}

	mutex sync.Mutex
	head  *types.Header
}

func newLatestBlockHead(url string) *LatestBlockHead {
	return &LatestBlockHead{
		url:  url,
		quit: make(chan struct{}),
	}
}

func (h *LatestBlockHead) Start() error {
	client, err := ethclient.DialContext(context.Background(), h.url)
	if err != nil {
		return err
	}
	heads := make(chan *types.Header)
	sub, err := client.SubscribeNewHead(context.Background(), heads)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case head := <-heads:
				h.mutex.Lock()
				h.head = head
				h.mutex.Unlock()
			case <-h.quit:
				sub.Unsubscribe()
			}
		}
	}()

	return nil
}

func (h *LatestBlockHead) Stop() {
	close(h.quit)
}

func (h *LatestBlockHead) GetBlockNum() uint64 {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.head.Number.Uint64()
}
