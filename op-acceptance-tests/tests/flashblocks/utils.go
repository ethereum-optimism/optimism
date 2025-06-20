package flashblocks

import (
	"encoding/json"
	"strings"
)

type Flashblock struct {
	PayloadID string `json:"payload_id"`
	Index     int    `json:"index"`
	Diff      struct {
		StateRoot    string `json:"state_root"`
		ReceiptsRoot string `json:"receipts_root"`
		LogsBloom    string `json:"logs_bloom"`
		GasUsed      string `json:"gas_used"`
		BlockHash    string `json:"block_hash"`
		Transactions []any  `json:"transactions"`
		Withdrawals  []any  `json:"withdrawals"`
	} `json:"diff"`
	Metadata struct {
		BlockNumber        int                    `json:"block_number"`
		NewAccountBalances map[string]string      `json:"new_account_balances"`
		Receipts           map[string]interface{} `json:"receipts"`
	} `json:"metadata"`
}

type FlashblocksStreamMode string

const (
	FlashblocksStreamMode_Leader   FlashblocksStreamMode = "leader"
	FlashblocksStreamMode_Follower FlashblocksStreamMode = "follower"
)

// UnmarshalJSON implements custom unmarshaling for Flashblock to lower case the keys of .metadata.new_account_balances.
func (f *Flashblock) UnmarshalJSON(data []byte) error {
	type TempFlashblock Flashblock // need a type alias to avoid infinite recursion
	temp := (*TempFlashblock)(f)

	if err := json.Unmarshal(data, temp); err != nil {
		return err
	}
	if f.Metadata.NewAccountBalances == nil {
		return nil
	}

	loweredBalances := make(map[string]string)
	for key, value := range f.Metadata.NewAccountBalances {
		loweredBalances[strings.ToLower(key)] = value
	}
	f.Metadata.NewAccountBalances = loweredBalances

	return nil
}
