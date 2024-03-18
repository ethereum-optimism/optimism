package resolved

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var mockClaimResolutionError = errors.New("mock claim resolution error")

func TestValidatorScheduler_Schedule(t *testing.T) {
	tests := []struct {
		name                  string
		validateErr           error
		games                 []types.GameMetadata
		expectedValidateCalls int
	}{
		{
			name:                  "SingleGame_Succeeds",
			games:                 []types.GameMetadata{{}},
			expectedValidateCalls: 1,
		},
		{
			name:                  "SingleGame_Fails",
			validateErr:           mockClaimResolutionError,
			games:                 []types.GameMetadata{{}},
			expectedValidateCalls: 1,
		},
		{
			name:                  "MultipleGames_Succeed",
			games:                 []types.GameMetadata{{}, {}, {}},
			expectedValidateCalls: 1,
		},
		{
			name:                  "MultipleGames_Fails",
			validateErr:           mockClaimResolutionError,
			games:                 []types.GameMetadata{{}, {}, {}},
			expectedValidateCalls: 1,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			scheduler, validator := setupTestValidatorScheduler(t)
			validator.err = test.validateErr
			scheduler.Start(ctx)
			defer scheduler.Close()

			err := scheduler.Schedule(1, test.games)
			require.NoError(t, err)
			require.Eventually(t, func() bool {
				return int(validator.calls.Load()) == test.expectedValidateCalls
			}, 10*time.Second, 10*time.Millisecond)
		})
	}
}

func setupTestValidatorScheduler(t *testing.T) (*ValidatorScheduler, *stubValidator) {
	logger := testlog.Logger(t, log.LvlInfo)
	validator := &stubValidator{}
	scheduler := NewValidatorScheduler(logger, validator.Validate)
	return scheduler, validator
}

type stubValidator struct {
	calls atomic.Int64
	err   error
}

func (s *stubValidator) Validate(_ context.Context, _ uint64, _ []types.GameMetadata) error {
	s.calls.Add(1)
	return s.err
}
