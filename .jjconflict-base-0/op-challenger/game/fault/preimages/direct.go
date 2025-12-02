package preimages

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/log"
)

var _ PreimageUploader = (*DirectPreimageUploader)(nil)

type PreimageGameContract interface {
	UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error)
}

// DirectPreimageUploader uploads the provided [types.PreimageOracleData]
// directly to the PreimageOracle contract in a single transaction.
type DirectPreimageUploader struct {
	log log.Logger

	txSender TxSender
	contract PreimageGameContract
}

func NewDirectPreimageUploader(logger log.Logger, txSender TxSender, contract PreimageGameContract) *DirectPreimageUploader {
	return &DirectPreimageUploader{logger, txSender, contract}
}

func (d *DirectPreimageUploader) UploadPreimage(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) error {
	if data == nil {
		return ErrNilPreimageData
	}
	d.log.Info("Updating oracle data", "key", fmt.Sprintf("%x", data.OracleKey))
	candidate, err := d.contract.UpdateOracleTx(ctx, claimIdx, data)
	if err != nil {
		return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
	}
	if err := d.txSender.SendAndWaitSimple("populate pre-image oracle", candidate); err != nil {
		return fmt.Errorf("failed to populate pre-image oracle: %w", err)
	}
	return nil
}
