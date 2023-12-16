package models

import (
	"github.com/ethereum/go-ethereum/common"
)

type QueryParams struct {
	Address common.Address
	Limit   int
	Cursor  string
}

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

type BridgeSupplyView struct {
	L1DepositSum         float64 `json:"l1DepositSum"`
	InitWithdrawalSum    float64 `json:"l2WithdrawalSum"`
	ProvenWithdrawSum    float64 `json:"provenSum"`
	FinalizedWithdrawSum float64 `json:"finalizedSum"`
}
