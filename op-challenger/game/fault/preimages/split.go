package preimages

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

var _ PreimageUploader = (*SplitPreimageUploader)(nil)

// PREIMAGE_SIZE_THRESHOLD is the size threshold for determining whether a preimage
// should be uploaded directly or through the large preimage uploader.
// TODO(client-pod#467): determine the correct size threshold to toggle between
//
//	the direct and large preimage uploaders.
const PREIMAGE_SIZE_THRESHOLD = 136 * 128

// SplitPreimageUploader routes preimage uploads to the appropriate uploader
// based on the size of the preimage.
type SplitPreimageUploader struct {
	directUploader PreimageUploader
	largeUploader  PreimageUploader
}

func NewSplitPreimageUploader(directUploader PreimageUploader, largeUploader PreimageUploader) *SplitPreimageUploader {
	return &SplitPreimageUploader{directUploader, largeUploader}
}

func (s *SplitPreimageUploader) UploadPreimage(ctx context.Context, parent uint64, data *types.PreimageOracleData) error {
	if data == nil {
		return ErrNilPreimageData
	}
	if len(data.OracleData) > PREIMAGE_SIZE_THRESHOLD {
		return s.largeUploader.UploadPreimage(ctx, parent, data)
	} else {
		return s.directUploader.UploadPreimage(ctx, parent, data)
	}
}
