package preimages

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/stretchr/testify/require"
)

func TestSplitPreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("DirectUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		require.NoError(t, err)
		require.Equal(t, 1, direct.updates)
		require.Equal(t, 0, large.updates)
	})

	t.Run("LargeUploadSucceeds", func(t *testing.T) {
		oracle, direct, large := newTestSplitPreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{OracleData: make([]byte, PREIMAGE_SIZE_THRESHOLD+1)})
		require.NoError(t, err)
		require.Equal(t, 1, large.updates)
		require.Equal(t, 0, direct.updates)
	})

	t.Run("NilPreimageOracleData", func(t *testing.T) {
		oracle, _, _ := newTestSplitPreimageUploader(t)
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

func newTestSplitPreimageUploader(t *testing.T) (*SplitPreimageUploader, *mockPreimageUploader, *mockPreimageUploader) {
	direct := &mockPreimageUploader{}
	large := &mockPreimageUploader{}
	return NewSplitPreimageUploader(direct, large), direct, large
}
