package claim

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type mockL2 struct {
	safeL2    eth.L2BlockRef
	safeL2Err error

	outputRoot    eth.Bytes32
	outputRootErr error

	requestedOutputRoot uint64
}

func (m *mockL2) L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error) {
	if label != eth.Safe {
		panic("unexpected usage")
	}
	if m.safeL2Err != nil {
		return eth.L2BlockRef{}, m.safeL2Err
	}
	return m.safeL2, nil
}

func (m *mockL2) L2OutputRoot(u uint64) (eth.Bytes32, error) {
	m.requestedOutputRoot = u
	if m.outputRootErr != nil {
		return eth.Bytes32{}, m.outputRootErr
	}
	return m.outputRoot, nil
}

var _ L2Source = (*mockL2)(nil)

func TestValidateClaim(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := eth.Bytes32{0x11}
		l2 := &mockL2{
			outputRoot: expected,
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(0), expected, l2)
		require.NoError(t, err)
	})

	t.Run("Valid-PriorToSafeHead", func(t *testing.T) {
		expected := eth.Bytes32{0x11}
		l2 := &mockL2{
			outputRoot: expected,
			safeL2: eth.L2BlockRef{
				Number: 10,
			},
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(20), expected, l2)
		require.NoError(t, err)
		require.Equal(t, uint64(10), l2.requestedOutputRoot)
	})

	t.Run("Invalid", func(t *testing.T) {
		l2 := &mockL2{
			outputRoot: eth.Bytes32{0x22},
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(0), eth.Bytes32{0x11}, l2)
		require.ErrorIs(t, err, ErrClaimNotValid)
	})

	t.Run("Invalid-PriorToSafeHead", func(t *testing.T) {
		l2 := &mockL2{
			outputRoot: eth.Bytes32{0x22},
			safeL2:     eth.L2BlockRef{Number: 10},
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(20), eth.Bytes32{0x55}, l2)
		require.ErrorIs(t, err, ErrClaimNotValid)
		require.Equal(t, uint64(10), l2.requestedOutputRoot)
	})

	t.Run("Error-safe-head", func(t *testing.T) {
		expectedErr := errors.New("boom")
		l2 := &mockL2{
			outputRoot: eth.Bytes32{0x11},
			safeL2:     eth.L2BlockRef{Number: 10},
			safeL2Err:  expectedErr,
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(0), eth.Bytes32{0x11}, l2)
		require.ErrorIs(t, err, expectedErr)
	})
	t.Run("Error-output-root", func(t *testing.T) {
		expectedErr := errors.New("boom")
		l2 := &mockL2{
			outputRoot:    eth.Bytes32{0x11},
			outputRootErr: expectedErr,
			safeL2:        eth.L2BlockRef{Number: 10},
		}
		logger := testlog.Logger(t, log.LevelError)
		err := ValidateClaim(logger, uint64(0), eth.Bytes32{0x11}, l2)
		require.ErrorIs(t, err, expectedErr)
	})
}
