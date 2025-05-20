package loadtest

import (
	"math"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

type nonceCounter struct {
	count   uint64
	countMu sync.Mutex
}

func (n *nonceCounter) Next() uint64 {
	n.countMu.Lock()
	defer n.countMu.Unlock()
	nonce := n.count
	n.count++
	return nonce
}

func retryForever(g txplan.ReceiptGetter) txplan.Option {
	return txplan.WithRetryInclusion(g, math.MaxInt, retry.Exponential())
}
