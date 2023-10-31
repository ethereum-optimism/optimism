package models

import (
	"github.com/ethereum-optimism/optimism/indexer/database"
	"github.com/ethereum/go-ethereum/common"
)

// DepositItem ... Deposit item model for API responses
type DepositItem struct {
	Guid           string `json:"guid"`
	From           string `json:"from"`
	To             string `json:"to"`
	Timestamp      uint64 `json:"timestamp"`
	L1BlockHash    string `json:"l1BlockHash"`
	L1TxHash       string `json:"l1TxHash"`
	L2TxHash       string `json:"l2TxHash"`
	Amount         string `json:"amount"`
	L1TokenAddress string `json:"l1TokenAddress"`
	L2TokenAddress string `json:"l2TokenAddress"`
}

// DepositResponse ... Data model for API JSON response
type DepositResponse struct {
	Cursor      string        `json:"cursor"`
	HasNextPage bool          `json:"hasNextPage"`
	Items       []DepositItem `json:"items"`
}

// WithdrawalItem ... Data model for API JSON response
type WithdrawalItem struct {
	Guid                   string `json:"guid"`
	From                   string `json:"from"`
	To                     string `json:"to"`
	TransactionHash        string `json:"transactionHash"`
	CrossDomainMessageHash string `json:"crossDomainMessageHash"`
	Timestamp              uint64 `json:"timestamp"`
	L2BlockHash            string `json:"l2BlockHash"`
	Amount                 string `json:"amount"`
	L1ProvenTxHash         string `json:"l1ProvenTxHash"`
	L1FinalizedTxHash      string `json:"l1FinalizedTxHash"`
	L1TokenAddress         string `json:"l1TokenAddress"`
	L2TokenAddress         string `json:"l2TokenAddress"`
}

// WithdrawalResponse ... Data model for API JSON response
type WithdrawalResponse struct {
	Cursor      string           `json:"cursor"`
	HasNextPage bool             `json:"hasNextPage"`
	Items       []WithdrawalItem `json:"items"`
}

// FIXME make a pure function that returns a struct instead of newWithdrawalResponse
// newWithdrawalResponse ... Converts a database.L2BridgeWithdrawalsResponse to an api.WithdrawalResponse
func CreateWithdrawalResponse(withdrawals *database.L2BridgeWithdrawalsResponse) WithdrawalResponse {
	items := make([]WithdrawalItem, len(withdrawals.Withdrawals))
	for i, withdrawal := range withdrawals.Withdrawals {

		cdh := withdrawal.L2BridgeWithdrawal.CrossDomainMessageHash
		if cdh == nil { // Zero value indicates that the withdrawal didn't have a cross domain message
			cdh = &common.Hash{0}
		}

		item := WithdrawalItem{
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

	return WithdrawalResponse{
		Cursor:      withdrawals.Cursor,
		HasNextPage: withdrawals.HasNextPage,
		Items:       items,
	}
}
