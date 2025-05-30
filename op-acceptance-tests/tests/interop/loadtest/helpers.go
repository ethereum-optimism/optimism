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

func retryInclusionForever(g txplan.ReceiptGetter) txplan.Option {
	return txplan.WithRetryInclusion(g, math.MaxInt, retry.Exponential())
}

func retrySubmissionForever(s txplan.TransactionSubmitter) txplan.Option {
	return txplan.WithRetrySubmission(s, math.MaxInt, retry.Exponential())
}
