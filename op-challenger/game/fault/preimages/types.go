package preimages

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
)

// PreimageUploader is responsible for posting preimages.
type PreimageUploader interface {
	// UploadPreimage uploads the provided preimage.
	UploadPreimage(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) error
}
