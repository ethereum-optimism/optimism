package runner

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestNewRunnerSetsTimeout(t *testing.T) {
	timeout := 5 * time.Minute
	r := NewRunner(nil, nil, nil, timeout)
	require.Equal(t, timeout, r.vmTimeout)
}

func TestRunOnceAppliesTimeout(t *testing.T) {
	tests := []struct {
		name           string
		vmTimeout      time.Duration
		expectDeadline bool
	}{
		{
			name:           "timeout applied when set",
			vmTimeout:      100 * time.Millisecond,
			expectDeadline: true,
		},
		{
			name:           "no deadline when timeout is zero",
			vmTimeout:      0,
			expectDeadline: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedCtx context.Context

			mockCreator := func(
				ctx context.Context,
				_ log.Logger,
				_ vm.Metricer,
				_ *config.Config,
				_ prestateFetcher,
				_ gameTypes.GameType,
				_ utils.LocalGameInputs,
				_ string,
			) (types.TraceProvider, error) {
				capturedCtx = ctx
				return &stubTraceProvider{}, nil
			}

			r := &Runner{
				vmTimeout:            tt.vmTimeout,
				traceProviderCreator: mockCreator,
			}

			err := r.runOnce(context.Background(), log.New(), "test", gameTypes.CannonGameType, nil, utils.LocalGameInputs{}, "")
			require.NoError(t, err)

			_, hasDeadline := capturedCtx.Deadline()
			require.Equal(t, tt.expectDeadline, hasDeadline)
		})
	}
}

type stubTraceProvider struct{}

func (s *stubTraceProvider) Get(_ context.Context, _ types.Position) (common.Hash, error) {
	// Return a hash with VMStatusValid as the first byte
	var hash common.Hash
	hash[0] = mipsevm.VMStatusValid
	return hash, nil
}

func (s *stubTraceProvider) GetStepData(_ context.Context, _ types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	return nil, nil, nil, nil
}

func (s *stubTraceProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *stubTraceProvider) GetL2BlockNumberChallenge(_ context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	return nil, types.ErrL2BlockNumberValid
}

// slowTraceProvider blocks until context is done, simulating a slow VM
type slowTraceProvider struct{}

func (s *slowTraceProvider) Get(ctx context.Context, _ types.Position) (common.Hash, error) {
	<-ctx.Done()
	return common.Hash{}, ctx.Err()
}

func (s *slowTraceProvider) GetStepData(_ context.Context, _ types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	return nil, nil, nil, nil
}

func (s *slowTraceProvider) AbsolutePreStateCommitment(_ context.Context) (common.Hash, error) {
	return common.Hash{}, nil
}

func (s *slowTraceProvider) GetL2BlockNumberChallenge(_ context.Context) (*types.InvalidL2BlockNumberChallenge, error) {
	return nil, types.ErrL2BlockNumberValid
}

func TestRunOnceReturnsTimeoutError(t *testing.T) {
	mockCreator := func(
		_ context.Context,
		_ log.Logger,
		_ vm.Metricer,
		_ *config.Config,
		_ prestateFetcher,
		_ gameTypes.GameType,
		_ utils.LocalGameInputs,
		_ string,
	) (types.TraceProvider, error) {
		return &slowTraceProvider{}, nil
	}

	r := &Runner{
		vmTimeout:            50 * time.Millisecond,
		traceProviderCreator: mockCreator,
	}

	err := r.runOnce(context.Background(), log.New(), "test", gameTypes.CannonGameType, nil, utils.LocalGameInputs{}, "")
	require.ErrorIs(t, err, ErrVMTimeout)
	require.ErrorIs(t, err, context.DeadlineExceeded)
}
