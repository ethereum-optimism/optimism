package driver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
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
	require.ErrorIs(t, err, derive.ErrTemporary)
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

func TestNoError(t *testing.T) {
	driver := createDriver(t, nil)
	err := driver.Step(context.Background())
	require.NoError(t, err)
}

func TestValidateClaim(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		expected := eth.Bytes32{0x11}
		driver.l2OutputRoot = func() (eth.Bytes32, error) {
			return expected, nil
		}
		valid := driver.ValidateClaim(expected)
		require.True(t, valid)
	})

	t.Run("Invalid", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		driver.l2OutputRoot = func() (eth.Bytes32, error) {
			return eth.Bytes32{0x22}, nil
		}
		valid := driver.ValidateClaim(eth.Bytes32{0x11})
		require.False(t, valid)
	})

	t.Run("Error", func(t *testing.T) {
		driver := createDriver(t, io.EOF)
		driver.l2OutputRoot = func() (eth.Bytes32, error) {
			return eth.Bytes32{}, errors.New("boom")
		}
		valid := driver.ValidateClaim(eth.Bytes32{0x11})
		require.False(t, valid)
	})
}

func createDriver(t *testing.T, derivationResult error) *Driver {
	derivation := &stubDerivation{nextErr: derivationResult}
	return &Driver{
		logger:   testlog.Logger(t, log.LvlDebug),
		pipeline: derivation,
	}
}

type stubDerivation struct {
	nextErr error
}

func (s stubDerivation) Step(ctx context.Context) error {
	return s.nextErr
}

func (s stubDerivation) SafeL2Head() eth.L2BlockRef {
	return eth.L2BlockRef{}
}
