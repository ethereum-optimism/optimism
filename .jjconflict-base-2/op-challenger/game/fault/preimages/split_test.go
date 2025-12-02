package preimages

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/stretchr/testify/require"
)

var mockLargePreimageSizeThreshold = uint64(100)

func TestSplitPreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("DirectUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t, mockLargePreimageSizeThreshold)
		err := oracle.UploadPreimage(context.Background(), 0, makePreimageData(nil, 0))
		require.NoError(t, err)
		require.Equal(t, 1, direct.updates)
		require.Equal(t, 0, large.updates)
	})

	t.Run("LocalDataUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t, mockLargePreimageSizeThreshold)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{IsLocal: true})
		require.NoError(t, err)
		require.Equal(t, 1, direct.updates)
		require.Equal(t, 0, large.updates)
	})

	t.Run("MaxSizeDirectUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t, mockLargePreimageSizeThreshold)
		err := oracle.UploadPreimage(context.Background(), 0, makePreimageData(make([]byte, mockLargePreimageSizeThreshold-1), 0))
		require.NoError(t, err)
		require.Equal(t, 1, direct.updates)
		require.Equal(t, 0, large.updates)
	})

	t.Run("LargeUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t, mockLargePreimageSizeThreshold)
		err := oracle.UploadPreimage(context.Background(), 0, makePreimageData(make([]byte, mockLargePreimageSizeThreshold), 0))
		require.NoError(t, err)
		require.Equal(t, 1, large.updates)
		require.Equal(t, 0, direct.updates)
	})

	t.Run("NilPreimageOracleData", func(t *testing.T) {
		oracle, _, _ := newTestSplitPreimageUploader(t, mockLargePreimageSizeThreshold)
		err := oracle.UploadPreimage(context.Background(), 0, nil)
		require.ErrorIs(t, err, ErrNilPreimageData)
	})
}

type mockPreimageUploader struct {
	updates     int
	uploadFails bool
}

func (s *mockPreimageUploader) UploadPreimage(ctx context.Context, parent uint64, data *types.PreimageOracleData) error {
	s.updates++
	if s.uploadFails {
		return mockUpdateOracleTxError
	}
	return nil
}

func newTestSplitPreimageUploader(t *testing.T, threshold uint64) (*SplitPreimageUploader, *mockPreimageUploader, *mockPreimageUploader) {
	direct := &mockPreimageUploader{}
	large := &mockPreimageUploader{}
	return NewSplitPreimageUploader(direct, large, threshold), direct, large
}
