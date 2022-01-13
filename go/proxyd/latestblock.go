package proxyd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"
)

const blockHeadSyncPeriod = 1 * time.Second

type LatestBlockHead struct {
	url    string
	client *http.Client
	quit   chan struct{}

	mutex    sync.RWMutex
	blockNum uint64
}

func newLatestBlockHead(url string) *LatestBlockHead {
	return &LatestBlockHead{
		url:    url,
		client: &http.Client{Timeout: 5 * time.Second},
		quit:   make(chan struct{}),
	}
}

func (h *LatestBlockHead) Start() error {
	go func() {
		ticker := time.NewTicker(blockHeadSyncPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				blockNum, err := h.getBlockNum()
				if err != nil {
					log.Error("error retrieving latest block number", "error", err)
				}
				log.Trace("polling block number", "blockNum", blockNum)
				h.mutex.Lock()
				h.blockNum = blockNum
				h.mutex.Unlock()

			case <-h.quit:
				return
			}
		}
	}()

	return nil
}

func (h *LatestBlockHead) getBlockNum() (uint64, error) {
	rpcReq := RPCReq{
		JSONRPC: "2.0",
		Method:  "eth_blockNumber",
		ID:      []byte(strconv.Itoa(1)),
	}
	body := mustMarshalJSON(&rpcReq)

	const maxRetries = 5
	var httpErr error

	for i := 0; i <= maxRetries; i++ {
		httpReq, err := http.NewRequest("POST", h.url, bytes.NewReader(body))
		if err != nil {
			return 0, err
		}
		httpReq.Header.Set("Content-Type", "application/json")

		httpRes, httpErr := h.client.Do(httpReq)
		if httpErr != nil {
			backoff := calcBackoff(i)
			log.Warn("http operation failed. retrying...", "error", err, "backoff", backoff)
			time.Sleep(backoff)
			continue
		}
		if httpRes.StatusCode != 200 {
			return 0, fmt.Errorf("resposne code %d", httpRes.StatusCode)
		}
		defer httpRes.Body.Close()

		res := new(RPCRes)
		if err := json.NewDecoder(httpRes.Body).Decode(res); err != nil {
			return 0, err
		}
		blockNumHex, ok := res.Result.(string)
		if !ok {
			return 0, fmt.Errorf("invalid eth_blockNumber result")
		}
		blockNum, err := hexutil.DecodeUint64(blockNumHex)
		if err != nil {
			return 0, err
		}

		return blockNum, nil
	}

	return 0, wrapErr(httpErr, "exceeded retries")
}

func (h *LatestBlockHead) Stop() {
	close(h.quit)
}

func (h *LatestBlockHead) GetBlockNum() uint64 {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return h.blockNum
}
