package source

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestSuperNodeSource_SyncStatus(t *testing.T) {
	t.Run("Single-Success", func(t *testing.T) {
		client := &mockSuperNodeClient{
			fn: func(_ context.Context, _ uint64) (eth.SuperRootAtTimestampResponse, error) {
				return eth.SuperRootAtTimestampResponse{
					CurrentL1:                 eth.BlockID{Hash: common.Hash{0xaa}, Number: 100},
					CurrentSafeTimestamp:      111,
					CurrentFinalizedTimestamp: 222,
				}, nil
			},
		}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client)
		status, err := source.SyncStatus(context.Background())
		require.NoError(t, err)
		require.Equal(t, SyncStatus{
			CurrentL1:   eth.BlockID{Hash: common.Hash{0xaa}, Number: 100},
			SafeL2:      111,
			FinalizedL2: 222,
		}, status)
	})

	t.Run("Multi-ReturnLowestCurrentL1", func(t *testing.T) {
		client1 := &mockSuperNodeClient{
			fn: func(_ context.Context, _ uint64) (eth.SuperRootAtTimestampResponse, error) {
				return eth.SuperRootAtTimestampResponse{
					CurrentL1:                 eth.BlockID{Hash: common.Hash{0x01}, Number: 999},
					CurrentSafeTimestamp:      9999,
					CurrentFinalizedTimestamp: 9999,
				}, nil
			},
		}
		client2 := &mockSuperNodeClient{
			fn: func(_ context.Context, _ uint64) (eth.SuperRootAtTimestampResponse, error) {
				return eth.SuperRootAtTimestampResponse{
					CurrentL1:                 eth.BlockID{Hash: common.Hash{0x02}, Number: 100},
					CurrentSafeTimestamp:      123,
					CurrentFinalizedTimestamp: 456,
				}, nil
			},
		}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client1, client2)
		status, err := source.SyncStatus(context.Background())
		require.NoError(t, err)
		require.Equal(t, SyncStatus{
			CurrentL1:   eth.BlockID{Hash: common.Hash{0x02}, Number: 100},
			SafeL2:      123,
			FinalizedL2: 456,
		}, status)
	})

	t.Run("Multi-SkipFailingClients", func(t *testing.T) {
		client1 := &mockSuperNodeClient{err: errors.New("test error")}
		client2 := &mockSuperNodeClient{
			fn: func(_ context.Context, _ uint64) (eth.SuperRootAtTimestampResponse, error) {
				return eth.SuperRootAtTimestampResponse{
					CurrentL1:                 eth.BlockID{Hash: common.Hash{0x02}, Number: 100},
					CurrentSafeTimestamp:      123,
					CurrentFinalizedTimestamp: 456,
				}, nil
			},
		}
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		source := NewSuperNodeProposalSource(logger, client1, client2)
		status, err := source.SyncStatus(context.Background())
		require.NoError(t, err)
		require.Equal(t, uint64(123), status.SafeL2)
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), testlog.NewAttributesFilter("err", client1.err.Error())))
	})
}

func TestSuperNodeSource_ProposalAtSequenceNum(t *testing.T) {
	chainA := eth.ChainIDFromUInt64(1)
	chainB := eth.ChainIDFromUInt64(2)
	timestamp := uint64(599)

	response := eth.SuperRootAtTimestampResponse{
		CurrentL1: eth.BlockID{
			Hash:   common.Hash{0x11},
			Number: 589111,
		},
		ChainIDs: []eth.ChainID{chainA, chainB},
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: eth.BlockID{
				Hash:   common.Hash{0x22},
				Number: 589100,
			},
			Super: eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{
				ChainID: chainA,
				Output:  eth.Bytes32{0xa1},
			}, eth.ChainIDAndOutput{
				ChainID: chainB,
				Output:  eth.Bytes32{0xa2},
			}),
			SuperRoot: eth.Bytes32{0xaa, 0xbb},
		},
	}

	expected := Proposal{
		Root:        common.Hash(response.Data.SuperRoot),
		SequenceNum: timestamp,
		CurrentL1: eth.BlockID{
			Hash:   response.CurrentL1.Hash,
			Number: response.CurrentL1.Number,
		},
		Legacy: LegacyProposalData{},
		Super:  response.Data.Super,
	}

	t.Run("Single-Success", func(t *testing.T) {
		client := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: response,
			},
		}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client)
		actual, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("Single-Error", func(t *testing.T) {
		expectedErr := errors.New("test error")
		client := &mockSuperNodeClient{
			err: expectedErr,
		}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client)
		_, err := source.ProposalAtSequenceNum(context.Background(), 294)
		require.ErrorIs(t, err, expectedErr)
	})

	t.Run("Single-NoData", func(t *testing.T) {
		client := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: {
					CurrentL1: eth.BlockID{
						Hash:   common.Hash{0x11},
						Number: 589111,
					},
					Data: nil, // No super root data
				},
			},
		}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client)
		_, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.ErrorIs(t, err, ErrNoSuperRootData)
	})

	t.Run("Multi-FirstSourceSuccess", func(t *testing.T) {
		client1 := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: response,
			},
		}
		client2 := &mockSuperNodeClient{}
		source := NewSuperNodeProposalSource(testlog.Logger(t, log.LvlInfo), client1, client2)
		actual, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		require.Equal(t, 1, client1.requestCount)
		require.Equal(t, 0, client2.requestCount)
	})

	t.Run("Multi-FailOverToSecondSource", func(t *testing.T) {
		client1 := &mockSuperNodeClient{
			err: errors.New("test error"),
		}
		client2 := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: response,
			},
		}
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		source := NewSuperNodeProposalSource(logger, client1, client2)
		actual, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		require.Equal(t, 1, client1.requestCount)
		require.Equal(t, 1, client2.requestCount)
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), testlog.NewAttributesFilter("err", client1.err.Error())))
	})

	t.Run("Multi-FailOverOnNoData", func(t *testing.T) {
		client1 := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: {
					CurrentL1: eth.BlockID{Hash: common.Hash{0x11}, Number: 100},
					Data:      nil, // No data
				},
			},
		}
		client2 := &mockSuperNodeClient{
			responses: map[uint64]eth.SuperRootAtTimestampResponse{
				timestamp: response,
			},
		}
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		source := NewSuperNodeProposalSource(logger, client1, client2)
		actual, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		require.Equal(t, 1, client1.requestCount)
		require.Equal(t, 1, client2.requestCount)
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), testlog.NewMessageContainsFilter("no super root data")))
	})

	t.Run("Multi-AllFail", func(t *testing.T) {
		client1 := &mockSuperNodeClient{
			err: errors.New("test error1"),
		}
		client2 := &mockSuperNodeClient{
			err: errors.New("test error2"),
		}
		logger, logs := testlog.CaptureLogger(t, log.LvlInfo)
		source := NewSuperNodeProposalSource(logger, client1, client2)
		_, err := source.ProposalAtSequenceNum(context.Background(), timestamp)
		require.ErrorIs(t, err, client1.err)
		require.ErrorIs(t, err, client2.err)
		require.Equal(t, 1, client1.requestCount)
		require.Equal(t, 1, client2.requestCount)
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), testlog.NewAttributesFilter("err", client1.err.Error())))
		require.NotNil(t, logs.FindLog(testlog.NewLevelFilter(slog.LevelWarn), testlog.NewAttributesFilter("err", client2.err.Error())))
	})
}

type mockSuperNodeClient struct {
	responses    map[uint64]eth.SuperRootAtTimestampResponse
	fn           func(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error)
	err          error
	requestCount int
}

func (m *mockSuperNodeClient) SuperRootAtTimestamp(ctx context.Context, timestamp uint64) (eth.SuperRootAtTimestampResponse, error) {
	m.requestCount++
	if m.fn != nil {
		return m.fn(ctx, timestamp)
	}
	if m.err != nil {
		return eth.SuperRootAtTimestampResponse{}, m.err
	}
	resp, ok := m.responses[timestamp]
	if !ok {
		return eth.SuperRootAtTimestampResponse{}, errors.New("timestamp not found in mock")
	}
	return resp, nil
}

func (m *mockSuperNodeClient) Close() {}
