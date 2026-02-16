package derive

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelTemporary, "temp"},
		{LevelReset, "reset"},
		{LevelCritical, "crit"},
		{Level(99), "unknown(99)"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestErrorError(t *testing.T) {
	t.Run("with underlying error", func(t *testing.T) {
		inner := errors.New("something broke")
		err := NewTemporaryError(inner)
		assert.Equal(t, "temp: something broke", err.Error())
	})

	t.Run("without underlying error", func(t *testing.T) {
		err := NewTemporaryError(nil)
		assert.Equal(t, "temp", err.Error())
	})

	t.Run("reset error message", func(t *testing.T) {
		err := NewResetError(fmt.Errorf("bad state"))
		assert.Equal(t, "reset: bad state", err.Error())
	})

	t.Run("critical error message", func(t *testing.T) {
		err := NewCriticalError(fmt.Errorf("fatal"))
		assert.Equal(t, "crit: fatal", err.Error())
	})
}

func TestErrorUnwrap(t *testing.T) {
	inner := errors.New("root cause")
	err := NewTemporaryError(inner)
	unwrapped := errors.Unwrap(err)
	assert.Equal(t, inner, unwrapped)
}

func TestErrorIs(t *testing.T) {
	t.Run("temporary matches temporary", func(t *testing.T) {
		err := NewTemporaryError(errors.New("conn refused"))
		require.True(t, errors.Is(err, ErrTemporary))
		require.False(t, errors.Is(err, ErrReset))
		require.False(t, errors.Is(err, ErrCritical))
	})

	t.Run("reset matches reset", func(t *testing.T) {
		err := NewResetError(errors.New("reorg"))
		require.True(t, errors.Is(err, ErrReset))
		require.False(t, errors.Is(err, ErrTemporary))
		require.False(t, errors.Is(err, ErrCritical))
	})

	t.Run("critical matches critical", func(t *testing.T) {
		err := NewCriticalError(errors.New("panic"))
		require.True(t, errors.Is(err, ErrCritical))
		require.False(t, errors.Is(err, ErrTemporary))
		require.False(t, errors.Is(err, ErrReset))
	})

	t.Run("does not match non-Error type", func(t *testing.T) {
		err := NewTemporaryError(errors.New("x"))
		require.False(t, errors.Is(err, errors.New("x")))
	})

	t.Run("nil comparison", func(t *testing.T) {
		e := Error{err: nil, level: LevelTemporary}
		assert.False(t, e.Is(nil))
	})
}

func TestNewError(t *testing.T) {
	err := NewError(errors.New("test"), LevelReset)
	var deriveErr Error
	require.True(t, errors.As(err, &deriveErr))
	assert.Equal(t, LevelReset, deriveErr.level)
}

func TestWrappedErrorChain(t *testing.T) {
	root := errors.New("connection timeout")
	wrapped := fmt.Errorf("fetching block: %w", root)
	err := NewTemporaryError(wrapped)

	require.True(t, errors.Is(err, root))
	require.True(t, errors.Is(err, ErrTemporary))
}
