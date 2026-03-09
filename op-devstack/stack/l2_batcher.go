package stack

import (
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// L2Batcher represents an L2 batch-submission service, posting L2 data of an L2 to L1.
type L2Batcher interface {
	Common
	ID() ComponentID
	ActivityAPI() apis.BatcherActivity
}
