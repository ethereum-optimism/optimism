package driver

import (
	"context"
	"errors"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/submit"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type DAManager struct {
	log    log.Logger
	engine derive.ResettableEngineControl

	TxMgr        *txmgr.SimpleTxManager
	RollupConfig *rollup.Config

	IsBroadcast bool
}

func NewDAManager(log log.Logger, rollup *rollup.Config, engine derive.ResettableEngineControl, txmgr *txmgr.SimpleTxManager, isBroadcast bool) *DAManager {
	return &DAManager{
		log:          log,
		engine:       engine,
		TxMgr:        txmgr,
		RollupConfig: rollup,
		IsBroadcast:  isBroadcast,
	}
}

func (d *DAManager) SendDA(ctx context.Context, index, length uint64, broadcaster, user common.Address, commitment, sign, data []byte) (common.Hash, error) {
	if !d.IsBroadcast {
		return common.Hash{}, errors.New("broadcast node not started")
	}
	d.log.Info("SendDA", "index", index, "length", length, "broadcaster", broadcaster.Hex(), "user", user.Hex(), "commitment", commitment, "sign", sign, "data", data)
	if !verifySignature(index, length, broadcaster, user, commitment, sign) {
		return common.Hash{}, errors.New("invalid public key")
	}
	input, err := submit.L1SubmitTxData(user, uint64(index), commitment, sign)
	if err != nil {
		return common.Hash{}, err
	}
	log.Info("L1SubmitTxData")

	tx, err := d.TxMgr.SendDA(ctx, txmgr.TxCandidate{
		TxData:   input,
		To:       &d.RollupConfig.SubmitContractAddress,
		GasLimit: 0,
	})
	log.Info("Send")

	if err != nil {
		return common.Hash{}, err
	}
	log.Info("L1Submit tx successfully published",
		"tx_hash", tx.Hash().Hex())

	d.engine.UploadFileDataByParams(ctx, index, length, broadcaster, user, commitment, sign, data, tx.Hash())
	return tx.Hash(), nil
}

func (d *DAManager) Broadcaster(ctx context.Context) (common.Address, error) {
	if d.IsBroadcast {
		return d.TxMgr.From(), nil
	}
	return common.Address{}, errors.New("broadcast node not started")
}

func (d *DAManager) loop() {

}
func verifySignature(index, length uint64, broadcaster, user common.Address, commitment, sign []byte) bool {
	return true
}
