package cannon

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// cannonUpdater is a [types.OracleUpdater] that exposes a method
// to update onchain cannon oracles with required data.
type cannonUpdater struct {
	log   log.Logger
	txMgr txmgr.TxManager

	fdgAbi  *contracts.FaultDisputeGameAbi
	fdgAddr common.Address

	preimageOracleAbi  *contracts.PreimageOracleAbi
	preimageOracleAddr common.Address
}

// NewOracleUpdater returns a new updater. The pre-image oracle address is loaded from the fault dispute game.
func NewOracleUpdater(
	ctx context.Context,
	logger log.Logger,
	txMgr txmgr.TxManager,
	fdgAddr common.Address,
	gameAbi *contracts.FaultDisputeGameAbi,
	oracleAbi *contracts.PreimageOracleAbi,
	gameCaller *contracts.FaultDisputeGame,
) (*cannonUpdater, error) {
	oracleAddr, err := gameCaller.PreimageOracleAddr(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load pre-image oracle address from game %v: %w", fdgAddr, err)
	}
	return NewOracleUpdaterWithOracle(logger, txMgr, fdgAddr, gameAbi, oracleAddr, oracleAbi)
}

// NewOracleUpdaterWithOracle returns a new updater using a specified pre-image oracle address.
func NewOracleUpdaterWithOracle(
	logger log.Logger,
	txMgr txmgr.TxManager,
	fdgAddr common.Address,
	gameAbi *contracts.FaultDisputeGameAbi,
	preimageOracleAddr common.Address,
	preimageOracleAbi *contracts.PreimageOracleAbi,
) (*cannonUpdater, error) {
	return &cannonUpdater{
		log:   logger,
		txMgr: txMgr,

		fdgAbi:  gameAbi,
		fdgAddr: fdgAddr,

		preimageOracleAbi:  preimageOracleAbi,
		preimageOracleAddr: preimageOracleAddr,
	}, nil
}

// UpdateOracle updates the oracle with the given data.
func (u *cannonUpdater) UpdateOracle(ctx context.Context, data *types.PreimageOracleData) error {
	if data.IsLocal {
		return u.sendLocalOracleData(ctx, data)
	}
	return u.sendGlobalOracleData(ctx, data)
}

// sendLocalOracleData sends the local oracle data to the [txmgr].
func (u *cannonUpdater) sendLocalOracleData(ctx context.Context, data *types.PreimageOracleData) error {
	txData, err := u.fdgAbi.AddLocalData(data)
	if err != nil {
		return fmt.Errorf("local oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, u.fdgAddr, txData)
}

// sendGlobalOracleData sends the global oracle data to the [txmgr].
func (u *cannonUpdater) sendGlobalOracleData(ctx context.Context, data *types.PreimageOracleData) error {
	txData, err := u.preimageOracleAbi.GlobalOracleData(data)
	if err != nil {
		return fmt.Errorf("global oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, u.fdgAddr, txData)
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (u *cannonUpdater) sendTxAndWait(ctx context.Context, addr common.Address, txData []byte) error {
	receipt, err := u.txMgr.Send(ctx, txmgr.TxCandidate{
		To:       &addr,
		TxData:   txData,
		GasLimit: 0,
	})
	if err != nil {
		return err
	}
	if receipt.Status == ethtypes.ReceiptStatusFailed {
		u.log.Error("Responder tx successfully published but reverted", "tx_hash", receipt.TxHash)
	} else {
		u.log.Debug("Responder tx successfully published", "tx_hash", receipt.TxHash)
	}
	return nil
}
