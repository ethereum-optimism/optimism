package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDerivationComplete(t *testing.T) {
	driver := createDriver(t, fmt.Errorf("derivation complete: %w", io.EOF))
	err := driver.Step(context.Background())
	require.ErrorIs(t, err, io.EOF)
}

func TestTemporaryError(t *testing.T) {
	driver := createDriver(t, fmt.Errorf("whoopsie: %w", derive.ErrTemporary))
	err := driver.Step(context.Background())
	require.NoError(t, err, "should allow derivation to continue after temporary error")
}

func TestNotEnoughDataError(t *testing.T) {
	driver := createDriver(t, fmt.Errorf("idk: %w", derive.NotEnoughData))
	err := driver.Step(context.Background())
	require.NoError(t, err)
}

func TestGenericError(t *testing.T) {
	expected := errors.New("boom")
	driver := createDriver(t, expected)
	err := driver.Step(context.Background())
	require.ErrorIs(t, err, expected)
}

func TestTargetBlock(t *testing.T) {
	t.Run("Reached", func(t *testing.T) {
		driver := createDriverWithNextBlock(t, derive.NotEnoughData, 1000)
		driver.targetBlockNum = 1000
		err := driver.Step(context.Background())
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("Exceeded", func(t *testing.T) {
		driver := createDriverWithNextBlock(t, derive.NotEnoughData, 1000)
		driver.targetBlockNum = 500
		err := driver.Step(context.Background())
		require.ErrorIs(t, err, io.EOF)
	})

	t.Run("NotYetReached", func(t *testing.T) {
		driver := createDriverWithNextBlock(t, derive.NotEnoughData, 1000)
		driver.targetBlockNum = 1001
		err := driver.Step(context.Background())
		// No error to indicate derivation should continue
		require.NoError(t, err)
	})
}

func TestNoError(t *testing.T) {
	driver := createDriver(t, nil)
	err := driver.Step(context.Background())
	require.NoError(t, err)
}

func TestValidateClaim(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		expected := eth.Bytes32{0x11}
		driver.l2OutputRoot = func(_ uint64) (eth.Bytes32, error) {
			return expected, nil
		}
		err := driver.ValidateClaim(uint64(0), expected)
		require.NoError(t, err)
	})

	t.Run("Valid-PriorToSafeHead", func(t *testing.T) {
		driver := createDriverWithNextBlock(t, io.EOF, 10)
		expected := eth.Bytes32{0x11}
		requestedOutputRoot := uint64(0)
		driver.l2OutputRoot = func(blockNum uint64) (eth.Bytes32, error) {
			requestedOutputRoot = blockNum
			return expected, nil
		}
		err := driver.ValidateClaim(uint64(20), expected)
		require.NoError(t, err)
		require.Equal(t, uint64(10), requestedOutputRoot)
	})

	t.Run("Invalid", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		driver.l2OutputRoot = func(_ uint64) (eth.Bytes32, error) {
			return eth.Bytes32{0x22}, nil
		}
		err := driver.ValidateClaim(uint64(0), eth.Bytes32{0x11})
		require.ErrorIs(t, err, ErrClaimNotValid)
	})

	t.Run("Invalid-PriorToSafeHead", func(t *testing.T) {
		driver := createDriverWithNextBlock(t, io.EOF, 10)
		expected := eth.Bytes32{0x11}
		requestedOutputRoot := uint64(0)
		driver.l2OutputRoot = func(blockNum uint64) (eth.Bytes32, error) {
			requestedOutputRoot = blockNum
			return expected, nil
		}
		err := driver.ValidateClaim(uint64(20), eth.Bytes32{0x55})
		require.ErrorIs(t, err, ErrClaimNotValid)
		require.Equal(t, uint64(10), requestedOutputRoot)
	})

	t.Run("Error", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		expectedErr := errors.New("boom")
		driver.l2OutputRoot = func(_ uint64) (eth.Bytes32, error) {
			return eth.Bytes32{}, expectedErr
		}
		err := driver.ValidateClaim(uint64(0), eth.Bytes32{0x11})
		require.ErrorIs(t, err, expectedErr)
	})
}

func createDriver(t *testing.T, derivationResult error) *Driver {
	return createDriverWithNextBlock(t, derivationResult, 0)
}

func createDriverWithNextBlock(t *testing.T, derivationResult error, nextBlockNum uint64) *Driver {
	derivation := &stubDeriver{nextErr: derivationResult, nextBlockNum: nextBlockNum}
	return &Driver{
		logger:         testlog.Logger(t, log.LevelDebug),
		deriver:        derivation,
		l2OutputRoot:   nil,
		targetBlockNum: 1_000_000,
	}
}

type stubDeriver struct {
	nextErr      error
	nextBlockNum uint64
}

func (s *stubDeriver) SyncStep(ctx context.Context) error {
	return s.nextErr
}

func (s *stubDeriver) SafeL2Head() eth.L2BlockRef {
	return eth.L2BlockRef{
		Number: s.nextBlockNum,
	}
}
