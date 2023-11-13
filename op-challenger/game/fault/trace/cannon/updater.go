package cannon

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-service/txmgr"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type GameContract interface {
	VMAddr(ctx context.Context) (common.Address, error)
	AddLocalDataTx(data *types.PreimageOracleData) (txmgr.TxCandidate, error)
}

// cannonUpdater is a [types.OracleUpdater] that exposes a method
// to update onchain cannon oracles with required data.
type cannonUpdater struct {
	log   log.Logger
	txMgr txmgr.TxManager

	gameContract GameContract

	preimageOracleAbi  abi.ABI
	preimageOracleAddr common.Address
}

// NewOracleUpdater returns a new updater. The pre-image oracle address is loaded from the fault dispute game.
func NewOracleUpdater(
	ctx context.Context,
	logger log.Logger,
	txMgr txmgr.TxManager,
	client bind.ContractCaller,
	gameContract GameContract,
) (*cannonUpdater, error) {
	opts := &bind.CallOpts{Context: ctx}
	vm, err := gameContract.VMAddr(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load VM address: %w", err)
	}
	mipsCaller, err := bindings.NewMIPSCaller(vm, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create MIPS caller for address %v: %w", vm, err)
	}
	oracleAddr, err := mipsCaller.Oracle(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load pre-image oracle address: %w", err)
	}
	return NewOracleUpdaterWithOracle(logger, txMgr, gameContract, oracleAddr)
}

// NewOracleUpdaterWithOracle returns a new updater using a specified pre-image oracle address.
func NewOracleUpdaterWithOracle(
	logger log.Logger,
	txMgr txmgr.TxManager,
	gameContract GameContract,
	preimageOracleAddr common.Address,
) (*cannonUpdater, error) {
	preimageOracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	return &cannonUpdater{
		log:   logger,
		txMgr: txMgr,

		gameContract: gameContract,

		preimageOracleAbi:  *preimageOracleAbi,
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
	tx, err := u.gameContract.AddLocalDataTx(data)
	if err != nil {
		return fmt.Errorf("local oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, tx)
}

// sendGlobalOracleData sends the global oracle data to the [txmgr].
func (u *cannonUpdater) sendGlobalOracleData(ctx context.Context, data *types.PreimageOracleData) error {
	txData, err := u.BuildGlobalOracleData(data)
	if err != nil {
		return fmt.Errorf("global oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, txmgr.TxCandidate{
		To:       &u.preimageOracleAddr,
		TxData:   txData,
		GasLimit: 0,
	})
}

// BuildGlobalOracleData takes the global preimage key and data
// and creates tx data to load the key, data pair into the
// PreimageOracle contract.
func (u *cannonUpdater) BuildGlobalOracleData(data *types.PreimageOracleData) ([]byte, error) {
	return u.preimageOracleAbi.Pack(
		"loadKeccak256PreimagePart",
		big.NewInt(int64(data.OracleOffset)),
		data.GetPreimageWithoutSize(),
	)
}

// sendTxAndWait sends a transaction through the [txmgr] and waits for a receipt.
// This sets the tx GasLimit to 0, performing gas estimation online through the [txmgr].
func (u *cannonUpdater) sendTxAndWait(ctx context.Context, tx txmgr.TxCandidate) error {
	receipt, err := u.txMgr.Send(ctx, tx)
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
