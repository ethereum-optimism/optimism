package builder

import (
	"context"
	"errors"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type BuilderMetrics interface {
	RecordBuilderPayloadBytes(num int)
}

type PayloadBuilder interface {
	Enabled() bool
	Timeout() time.Duration
	GetPayload(ctx context.Context, ref eth.L2BlockRef, log log.Logger, metrics BuilderMetrics) (*eth.ExecutionPayloadEnvelope, error)
}

type NoOpBuilder struct{}

func (n *NoOpBuilder) GetPayload(_ context.Context, _ eth.L2BlockRef, _ log.Logger, _ BuilderMetrics) (*eth.ExecutionPayloadEnvelope, error) {
	return nil, errors.New("Builder not enabled")
}

func (n *NoOpBuilder) Enabled() bool {
	return false
}

func (n *NoOpBuilder) Timeout() time.Duration {
	return 0
}

var _ PayloadBuilder = &NoOpBuilder{}
