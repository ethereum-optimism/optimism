package preimages

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestLargePreimageUploader_UploadPreimage(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		oracle, _, _ := newTestLargePreimageUploader(t)
		err := oracle.UploadPreimage(context.Background(), 0, &types.PreimageOracleData{})
		// todo(proofs#467): fix this to not error. See LargePreimageUploader.UploadPreimage.
		require.ErrorIs(t, err, errNotSupported)
	})
}

func newTestLargePreimageUploader(t *testing.T) (*LargePreimageUploader, *mockTxMgr, *mockPreimageOracleContract) {
	logger := testlog.Logger(t, log.LvlError)
	txMgr := &mockTxMgr{}
	contract := &mockPreimageOracleContract{}
	return NewLargePreimageUploader(logger, txMgr, contract), txMgr, contract
}
