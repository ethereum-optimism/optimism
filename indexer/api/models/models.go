package models

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
	Guid                 string `json:"guid"`
	From                 string `json:"from"`
	To                   string `json:"to"`
	TransactionHash      string `json:"transactionHash"`
	MessageHash          string `json:"messageHash"`
	Timestamp            uint64 `json:"timestamp"`
	L2BlockHash          string `json:"l2BlockHash"`
	Amount               string `json:"amount"`
	ProofTransactionHash string `json:"proofTransactionHash"`
	ClaimTransactionHash string `json:"claimTransactionHash"`
	L1TokenAddress       string `json:"l1TokenAddress"`
	L2TokenAddress       string `json:"l2TokenAddress"`
}

// WithdrawalResponse ... Data model for API JSON response
type WithdrawalResponse struct {
	Cursor      string           `json:"cursor"`
	HasNextPage bool             `json:"hasNextPage"`
	Items       []WithdrawalItem `json:"items"`
}
