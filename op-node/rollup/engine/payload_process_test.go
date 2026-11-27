package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestProcessPayload_InvalidDepositsOnlyDoesNotLoop covers a derived deposits-only block
// that the EL rejects (e.g. op-reth answering "links to previously rejected block").
// Requesting its deposits-only replacement rebuilds the same block, which the EL rejects
// again, forever. It must be a critical error instead, as in onBuildInvalid.
func TestProcessPayload_InvalidDepositsOnlyDoesNotLoop(t *testing.T) {
	cfg, _, refA1, payload := buildSimpleCfgAndPayload(t)
	holocene := uint64(0)
	cfg.HoloceneTime = &holocene

	mockEngine := &testutils.MockEngine{}
	mockEngine.ExpectNewPayload(payload.ExecutionPayload, nil, &eth.PayloadStatusV1{Status: eth.ExecutionInvalid}, nil)

	var events []string
	emitter := &testutils.MockEmitter{}
	emitter.Mock.On("Emit", mock.Anything).Run(func(args mock.Arguments) {
		switch args.Get(0).(type) {
		case derive.DepositsOnlyPayloadAttributesRequestEvent:
			events = append(events, "deposits-only-request")
		case rollup.CriticalErrorEvent:
			events = append(events, "critical")
		}
	})

	ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0), metrics.NoopMetrics,
		cfg, &sync.Config{}, &testutils.MockL1Source{}, emitter, nil)
	ec.onPayloadProcess(context.Background(), PayloadProcessEvent{
		Envelope:    payload,
		Ref:         refA1,
		DerivedFrom: eth.L1BlockRef{Number: 1, Time: refA1.Time},
	})

	mockEngine.AssertExpectations(t)
	require.Equal(t, []string{"critical"}, events)
}
