package preimages

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var ErrNilPreimageData = fmt.Errorf("cannot upload nil preimage data")

// PreimageUploader is responsible for posting preimages.
type PreimageUploader interface {
	// UploadPreimage uploads the provided preimage.
	UploadPreimage(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) error
}

// PreimageOracleContract is the interface for interacting with the PreimageOracle contract.
type PreimageOracleContract interface {
	UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error)
}
