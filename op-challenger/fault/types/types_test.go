package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPreimageOracleData(t *testing.T) {
	t.Run("LocalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{1, 2, 3}, []byte{4, 5, 6})
		require.True(t, data.IsLocal)
		require.Equal(t, []byte{1, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
	})

	t.Run("GlobalData", func(t *testing.T) {
		data := NewPreimageOracleData([]byte{0, 2, 3}, []byte{4, 5, 6})
		require.False(t, data.IsLocal)
		require.Equal(t, []byte{0, 2, 3}, data.OracleKey)
		require.Equal(t, []byte{4, 5, 6}, data.OracleData)
	})
}
