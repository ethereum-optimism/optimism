package split

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var (
	mockGetError   = errors.New("mock get error")
	mockOutput     = common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	mockCommitment = common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
)

func TestGet(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getError: mockGetError}
		splitProvider := newSplitTraceProvider(t, &mockOutputProvider, nil, 40)
		_, err := splitProvider.Get(context.Background(), types.NewPosition(1, common.Big0))
		require.ErrorIs(t, err, mockGetError)
	})

	t.Run("ReturnsCorrectOutputFromTopProvider", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getOutput: mockOutput}
		splitProvider := newSplitTraceProvider(t, &mockOutputProvider, &mockTraceProvider{}, 40)
		output, err := splitProvider.Get(context.Background(), types.NewPosition(6, big.NewInt(3)))
		require.NoError(t, err)
		expectedGIndex := types.NewPosition(6, big.NewInt(3)).ToGIndex()
		require.Equal(t, common.BigToHash(expectedGIndex), output)
	})

	t.Run("ReturnsCorrectOutputWithMultipleProviders", func(t *testing.T) {
		bottomProvider := mockTraceProvider{getOutput: mockOutput}
		splitProvider := newSplitTraceProvider(t, &mockTraceProvider{}, &bottomProvider, 40)
		output, err := splitProvider.Get(context.Background(), types.NewPosition(42, big.NewInt(17)))
		require.NoError(t, err)
		expectedGIndex := types.NewPosition(2, big.NewInt(1)).ToGIndex()
		require.Equal(t, common.BigToHash(expectedGIndex), output)
	})
}

func TestAbsolutePreStateCommitment(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{absolutePreStateCommitmentError: mockGetError}
		splitProvider := newSplitTraceProvider(t, nil, &mockOutputProvider, 40)
		_, err := splitProvider.AbsolutePreStateCommitment(context.Background())
		require.ErrorIs(t, err, mockGetError)
	})

	t.Run("ReturnsCorrectOutput", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{absolutePreStateCommitment: mockCommitment}
		splitProvider := newSplitTraceProvider(t, nil, &mockOutputProvider, 40)
		output, err := splitProvider.AbsolutePreStateCommitment(context.Background())
		require.NoError(t, err)
		require.Equal(t, mockCommitment, output)
	})
}

func TestGetStepData(t *testing.T) {
	t.Run("ErrorBubblesUp", func(t *testing.T) {
		mockOutputProvider := mockTraceProvider{getStepDataError: mockGetError}
		splitProvider := newSplitTraceProvider(t, &mockOutputProvider, nil, 40)
		_, _, _, err := splitProvider.GetStepData(context.Background(), types.NewPosition(0, common.Big0))
		require.ErrorIs(t, err, mockGetError)
	})

	t.Run("ReturnsCorrectStepData", func(t *testing.T) {
		expectedStepData := []byte{1, 2, 3, 4}
		mockOutputProvider := mockTraceProvider{stepPrestateData: expectedStepData}
		splitProvider := newSplitTraceProvider(t, nil, &mockOutputProvider, 40)
		output, _, _, err := splitProvider.GetStepData(context.Background(), types.NewPosition(41, common.Big0))
		require.NoError(t, err)
		require.Equal(t, expectedStepData, output)
	})
}

type mockTraceProvider struct {
	getOutput                       common.Hash
	getError                        error
	absolutePreStateCommitmentError error
	absolutePreStateCommitment      common.Hash
	absolutePreStateError           error
	preImageData                    []byte
	getStepDataError                error
	stepPrestateData                []byte
}

func newSplitTraceProvider(t *testing.T, tp *mockTraceProvider, bp *mockTraceProvider, topDepth uint64) SplitTraceProvider {
	return SplitTraceProvider{
		logger:         testlog.Logger(t, log.LvlInfo),
		topProvider:    tp,
		bottomProvider: bp,
		topDepth:       topDepth,
	}
}

func (m *mockTraceProvider) Get(ctx context.Context, pos types.Position) (common.Hash, error) {
	if m.getError != nil {
		return common.Hash{}, m.getError
	}
	return common.BigToHash(pos.ToGIndex()), nil
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
	return m.preImageData, nil
}

func (m *mockTraceProvider) GetStepData(ctx context.Context, pos types.Position) ([]byte, []byte, *types.PreimageOracleData, error) {
	if m.getStepDataError != nil {
		return nil, nil, nil, m.getStepDataError
	}
	return m.stepPrestateData, nil, nil, nil
}
