package kvstore

import (
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-program/host/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestRecordAndReadKVFormat(t *testing.T) {
	for _, format := range types.SupportedDataFormats {
		format := format
		t.Run(string(format), func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, recordKVFormat(dir, format))
			actual, err := readKVFormat(dir)
			require.NoError(t, err)
			require.Equal(t, format, actual)
		})
	}

	t.Run("Unsupported", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, recordKVFormat(dir, "nope"))
		_, err := readKVFormat(dir)
		require.ErrorIs(t, err, ErrUnsupportedFormat)
	})

	t.Run("NotRecorded", func(t *testing.T) {
		dir := t.TempDir()
		_, err := readKVFormat(dir)
		require.ErrorIs(t, err, ErrFormatUnavailable)
	})
}

func TestNewDiskKV(t *testing.T) {
	for _, existingFormat := range types.SupportedDataFormats {
		existingFormat := existingFormat

		for _, specifiedFormat := range types.SupportedDataFormats {
			specifiedFormat := specifiedFormat
			t.Run(fmt.Sprintf("%v->%v", existingFormat, specifiedFormat), func(t *testing.T) {
				dir := t.TempDir()
				logger := testlog.Logger(t, log.LevelError)
				hash := common.Hash{0xaa}
				value := []byte{1, 2, 3, 4, 5, 6}
				kv1, err := NewDiskKV(logger, dir, existingFormat)
				require.NoError(t, err)
				require.NoError(t, kv1.Put(hash, value))
				require.NoError(t, kv1.Close())

				// Should use existing format
				kv2, err := NewDiskKV(logger, dir, specifiedFormat)
				require.NoError(t, err)
				actual, err := kv2.Get(hash)
				require.NoError(t, err)
				require.Equal(t, value, actual)
				require.NoError(t, kv2.Close())
			})
		}
	}
}
