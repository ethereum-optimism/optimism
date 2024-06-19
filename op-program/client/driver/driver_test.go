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
