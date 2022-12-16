package db

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Withdrawal contains transaction data for withdrawals made via the L2 to L1 bridge.
type Withdrawal struct {
	GUID        string
	TxHash      common.Hash
	L1Token     common.Address
	L2Token     common.Address
	FromAddress common.Address
	ToAddress   common.Address
	Amount      *big.Int
	Data        []byte
	LogIndex    uint
	BedrockHash *common.Hash
}

// String returns the tx hash for the withdrawal.
func (w Withdrawal) String() string {
	return w.TxHash.String()
}

// WithdrawalJSON contains Withdrawal data suitable for JSON serialization.
type WithdrawalJSON struct {
	GUID                     string          `json:"guid"`
	FromAddress              string          `json:"from"`
	ToAddress                string          `json:"to"`
	L1Token                  string          `json:"l1Token"`
	L2Token                  *Token          `json:"l2Token"`
	Amount                   string          `json:"amount"`
	Data                     []byte          `json:"data"`
	LogIndex                 uint64          `json:"logIndex"`
	BlockNumber              uint64          `json:"blockNumber"`
	BlockTimestamp           string          `json:"blockTimestamp"`
	TxHash                   string          `json:"transactionHash"`
	Batch                    *StateBatchJSON `json:"batch"`
	BedrockWithdrawalHash    *string         `json:"bedrockWithdrawalHash"`
	BedrockProvenTxHash      *string         `json:"bedrockProvenTxHash"`
	BedrockProvenLogIndex    *int            `json:"bedrockProvenLogIndex"`
	BedrockFinalizedTxHash   *string         `json:"bedrockFinalizedTxHash"`
	BedrockFinalizedLogIndex *int            `json:"bedrockFinalizedLogIndex"`
	BedrockFinalizedSuccess  *bool           `json:"bedrockFinalizedSuccess"`
}

type FinalizationState int

const (
	FinalizationStateAny FinalizationState = iota
	FinalizationStateFinalized
	FinalizationStateUnfinalized
)

func ParseFinalizationState(in string) FinalizationState {
	switch in {
	case "true":
		return FinalizationStateFinalized
	case "false":
		return FinalizationStateUnfinalized
	default:
		return FinalizationStateAny
	}
}

func (f FinalizationState) SQL() string {
	switch f {
	case FinalizationStateFinalized:
		return "AND withdrawals.br_withdrawal_finalized_tx_hash IS NOT NULL"
	case FinalizationStateUnfinalized:
		return "AND withdrawals.br_withdrawal_finalized_tx_hash IS NULL"
	}

	return ""
}

type ProvenWithdrawal struct {
	From           common.Address
	To             common.Address
	WithdrawalHash common.Hash
	TxHash         common.Hash
	LogIndex       uint
}

type FinalizedWithdrawal struct {
	WithdrawalHash common.Hash
	TxHash         common.Hash
	Success        bool
	LogIndex       uint
}
