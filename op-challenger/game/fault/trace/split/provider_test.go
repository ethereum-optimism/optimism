package split

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-node/testlog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockGetError   = fmt.Errorf("mock get error")
	mockOutput     = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	mockCommitment = common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func TestGet(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getError: mockGetError}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40, 20},
		}
		_, err := splitProvider.Get(context.Background(), types.NewPosition(1, 0))
		require.ErrorIs(t, err, mockGetError)
	})

	t.Run("ReturnsCorrectOutput", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getOutput: mockOutput}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40, 20},
		}
		output, err := splitProvider.Get(context.Background(), types.NewPosition(1, 0))
		require.NoError(t, err)
		require.Equal(t, mockOutput, output)
	})

	t.Run("ReturnsCorrectOutputWithMultipleProviders", func(t *testing.T) {
		firstOutputProvider := mockTraceProvider{}
		secondOutputProvider := mockTraceProvider{getOutput: mockOutput}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&firstOutputProvider, &secondOutputProvider},
			depthTiers: []uint64{40, 20},
		}
		output, err := splitProvider.Get(context.Background(), types.NewPosition(41, 0))
		require.NoError(t, err)
		require.Equal(t, mockOutput, output)
	})
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{absolutePreStateCommitmentError: mockGetError}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40, 20},
		}
		_, err := splitProvider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, mockGetError)
	})

	t.Run("ReturnsCorrectOutput", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{absolutePreStateCommitment: mockCommitment}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40, 20},
		}
		output, err := splitProvider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, mockCommitment, output)
	})
}

func TestAbsolutePreState(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{absolutePreStateError: mockGetError}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40},
		}
		_, err := splitProvider.AbsolutePreState(context.Background())
		require.ErrorIs(t, err, mockGetError)
	})
}

func TestGetStepData(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getStepDataError: mockGetError}
		splitProvider := SplitTraceProvider{
			logger:     testlog.Logger(t, log.LvlInfo),
			providers:  []types.TraceProvider{&mockOutputProvider},
			depthTiers: []uint64{40},
		}
		_, _, _, err := splitProvider.GetStepData(context.Background(), 0)
		require.ErrorIs(t, err, mockGetError)
	})
}

type mockTraceProvider struct {
	getOutput                       common.Hash
	getError                        error
	absolutePreStateCommitmentError error
	absolutePreStateCommitment      common.Hash
	absolutePreStateError           error
	getStepDataError                error
}

func (m *mockTraceProvider) Get(ctx context.Context, i uint64) (common.Hash, error) {
	if m.getError != nil {
		return common.Hash{}, m.getError
	}
	return m.getOutput, nil
}

func (m *mockTraceProvider) AbsolutePreStateCommitment(ctx context.Context) (hash common.Hash, err error) {
	if m.absolutePreStateCommitmentError != nil {
		return common.Hash{}, m.absolutePreStateCommitmentError
	}
	return m.absolutePreStateCommitment, nil
}

func (m *mockTraceProvider) AbsolutePreState(ctx context.Context) (preimage []byte, err error) {
	if m.absolutePreStateError != nil {
		return []byte{}, m.absolutePreStateError
	}
	return []byte{}, nil
}

func (m *mockTraceProvider) GetStepData(ctx context.Context, i uint64) ([]byte, []byte, *types.PreimageOracleData, error) {
	if m.getStepDataError != nil {
		return nil, nil, nil, m.getStepDataError
	}
	return nil, nil, nil, nil
}
