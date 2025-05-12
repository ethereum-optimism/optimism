package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
)

// L2Batcher wraps a stack.L2Batcher interface for DSL operations
type L2Batcher struct {
	commonImpl
	inner stack.L2Batcher
}

// NewL2Batcher creates a new L2Batcher DSL wrapper
func NewL2Batcher(inner stack.L2Batcher) *L2Batcher {
	return &L2Batcher{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (b *L2Batcher) String() string {
	return b.inner.ID().String()
}

// Escape returns the underlying stack.L2Batcher
func (b *L2Batcher) Escape() stack.L2Batcher {
	return b.inner
}

func (b *L2Batcher) ActivityAPI() apis.BatcherActivity {
	return b.inner.ActivityAPI()
}
