package preimages

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

var _ PreimageUploader = (*SplitPreimageUploader)(nil)

// SplitPreimageUploader routes preimage uploads to the appropriate uploader
// based on the size of the preimage.
type SplitPreimageUploader struct {
	largePreimageSizeThreshold uint64
	directUploader             PreimageUploader
	largeUploader              PreimageUploader
}

func NewSplitPreimageUploader(directUploader PreimageUploader, largeUploader PreimageUploader, minLargePreimageSize uint64) *SplitPreimageUploader {
	return &SplitPreimageUploader{minLargePreimageSize, directUploader, largeUploader}
}

func (s *SplitPreimageUploader) UploadPreimage(ctx context.Context, parent uint64, data *types.PreimageOracleData) error {
	if data == nil {
		return ErrNilPreimageData
	}
	// Always route local preimage uploads to the direct uploader.
	// Large-preimage proposals are keccak-only on-chain, so non-keccak preimages must go through the direct path.
	isKeccak := len(data.OracleKey) > 0 && preimage.KeyType(data.OracleKey[0]) == preimage.Keccak256KeyType
	if data.IsLocal || !isKeccak || uint64(len(data.GetPreimageWithoutSize())) < s.largePreimageSizeThreshold {
		return s.directUploader.UploadPreimage(ctx, parent, data)
	} else {
		return s.largeUploader.UploadPreimage(ctx, parent, data)
	}
}
