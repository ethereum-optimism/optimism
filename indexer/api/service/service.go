package service

import (
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/indexer/api/models"
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
)

type Service interface {
	GetDeposits(*models.Params) (*database.L1BridgeDepositsResponse, error)
	DepositResponse(*database.L1BridgeDepositsResponse) models.DepositResponse
	GetWithdrawals(params *models.Params) (*database.L2BridgeWithdrawalsResponse, error)
	WithdrawResponse(*database.L2BridgeWithdrawalsResponse) models.WithdrawalResponse
	GetSupplyInfo() (*models.BridgeSupplyView, error)

	QueryParams(a, l, c string) (*models.Params, error)
}

type ApiSvc struct {
	v      *Validator
	db     database.BridgeTransfersView
	logger log.Logger
}

func New(v *Validator, db database.BridgeTransfersView, l log.Logger) Service {
	return &ApiSvc{
		logger: l,
		v:      v,
		db:     db,
	}
}

func (svc *ApiSvc) QueryParams(a, c, l string) (*models.Params, error) {
	address, err := svc.v.ParseValidateAddress(a)
	if err != nil {
		svc.logger.Error("invalid address param", "param", a, "err", err)
		return nil, err
	}

	err = svc.v.ValidateCursor(c)
	if err != nil {
		svc.logger.Error("invalid cursor param", "cursor", c, "err", err)
		return nil, err
	}

	limit, err := svc.v.ParseValidateLimit(l)
	if err != nil {
		svc.logger.Error("invalid query param", "cursor", c, "err", err)
		return nil, err
	}

	return &models.Params{
		Address: address,
		Cursor:  c,
		Limit:   limit,
	}, nil

}

func (svc *ApiSvc) GetWithdrawals(params *models.Params) (*database.L2BridgeWithdrawalsResponse, error) {
	withdrawals, err := svc.db.L2BridgeWithdrawalsByAddress(params.Address, params.Cursor, params.Limit)
	if err != nil {
		svc.logger.Error("error getting withdrawals", "err", err.Error(), "address", params.Address.String())
		return nil, err
	}

	svc.logger.Debug("read withdrawals from db", "count", len(withdrawals.Withdrawals), "address", params.Address.String())
	return withdrawals, nil
}

func (svc *ApiSvc) WithdrawResponse(withdrawals *database.L2BridgeWithdrawalsResponse) models.WithdrawalResponse {
	items := make([]models.WithdrawalItem, len(withdrawals.Withdrawals))
	for i, withdrawal := range withdrawals.Withdrawals {

		cdh := withdrawal.L2BridgeWithdrawal.CrossDomainMessageHash
		if cdh == nil { // Zero value indicates that the withdrawal didn't have a cross domain message
			cdh = &common.Hash{0}
		}

		item := models.WithdrawalItem{
			Guid:                   withdrawal.L2BridgeWithdrawal.TransactionWithdrawalHash.String(),
			L2BlockHash:            withdrawal.L2BlockHash.String(),
			Timestamp:              withdrawal.L2BridgeWithdrawal.Tx.Timestamp,
			From:                   withdrawal.L2BridgeWithdrawal.Tx.FromAddress.String(),
			To:                     withdrawal.L2BridgeWithdrawal.Tx.ToAddress.String(),
			TransactionHash:        withdrawal.L2TransactionHash.String(),
			Amount:                 withdrawal.L2BridgeWithdrawal.Tx.Amount.String(),
			CrossDomainMessageHash: cdh.String(),
			L1ProvenTxHash:         withdrawal.ProvenL1TransactionHash.String(),
			L1FinalizedTxHash:      withdrawal.FinalizedL1TransactionHash.String(),
			L1TokenAddress:         withdrawal.L2BridgeWithdrawal.TokenPair.RemoteTokenAddress.String(),
			L2TokenAddress:         withdrawal.L2BridgeWithdrawal.TokenPair.LocalTokenAddress.String(),
		}
		items[i] = item
	}

	return models.WithdrawalResponse{
		Cursor:      withdrawals.Cursor,
		HasNextPage: withdrawals.HasNextPage,
		Items:       items,
	}
}

func (svc *ApiSvc) GetDeposits(params *models.Params) (*database.L1BridgeDepositsResponse, error) {
	deposits, err := svc.db.L1BridgeDepositsByAddress(params.Address, params.Cursor, params.Limit)
	if err != nil {
		svc.logger.Error("error getting deposits", "err", err.Error(), "address", params.Address.String())
		return nil, err
	}

	svc.logger.Debug("read deposits from db", "count", len(deposits.Deposits), "address", params.Address.String())
	return deposits, nil
}

// DepositResponse ... Converts a database.L1BridgeDepositsResponse to an api.DepositResponse
func (svc *ApiSvc) DepositResponse(deposits *database.L1BridgeDepositsResponse) models.DepositResponse {
	items := make([]models.DepositItem, len(deposits.Deposits))
	for i, deposit := range deposits.Deposits {
		item := models.DepositItem{
			Guid:           deposit.L1BridgeDeposit.TransactionSourceHash.String(),
			L1BlockHash:    deposit.L1BlockHash.String(),
			Timestamp:      deposit.L1BridgeDeposit.Tx.Timestamp,
			L1TxHash:       deposit.L1TransactionHash.String(),
			L2TxHash:       deposit.L2TransactionHash.String(),
			From:           deposit.L1BridgeDeposit.Tx.FromAddress.String(),
			To:             deposit.L1BridgeDeposit.Tx.ToAddress.String(),
			Amount:         deposit.L1BridgeDeposit.Tx.Amount.String(),
			L1TokenAddress: deposit.L1BridgeDeposit.TokenPair.LocalTokenAddress.String(),
			L2TokenAddress: deposit.L1BridgeDeposit.TokenPair.RemoteTokenAddress.String(),
		}
		items[i] = item
	}

	return models.DepositResponse{
		Cursor:      deposits.Cursor,
		HasNextPage: deposits.HasNextPage,
		Items:       items,
	}
}

// GetSupplyInfo ... Fetch native bridge supply info
func (svc *ApiSvc) GetSupplyInfo() (*models.BridgeSupplyView, error) {
	depositSum, err := svc.db.L1BridgeDepositSum()
	if err != nil {
		svc.logger.Error("error getting deposit sum", "err", err)
		return nil, err
	}

	initSum, err := svc.db.L2BridgeWithdrawalSum(models.All)
	if err != nil {
		svc.logger.Error("error getting init sum", "err", err)
		return nil, err
	}

	provenSum, err := svc.db.L2BridgeWithdrawalSum(models.Proven)
	if err != nil {
		svc.logger.Error("error getting proven sum", "err", err)
		return nil, err
	}

	finalizedSum, err := svc.db.L2BridgeWithdrawalSum(models.Finalized)
	if err != nil {
		svc.logger.Error("error getting finalized sum", "err", err)
		return nil, err
	}

	return &models.BridgeSupplyView{
		L1DepositSum:         depositSum,
		InitWithdrawalSum:    initSum,
		ProvenWithdrawSum:    provenSum,
		FinalizedWithdrawSum: finalizedSum,
	}, nil
}
