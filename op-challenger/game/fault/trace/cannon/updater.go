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

// cannonUpdater is a [types.OracleUpdater] that exposes a method
// to update onchain cannon oracles with required data.
type cannonUpdater struct {
	log   log.Logger
	txMgr txmgr.TxManager

	fdgAbi  abi.ABI
	fdgAddr common.Address

	preimageOracleAbi  abi.ABI
	preimageOracleAddr common.Address
}

// NewOracleUpdater returns a new updater. The pre-image oracle address is loaded from the fault dispute game.
func NewOracleUpdater(
	ctx context.Context,
	logger log.Logger,
	txMgr txmgr.TxManager,
	fdgAddr common.Address,
	client bind.ContractCaller,
) (*cannonUpdater, error) {
	gameCaller, err := bindings.NewFaultDisputeGameCaller(fdgAddr, client)
	if err != nil {
		return nil, fmt.Errorf("create caller for game %v: %w", fdgAddr, err)
	}
	opts := &bind.CallOpts{Context: ctx}
	vm, err := gameCaller.VM(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load VM address from game %v: %w", fdgAddr, err)
	}
	mipsCaller, err := bindings.NewMIPSCaller(vm, client)
	if err != nil {
		return nil, fmt.Errorf("failed to create MIPS caller for address %v: %w", vm, err)
	}
	oracleAddr, err := mipsCaller.Oracle(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to load pre-image oracle address from game %v: %w", fdgAddr, err)
	}
	return NewOracleUpdaterWithOracle(logger, txMgr, fdgAddr, oracleAddr)
}

// NewOracleUpdaterWithOracle returns a new updater using a specified pre-image oracle address.
func NewOracleUpdaterWithOracle(
	logger log.Logger,
	txMgr txmgr.TxManager,
	fdgAddr common.Address,
	preimageOracleAddr common.Address,
) (*cannonUpdater, error) {
	fdgAbi, err := bindings.FaultDisputeGameMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	preimageOracleAbi, err := bindings.PreimageOracleMetaData.GetAbi()
	if err != nil {
		return nil, err
	}

	return &cannonUpdater{
		log:   logger,
		txMgr: txMgr,

		fdgAbi:  *fdgAbi,
		fdgAddr: fdgAddr,

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
	txData, err := u.BuildLocalOracleData(data)
	if err != nil {
		return fmt.Errorf("local oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, u.fdgAddr, txData)
}

// sendGlobalOracleData sends the global oracle data to the [txmgr].
func (u *cannonUpdater) sendGlobalOracleData(ctx context.Context, data *types.PreimageOracleData) error {
	txData, err := u.BuildGlobalOracleData(data)
	if err != nil {
		return fmt.Errorf("global oracle tx data build: %w", err)
	}
	return u.sendTxAndWait(ctx, u.fdgAddr, txData)
}

// BuildLocalOracleData takes the local preimage key and data
// and creates tx data to load the key, data pair into the
// PreimageOracle contract from the FaultDisputeGame contract call.
func (u *cannonUpdater) BuildLocalOracleData(data *types.PreimageOracleData) ([]byte, error) {
	return u.fdgAbi.Pack(
		"addLocalData",
		data.GetIdent(),
		big.NewInt(int64(data.LocalContext)),
		big.NewInt(int64(data.OracleOffset)),
	)
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
