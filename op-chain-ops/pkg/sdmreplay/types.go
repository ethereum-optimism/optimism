package sdmreplay

// Config controls a replay run.
type Config struct {
	RPCURL             string
	FromBlockSelector  string
	ToBlockSelector    string
	FromBlock          uint64
	ToBlock            uint64
	ComparePayload     bool
	CompareRPCReceipts bool
	FailOnMismatch     bool
	SkipEmptyBlocks    bool
	IncludeTrace       bool
	SummaryOnly        bool
	Workers            int
	Format             string
}

// RunConfigRecord describes the replay invocation and selected node mode.
type RunConfigRecord struct {
	Type               string `json:"type"`
	RPC                string `json:"rpc"`
	FromBlock          string `json:"from_block"`
	ToBlock            string `json:"to_block"`
	ResolvedFromBlock  uint64 `json:"resolved_from_block"`
	ResolvedToBlock    uint64 `json:"resolved_to_block"`
	HeadBlock          uint64 `json:"head_block"`
	ChainID            uint64 `json:"chain_id"`
	ClientVersion      string `json:"client_version"`
	ReplayMode         string `json:"replay_mode"`
	ComparePayload     bool   `json:"compare_payload"`
	CompareRPCReceipts bool   `json:"compare_rpc_receipts"`
	SummaryOnly        bool   `json:"summary_only"`
	IncludeTrace       bool   `json:"include_trace"`
}

// TxRecord is one per-transaction JSONL record.
type TxRecord struct {
	Type               string  `json:"type"`
	BlockNum           uint64  `json:"block_num"`
	TxIndex            int     `json:"tx_index"`
	ReplayTxIndex      int     `json:"replay_tx_index,omitempty"`
	TxHash             string  `json:"tx_hash"`
	TxType             string  `json:"tx_type"`
	From               string  `json:"from"`
	To                 string  `json:"to,omitempty"`
	GasUsed            uint64  `json:"gas_used"`
	OPGasRefundReplay  uint64  `json:"op_gas_refund_replay"`
	OPGasRefundReceipt uint64  `json:"op_gas_refund_receipt"`
	OPGasRefundPayload uint64  `json:"op_gas_refund_payload"`
	EffectiveGas       uint64  `json:"effective_gas"`
	RefundRatio        float64 `json:"refund_ratio"`
	Status             uint64  `json:"status"`
	IsSDMTx            bool    `json:"is_sdm_tx"`
	IsDepositTx        bool    `json:"is_deposit_tx"`
	Mismatch           bool    `json:"mismatch"`
	SstoreCount        uint64  `json:"sstore_count,omitempty"`
	SstoreGas          uint64  `json:"sstore_gas,omitempty"`
	SstoreRatio        float64 `json:"sstore_ratio,omitempty"`
	StorageHeavy       bool    `json:"storage_heavy,omitempty"`
	WallClockMicros    int64   `json:"wall_clock_micros,omitempty"`
	CalldataLen        int     `json:"calldata_len,omitempty"`
	AccountingSource   string  `json:"accounting_source,omitempty"`
}

// BlockRecord is one per-block JSONL record.
type BlockRecord struct {
	Type                   string  `json:"type"`
	BlockNum               uint64  `json:"block_num"`
	BlockHash              string  `json:"block_hash"`
	ParentHash             string  `json:"parent_hash"`
	TxCountTotal           int     `json:"tx_count_total"`
	TxCountUser            int     `json:"tx_count_user"`
	SDMTxPresent           bool    `json:"sdm_tx_present"`
	SDMPayloadEntryCount   int     `json:"sdm_payload_entry_count"`
	BlockGasUsed           uint64  `json:"block_gas_used"`
	BlockOPGasRefund       uint64  `json:"block_op_gas_refund"`
	BlockEffectiveGas      uint64  `json:"block_effective_gas"`
	BlockRefundRatio       float64 `json:"block_refund_ratio"`
	AvgRefundRatio         float64 `json:"avg_refund_ratio"`
	NodeReceiptRefundTotal uint64  `json:"node_receipt_refund_total"`
	ReplayRefundTotal      uint64  `json:"replay_refund_total"`
	PayloadRefundTotal     uint64  `json:"payload_refund_total"`
	MismatchCount          int     `json:"mismatch_count"`
	ReplayMode             string  `json:"replay_mode"`
}

// SummaryRecord aggregates the full run.
type SummaryRecord struct {
	Type                   string  `json:"type"`
	FromBlock              uint64  `json:"from_block"`
	ToBlock                uint64  `json:"to_block"`
	BlocksProcessed        int     `json:"blocks_processed"`
	BlocksSkipped          int     `json:"blocks_skipped"`
	BlocksWithSDMTx        int     `json:"blocks_with_sdm_tx"`
	TxCountTotal           int     `json:"tx_count_total"`
	TxCountUser            int     `json:"tx_count_user"`
	TotalGasUsed           uint64  `json:"total_gas_used"`
	ReplayRefundTotal      uint64  `json:"replay_refund_total"`
	NodeReceiptRefundTotal uint64  `json:"node_receipt_refund_total"`
	PayloadRefundTotal     uint64  `json:"payload_refund_total"`
	EffectiveGasTotal      uint64  `json:"effective_gas_total"`
	TotalRefundRatio       float64 `json:"total_refund_ratio"`
	AvgRefundRatio         float64 `json:"avg_refund_ratio"`
	MismatchCount          int     `json:"mismatch_count"`
	ReplayMode             string  `json:"replay_mode"`
}

// MismatchRecord describes one disagreement between block sources.
type MismatchRecord struct {
	Type     string `json:"type"`
	BlockNum uint64 `json:"block_num"`
	TxIndex  int    `json:"tx_index,omitempty"`
	TxHash   string `json:"tx_hash,omitempty"`
	Category string `json:"category"`
	Expected uint64 `json:"expected,omitempty"`
	Actual   uint64 `json:"actual,omitempty"`
	Message  string `json:"message"`
}

// BlockResult holds all emitted records for one block.
type BlockResult struct {
	Block      BlockRecord
	Txs        []TxRecord
	Mismatches []MismatchRecord
}

// RangeResult is the structured result before JSONL encoding.
type RangeResult struct {
	RunConfig RunConfigRecord
	Blocks    []BlockResult
	Summary   SummaryRecord
}
