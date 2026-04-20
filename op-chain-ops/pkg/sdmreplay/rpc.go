package sdmreplay

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Client fetches blocks, receipts, and traces from an RPC endpoint.
type Client struct {
	url    string
	client *http.Client
}

// NewClient constructs an RPC client with a conservative timeout.
func NewClient(url string) *Client {
	return &Client{
		url: url,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

type jsonrpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int         `json:"id"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RPCTransaction is the minimal full-transaction block shape the replay tool needs.
type RPCTransaction struct {
	Hash  common.Hash     `json:"hash"`
	Type  hexutil.Uint64  `json:"type"`
	From  common.Address  `json:"from"`
	To    *common.Address `json:"to"`
	Input hexutil.Bytes   `json:"input"`
	Gas   hexutil.Uint64  `json:"gas"`
}

// RPCBlock is the minimal block shape the replay tool needs.
type RPCBlock struct {
	Number       hexutil.Uint64   `json:"number"`
	Hash         common.Hash      `json:"hash"`
	ParentHash   common.Hash      `json:"parentHash"`
	GasUsed      hexutil.Uint64   `json:"gasUsed"`
	Transactions []RPCTransaction `json:"transactions"`
}

// RPCReceipt is the minimal receipt shape the replay tool needs.
type RPCReceipt struct {
	TransactionHash  common.Hash     `json:"transactionHash"`
	TransactionIndex hexutil.Uint64  `json:"transactionIndex"`
	GasUsed          hexutil.Uint64  `json:"gasUsed"`
	Status           hexutil.Uint64  `json:"status"`
	OPGasRefund      *hexutil.Uint64 `json:"opGasRefund"`
}

// TracerResult is the per-tx result returned by storageProfileTracer.
type TracerResult struct {
	TxHash          string  `json:"txHash"`
	From            string  `json:"from"`
	To              *string `json:"to"`
	GasUsed         uint64  `json:"gasUsed"`
	OPGasRefund     uint64  `json:"opGasRefund"`
	EffectiveGas    uint64  `json:"effectiveGas"`
	RefundRatio     float64 `json:"refundRatio"`
	SstoreCount     uint64  `json:"sstoreCount"`
	SstoreGas       uint64  `json:"sstoreGas"`
	SstoreRatio     float64 `json:"sstoreRatio"`
	StorageHeavy    bool    `json:"storageHeavy"`
	WallClockMicros int64   `json:"wallClockMicros"`
	CalldataLen     int     `json:"calldataLen"`
	Status          uint64  `json:"status"`
}

// ReplaySdmBlockOptions configures debug_replaySDMBlock.
type ReplaySdmBlockOptions struct {
	ComparePayload  bool `json:"compare_payload"`
	CompareReceipts bool `json:"compare_receipts"`
}

// ReplaySdmConfig describes the replay mode used by the node.
type ReplaySdmConfig struct {
	Mode            string `json:"mode"`
	ComparePayload  bool   `json:"compare_payload"`
	CompareReceipts bool   `json:"compare_receipts"`
}

// ReplaySdmRefundEvent is one exact refund attribution event from debug_replaySDMBlock.
type ReplaySdmRefundEvent struct {
	ClaimingReplayTxIndex      uint64         `json:"claiming_replay_tx_index"`
	ClaimingTxIndex            uint64         `json:"claiming_tx_index"`
	Kind                       string         `json:"kind"`
	Amount                     uint64         `json:"amount"`
	Address                    common.Address `json:"address"`
	Slot                       *common.Hash   `json:"slot"`
	FirstWarmedByReplayTxIndex uint64         `json:"first_warmed_by_replay_tx_index"`
	FirstWarmedByTxIndex       uint64         `json:"first_warmed_by_tx_index"`
}

// ReplaySdmTx is the per-transaction output from debug_replaySDMBlock.
type ReplaySdmTx struct {
	TxIndex            uint64                 `json:"tx_index"`
	ReplayTxIndex      uint64                 `json:"replay_tx_index"`
	TxHash             common.Hash            `json:"tx_hash"`
	TxType             uint64                 `json:"tx_type"`
	IsDepositTx        bool                   `json:"is_deposit_tx"`
	GasUsed            uint64                 `json:"gas_used"`
	OPGasRefundReplay  uint64                 `json:"op_gas_refund_replay"`
	OPGasRefundPayload *uint64                `json:"op_gas_refund_payload"`
	OPGasRefundReceipt *uint64                `json:"op_gas_refund_receipt"`
	EffectiveGas       uint64                 `json:"effective_gas"`
	RefundBreakdown    []ReplaySdmRefundEvent `json:"refund_breakdown"`
	Mismatch           bool                   `json:"mismatch"`
}

// ReplaySdmMismatch is one mismatch row from debug_replaySDMBlock.
type ReplaySdmMismatch struct {
	Category string  `json:"category"`
	BlockNum uint64  `json:"block_num"`
	TxIndex  *uint64 `json:"tx_index"`
	Expected *uint64 `json:"expected"`
	Actual   *uint64 `json:"actual"`
	Message  string  `json:"message"`
}

// ReplaySdmSummary is the block-level summary from debug_replaySDMBlock.
type ReplaySdmSummary struct {
	BlockNum               uint64      `json:"block_num"`
	BlockHash              common.Hash `json:"block_hash"`
	TxCountTotal           int         `json:"tx_count_total"`
	TxCountUser            int         `json:"tx_count_user"`
	SDMTxPresent           bool        `json:"post_exec_tx_present"`
	SDMPayloadEntryCount   int         `json:"post_exec_payload_entry_count"`
	BlockGasUsed           uint64      `json:"block_gas_used"`
	ReplayRefundTotal      uint64      `json:"replay_refund_total"`
	PayloadRefundTotal     uint64      `json:"payload_refund_total"`
	NodeReceiptRefundTotal uint64      `json:"node_receipt_refund_total"`
	BlockEffectiveGas      uint64      `json:"block_effective_gas"`
	MismatchCount          int         `json:"mismatch_count"`
	ReplayMode             string      `json:"replay_mode"`
}

// ReplaySdmBlock is the full response from debug_replaySDMBlock.
type ReplaySdmBlock struct {
	Config       ReplaySdmConfig     `json:"config"`
	BlockNum     uint64              `json:"block_num"`
	BlockHash    common.Hash         `json:"block_hash"`
	ParentHash   common.Hash         `json:"parent_hash"`
	SDMTxPresent bool                `json:"post_exec_tx_present"`
	SDMTxIndex   *uint64             `json:"post_exec_tx_index"`
	Txs          []ReplaySdmTx       `json:"txs"`
	Mismatches   []ReplaySdmMismatch `json:"mismatches"`
	Summary      ReplaySdmSummary    `json:"summary"`
}

func (c *Client) CallContext(ctx context.Context, method string, params interface{}, out interface{}) error {
	req := jsonrpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	var rpcResp jsonrpcResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if rpcResp.Error != nil {
		return fmt.Errorf("rpc error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}
	if out == nil {
		return nil
	}
	if string(rpcResp.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(rpcResp.Result, out); err != nil {
		return fmt.Errorf("unmarshal result: %w", err)
	}
	return nil
}

// GetBlockNumber returns the current head block number.
func (c *Client) GetBlockNumber(ctx context.Context) (uint64, error) {
	var head string
	if err := c.CallContext(ctx, "eth_blockNumber", []interface{}{}, &head); err != nil {
		return 0, err
	}
	return ParseHexUint64(head)
}

// ClientVersion returns web3_clientVersion.
func (c *Client) ClientVersion(ctx context.Context) (string, error) {
	var version string
	if err := c.CallContext(ctx, "web3_clientVersion", []interface{}{}, &version); err != nil {
		return "", err
	}
	return version, nil
}

// ChainID returns the chain ID of the connected RPC.
func (c *Client) ChainID(ctx context.Context) (uint64, error) {
	var chainID string
	if err := c.CallContext(ctx, "eth_chainId", []interface{}{}, &chainID); err != nil {
		return 0, err
	}
	return ParseHexUint64(chainID)
}

// GetBlockByNumber fetches a block with full transaction objects.
func (c *Client) GetBlockByNumber(ctx context.Context, blockNum uint64) (*RPCBlock, error) {
	var raw json.RawMessage
	if err := c.CallContext(ctx, "eth_getBlockByNumber", []interface{}{fmt.Sprintf("0x%x", blockNum), true}, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("block %d not found", blockNum)
	}
	var block RPCBlock
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, fmt.Errorf("decode block %d: %w", blockNum, err)
	}
	return &block, nil
}

// GetTransactionReceipt fetches a transaction receipt, including opGasRefund if exposed.
func (c *Client) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*RPCReceipt, error) {
	var raw json.RawMessage
	if err := c.CallContext(ctx, "eth_getTransactionReceipt", []interface{}{txHash}, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("receipt %s not found", txHash.Hex())
	}
	var receipt RPCReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return nil, fmt.Errorf("decode receipt %s: %w", txHash.Hex(), err)
	}
	return &receipt, nil
}

// ReplaySdmBlock counterfactually replays a historical block through debug_replaySDMBlock.
func (c *Client) ReplaySdmBlock(
	ctx context.Context,
	blockNum uint64,
	comparePayload bool,
	compareReceipts bool,
) (*ReplaySdmBlock, error) {
	var replay ReplaySdmBlock
	err := c.CallContext(
		ctx,
		"debug_replaySDMBlock",
		[]interface{}{
			fmt.Sprintf("0x%x", blockNum),
			ReplaySdmBlockOptions{
				ComparePayload:  comparePayload,
				CompareReceipts: compareReceipts,
			},
		},
		&replay,
	)
	if err != nil {
		return nil, fmt.Errorf("replay block %d: %w", blockNum, err)
	}
	return &replay, nil
}

type traceBlockResult struct {
	TxHash string          `json:"txHash"`
	Result json.RawMessage `json:"result"`
}

// TraceBlock runs storageProfileTracer for the block and returns per-tx results.
func (c *Client) TraceBlock(ctx context.Context, blockNum uint64) ([]TracerResult, error) {
	params := []interface{}{
		fmt.Sprintf("0x%x", blockNum),
		map[string]string{"tracer": "storageProfileTracer"},
	}

	var blockResults []traceBlockResult
	if err := c.CallContext(ctx, "debug_traceBlockByNumber", params, &blockResults); err != nil {
		return nil, fmt.Errorf("trace block %d: %w", blockNum, err)
	}

	var profiles []TracerResult
	for _, br := range blockResults {
		var txProfiles []TracerResult
		if err := json.Unmarshal(br.Result, &txProfiles); err != nil {
			return nil, fmt.Errorf("decode tracer result for %s: %w", br.TxHash, err)
		}
		profiles = append(profiles, txProfiles...)
	}
	return profiles, nil
}
