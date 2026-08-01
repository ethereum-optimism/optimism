package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/metrics"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	opsigner "github.com/ethereum-optimism/optimism/op-service/signer"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// TestCommitBlockRejectsNonAcceptedStatus checks that CommitBlock only advances the unsafe
// head for payload statuses it explicitly accepts. Statuses that are invalid, or that this
// version does not recognize, must be reported as an error and leave the head untouched.
func TestCommitBlockRejectsNonAcceptedStatus(t *testing.T) {
	validationErr := "bad state root"
	for _, tc := range []struct {
		name   string
		status *eth.PayloadStatusV1
	}{
		{
			name:   "invalid",
			status: &eth.PayloadStatusV1{Status: eth.ExecutionInvalid, ValidationError: &validationErr},
		},
		{
			name:   "invalid terminal block",
			status: &eth.PayloadStatusV1{Status: eth.ExecutionInvalidTerminalBlock},
		},
		{
			name:   "unrecognized status",
			status: &eth.PayloadStatusV1{Status: eth.ExecutePayloadStatus("SOMETHING_NEW")},
		},
		{
			name:   "empty status",
			status: &eth.PayloadStatusV1{},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, refA0, _, payloadA1 := buildSimpleCfgAndPayload(t)
			mockEngine := &testutils.MockEngine{}
			emitter := &testutils.MockEmitter{}
			ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
				metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync},
				&testutils.MockL1Source{}, emitter, nil)
			ec.SetUnsafeHead(refA0)

			mockEngine.ExpectNewPayload(payloadA1.ExecutionPayload, nil, tc.status, nil)

			err := ec.CommitBlock(context.Background(), &opsigner.SignedExecutionPayloadEnvelope{
				Envelope: payloadA1,
			})

			require.Error(t, err)
			// The unsafe head must not have moved to the rejected payload.
			require.Equal(t, refA0, ec.UnsafeL2Head())
			mockEngine.AssertExpectations(t)
			emitter.AssertExpectations(t)
		})
	}

	t.Run("invalid status reports the engine's validation error", func(t *testing.T) {
		cfg, refA0, _, payloadA1 := buildSimpleCfgAndPayload(t)
		mockEngine := &testutils.MockEngine{}
		emitter := &testutils.MockEmitter{}
		ec := NewEngineController(context.Background(), mockEngine, testlog.Logger(t, 0),
			metrics.NoopMetrics, cfg, &sync.Config{SyncMode: sync.CLSync},
			&testutils.MockL1Source{}, emitter, nil)
		ec.SetUnsafeHead(refA0)

		mockEngine.ExpectNewPayload(payloadA1.ExecutionPayload, nil,
			&eth.PayloadStatusV1{Status: eth.ExecutionInvalid, ValidationError: &validationErr}, nil)

		err := ec.CommitBlock(context.Background(), &opsigner.SignedExecutionPayloadEnvelope{
			Envelope: payloadA1,
		})

		require.ErrorContains(t, err, validationErr)
		mockEngine.AssertExpectations(t)
		emitter.AssertExpectations(t)
	})
}
