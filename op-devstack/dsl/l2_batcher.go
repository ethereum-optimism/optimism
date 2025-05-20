package dsl

import (
	"fmt"
	"strings"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/retry"
	"github.com/stretchr/testify/require"
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

func (b *L2Batcher) Stop() {
	err := retry.Do0(b.ctx, 3, retry.Exponential(), func() error {
		err := b.Escape().ActivityAPI().StopBatcher(b.ctx)
		if err != nil && strings.Contains(err.Error(), "batcher is not running") {
			return nil
		}
		return err
	})
	require.NoError(b.t, err, fmt.Sprintf("Expected to be able to call StopBatcher API on chain %s, but got error", b.inner.ID().ChainID))
}

func (b *L2Batcher) Start() {
	err := retry.Do0(b.ctx, 3, retry.Exponential(), func() error {
		return b.inner.ActivityAPI().StartBatcher(b.ctx)
	})
	require.NoError(b.t, err, fmt.Sprintf("Expected to be able to call StartBatcher API on chain %s, but got error", b.inner.ID().ChainID))
}
