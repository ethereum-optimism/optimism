package preimages

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var _ PreimageUploader = (*DirectPreimageUploader)(nil)

var ErrNilPreimageData = fmt.Errorf("cannot upload nil preimage data")

type PreimageOracleContract interface {
	UpdateOracleTx(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) (txmgr.TxCandidate, error)
}

// DirectPreimageUploader uploads the provided [types.PreimageOracleData]
// directly to the PreimageOracle contract in a single transaction.
type DirectPreimageUploader struct {
	log log.Logger

	txMgr    txmgr.TxManager
	contract PreimageOracleContract
}

func NewDirectPreimageUploader(logger log.Logger, txMgr txmgr.TxManager, contract PreimageOracleContract) *DirectPreimageUploader {
	return &DirectPreimageUploader{logger, txMgr, contract}
}

func (d *DirectPreimageUploader) UploadPreimage(ctx context.Context, claimIdx uint64, data *types.PreimageOracleData) error {
	if data == nil {
		return ErrNilPreimageData
	}
	d.log.Info("Updating oracle data", "key", data.OracleKey)
	candidate, err := d.contract.UpdateOracleTx(ctx, claimIdx, data)
	if err != nil {
		return fmt.Errorf("failed to create pre-image oracle tx: %w", err)
	}
	if err := d.sendTxAndWait(ctx, candidate); err != nil {
		return fmt.Errorf("failed to populate pre-image oracle: %w", err)
	}
	return nil
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (d *DirectPreimageUploader) sendTxAndWait(ctx context.Context, candidate txmgr.TxCandidate) error {
	receipt, err := d.txMgr.Send(ctx, candidate)
	if err != nil {
		return err
	}
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		d.log.Error("DirectPreimageUploader tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		d.log.Debug("DirectPreimageUploader tx successfully published", "tx_hash", receipt.TxHash)
	}
	return nil
}
