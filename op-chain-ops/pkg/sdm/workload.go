package sdm

import (
	"context"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

// StateBloatBin is a tiny contract with run(uint256 n), which writes n stable storage slots.
// Sending repeated calls to the same contract in one block warms the same account/slots and
// should produce SDM refund entries in an SDM-enabled op-reth/op-rbuilder devnet.
const StateBloatBin = "6080604052348015600e575f5ffd5b5060f28061001b5f395ff3fe6080604052348015600e575f5ffd5b50600436106026575f3560e01c8063a444f5e914602a575b5f5ffd5b60406004803603810190603c91906096565b6042565b005b5f5f90505b8181101560605760018101815580806001019150506047565b5050565b5f5ffd5b5f819050919050565b6078816068565b81146081575f5ffd5b50565b5f813590506090816071565b92915050565b5f6020828403121560a85760a76064565b5b5f60b3848285016084565b9150509291505056fea2646970667358221220fb9ef6750b6ac6ded2dd901595e50b6daefe24726b41a0346f3a36ac6fcf5f8264736f6c634300081c0033"

const runSelectorHex = "a444f5e9" // run(uint256)

// EncodeRun returns calldata for StateBloat.run(n).
func EncodeRun(n uint64) []byte {
	selector, _ := hex.DecodeString(runSelectorHex)
	data := make([]byte, 4+32)
	copy(data[:4], selector)
	binary.BigEndian.PutUint64(data[len(data)-8:], n)
	return data
}

// Caller is the subset shared by go-ethereum RPC clients and op-service RPC wrappers.
type Caller interface {
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

// RPCTransaction is a minimal transaction object returned by eth_getBlockByNumber(..., true).
type RPCTransaction struct {
	Hash  common.Hash    `json:"hash"`
	Type  hexutil.Uint64 `json:"type"`
	Input hexutil.Bytes  `json:"input"`
}

// RPCBlock is a minimal block object returned by eth_getBlockByNumber(..., true).
type RPCBlock struct {
	Number       hexutil.Uint64   `json:"number"`
	Hash         common.Hash      `json:"hash"`
	GasUsed      hexutil.Uint64   `json:"gasUsed"`
	Transactions []RPCTransaction `json:"transactions"`
}

// RPCReceipt is a minimal raw JSON receipt. It avoids ethclient's typed receipt decoding, which
// may reject experimental post-exec receipt types before all client libraries know about them.
type RPCReceipt struct {
	TxHash           common.Hash     `json:"transactionHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	TransactionIndex hexutil.Uint64  `json:"transactionIndex"`
	ContractAddress  *common.Address `json:"contractAddress"`
	Status           hexutil.Uint64  `json:"status"`
	GasUsed          hexutil.Uint64  `json:"gasUsed"`
}

func (r *RPCReceipt) BlockNum() uint64 {
	if r == nil || r.BlockNumber == nil {
		return 0
	}
	return bigs.Uint64Strict((*big.Int)(r.BlockNumber))
}

type ReplaySDMRefundEvent struct {
	ClaimingReplayTxIndex      uint64         `json:"claiming_replay_tx_index"`
	ClaimingTxIndex            uint64         `json:"claiming_tx_index"`
	Kind                       string         `json:"kind"`
	Amount                     uint64         `json:"amount"`
	Address                    common.Address `json:"address"`
	Slot                       *common.Hash   `json:"slot"`
	FirstWarmedByReplayTxIndex uint64         `json:"first_warmed_by_replay_tx_index"`
	FirstWarmedByTxIndex       uint64         `json:"first_warmed_by_tx_index"`
}

type ReplaySDMTx struct {
	TxIndex            uint64                 `json:"tx_index"`
	ReplayTxIndex      uint64                 `json:"replay_tx_index"`
	TxHash             common.Hash            `json:"tx_hash"`
	TxType             uint64                 `json:"tx_type"`
	IsDepositTx        bool                   `json:"is_deposit_tx"`
	RawGasUsed         uint64                 `json:"raw_gas_used"`
	CanonicalGasUsed   uint64                 `json:"canonical_gas_used"`
	OPGasRefundReplay  uint64                 `json:"op_gas_refund_replay"`
	OPGasRefundPayload *uint64                `json:"op_gas_refund_payload"`
	RefundBreakdown    []ReplaySDMRefundEvent `json:"refund_breakdown"`
	Mismatch           bool                   `json:"mismatch"`
}

type ReplaySDMMismatch struct {
	Category string  `json:"category"`
	BlockNum uint64  `json:"block_num"`
	TxIndex  *uint64 `json:"tx_index"`
	Expected *uint64 `json:"expected"`
	Actual   *uint64 `json:"actual"`
	Message  string  `json:"message"`
}

type ReplaySDMSummary struct {
	BlockNum                  uint64      `json:"block_num"`
	BlockHash                 common.Hash `json:"block_hash"`
	TxCountTotal              int         `json:"tx_count_total"`
	TxCountUser               int         `json:"tx_count_user"`
	PostExecTxPresent         bool        `json:"post_exec_tx_present"`
	PostExecPayloadEntryCount int         `json:"post_exec_payload_entry_count"`
	BlockGasUsed              uint64      `json:"block_gas_used"`
	BlockRawGasUsed           uint64      `json:"block_raw_gas_used"`
	ReplayRefundTotal         uint64      `json:"replay_refund_total"`
	PayloadRefundTotal        uint64      `json:"payload_refund_total"`
	MismatchCount             int         `json:"mismatch_count"`
}

type ReplaySDMBlock struct {
	BlockNum                uint64              `json:"block_num"`
	BlockHash               common.Hash         `json:"block_hash"`
	ParentHash              common.Hash         `json:"parent_hash"`
	PostExecTxPresent       bool                `json:"post_exec_tx_present"`
	PostExecTxIndex         *uint64             `json:"post_exec_tx_index"`
	EmbeddedPayload         *PostExecPayload    `json:"embedded_payload"`
	SynthesizedPayload      PostExecPayload     `json:"synthesized_payload"`
	SynthesizedPayloadBytes hexutil.Bytes       `json:"synthesized_payload_bytes"`
	Txs                     []ReplaySDMTx       `json:"txs"`
	Mismatches              []ReplaySDMMismatch `json:"mismatches"`
	Summary                 ReplaySDMSummary    `json:"summary"`
}

// ValidationOptions controls how ValidatePostExecBlock checks the selected block.
type ValidationOptions struct {
	ComparePayload bool `json:"compare_payload"`
	CheckReceipts  bool `json:"check_receipts"`
	CheckReplay    bool `json:"check_replay"`
}

func DefaultValidationOptions() ValidationOptions {
	return ValidationOptions{ComparePayload: true, CheckReceipts: true, CheckReplay: true}
}

type ValidationResult struct {
	Block              *RPCBlock         `json:"block"`
	PostExecTx         *RPCTransaction   `json:"post_exec_tx"`
	PostExecIndex      int               `json:"post_exec_index"`
	Payload            *PostExecPayload  `json:"payload"`
	ReceiptRefunds     map[uint64]uint64 `json:"receipt_refunds,omitempty"`
	TotalPayloadRefund uint64            `json:"total_payload_refund"`
	Replay             *ReplaySDMBlock   `json:"replay,omitempty"`
}

func GetBlockWithTxs(ctx context.Context, rpcClient Caller, blockNum uint64) (*RPCBlock, error) {
	var raw json.RawMessage
	if err := rpcClient.CallContext(ctx, &raw, "eth_getBlockByNumber", fmt.Sprintf("0x%x", blockNum), true); err != nil {
		return nil, fmt.Errorf("eth_getBlockByNumber(%d): %w", blockNum, err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("block %d not found", blockNum)
	}

	var block RPCBlock
	if err := json.Unmarshal(raw, &block); err != nil {
		return nil, fmt.Errorf("unmarshal block %d: %w", blockNum, err)
	}
	return &block, nil
}

func FindPostExecTransaction(block *RPCBlock) (*RPCTransaction, int) {
	for i := range block.Transactions {
		tx := &block.Transactions[i]
		if uint64(tx.Type) == SDMTxType {
			return tx, i
		}
	}
	return nil, -1
}

func ReplayBlockWithSDM(ctx context.Context, rpcClient Caller, blockNum uint64, comparePayload bool) (*ReplaySDMBlock, error) {
	var raw json.RawMessage
	if err := rpcClient.CallContext(ctx, &raw, "debug_replaySDMBlock",
		fmt.Sprintf("0x%x", blockNum),
		map[string]bool{"compare_payload": comparePayload},
	); err != nil {
		return nil, fmt.Errorf("debug_replaySDMBlock(%d): %w", blockNum, err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("nil debug_replaySDMBlock result for block %d", blockNum)
	}

	var replay ReplaySDMBlock
	if err := json.Unmarshal(raw, &replay); err != nil {
		return nil, fmt.Errorf("unmarshal replay result for block %d: %w", blockNum, err)
	}
	return &replay, nil
}

func GetOPGasRefund(ctx context.Context, rpcClient Caller, txHash common.Hash) (uint64, bool, error) {
	var raw json.RawMessage
	if err := rpcClient.CallContext(ctx, &raw, "eth_getTransactionReceipt", txHash); err != nil {
		return 0, false, fmt.Errorf("eth_getTransactionReceipt(%s): %w", txHash, err)
	}
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false, fmt.Errorf("receipt %s not found", txHash)
	}

	var result struct {
		OPGasRefund *hexutil.Uint64 `json:"opGasRefund"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return 0, false, fmt.Errorf("unmarshal receipt %s: %w", txHash, err)
	}
	if result.OPGasRefund == nil {
		return 0, false, nil
	}
	return uint64(*result.OPGasRefund), true, nil
}

func ValidatePostExecBlock(ctx context.Context, rpcClient Caller, blockNum uint64, opts ValidationOptions) (*ValidationResult, error) {
	block, err := GetBlockWithTxs(ctx, rpcClient, blockNum)
	if err != nil {
		return nil, err
	}
	if len(block.Transactions) == 0 {
		return nil, fmt.Errorf("block %d has no transactions", blockNum)
	}

	postExecTx, postExecPos := FindPostExecTransaction(block)
	if postExecTx == nil {
		return nil, fmt.Errorf("block %d does not contain post-exec tx type 0x%x", blockNum, SDMTxType)
	}
	if postExecPos != len(block.Transactions)-1 {
		return nil, fmt.Errorf("block %d post-exec tx at index %d, want trailing index %d", blockNum, postExecPos, len(block.Transactions)-1)
	}
	payload, err := DecodePayload(postExecTx.Input)
	if err != nil {
		return nil, fmt.Errorf("decode block %d post-exec payload: %w", blockNum, err)
	}
	if payload.BlockNumber != blockNum {
		return nil, fmt.Errorf("post-exec payload block_number %d, want %d", payload.BlockNumber, blockNum)
	}
	if len(payload.GasRefundEntries) == 0 {
		return nil, fmt.Errorf("block %d post-exec payload has no SDM refund entries", blockNum)
	}

	result := &ValidationResult{
		Block:          block,
		PostExecTx:     postExecTx,
		PostExecIndex:  postExecPos,
		Payload:        payload,
		ReceiptRefunds: make(map[uint64]uint64, len(payload.GasRefundEntries)),
	}
	for _, entry := range payload.GasRefundEntries {
		result.TotalPayloadRefund += entry.GasRefund
		if int(entry.Index) >= len(block.Transactions) {
			return nil, fmt.Errorf("payload entry index %d out of range for block tx count %d", entry.Index, len(block.Transactions))
		}
		targetTx := block.Transactions[entry.Index]
		if uint64(targetTx.Type) == SDMTxType {
			return nil, fmt.Errorf("payload entry index %d targets post-exec tx", entry.Index)
		}
		if uint64(targetTx.Type) == types.DepositTxType {
			return nil, fmt.Errorf("payload entry index %d targets deposit tx", entry.Index)
		}
		if opts.CheckReceipts {
			refund, present, err := GetOPGasRefund(ctx, rpcClient, targetTx.Hash)
			if err != nil {
				return nil, err
			}
			if !present {
				return nil, fmt.Errorf("receipt %s at tx index %d does not expose opGasRefund", targetTx.Hash, entry.Index)
			}
			if refund != entry.GasRefund {
				return nil, fmt.Errorf("receipt opGasRefund %d for tx index %d, want payload refund %d", refund, entry.Index, entry.GasRefund)
			}
			result.ReceiptRefunds[entry.Index] = refund
		}
	}
	if !opts.CheckReceipts {
		result.ReceiptRefunds = nil
	}

	if opts.CheckReplay {
		replay, err := ReplayBlockWithSDM(ctx, rpcClient, blockNum, opts.ComparePayload)
		if err != nil {
			return nil, err
		}
		result.Replay = replay
		if !replay.PostExecTxPresent {
			return nil, fmt.Errorf("replay does not report post-exec tx for block %d", blockNum)
		}
		if replay.PostExecTxIndex == nil {
			return nil, fmt.Errorf("replay does not report post-exec tx index for block %d", blockNum)
		}
		if *replay.PostExecTxIndex != uint64(postExecPos) {
			return nil, fmt.Errorf("replay post-exec tx index %d, want %d", *replay.PostExecTxIndex, postExecPos)
		}
		if len(replay.Mismatches) > 0 {
			return nil, fmt.Errorf("replay reported %d SDM mismatch(es): %v", len(replay.Mismatches), replay.Mismatches)
		}
		if replay.Summary.PostExecPayloadEntryCount != len(payload.GasRefundEntries) {
			return nil, fmt.Errorf("replay payload entry count %d, want %d", replay.Summary.PostExecPayloadEntryCount, len(payload.GasRefundEntries))
		}
		if replay.Summary.ReplayRefundTotal != replay.Summary.PayloadRefundTotal {
			return nil, fmt.Errorf("replay refund total %d, payload total %d", replay.Summary.ReplayRefundTotal, replay.Summary.PayloadRefundTotal)
		}
		if replay.Summary.PayloadRefundTotal != result.TotalPayloadRefund {
			return nil, fmt.Errorf("replay payload refund total %d, decoded payload total %d", replay.Summary.PayloadRefundTotal, result.TotalPayloadRefund)
		}
	}

	return result, nil
}

type TxSender struct {
	RPC      *gethrpc.Client
	Eth      *ethclient.Client
	ChainID  *big.Int
	PrivKey  *ecdsa.PrivateKey
	From     common.Address
	GasLimit uint64
}

func DialTxSender(ctx context.Context, rpcURL string, priv *ecdsa.PrivateKey, from common.Address, gasLimit uint64) (*TxSender, error) {
	rpcClient, err := gethrpc.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial RPC %s: %w", rpcURL, err)
	}
	ethClient := ethclient.NewClient(rpcClient)
	chainID, err := ethClient.ChainID(ctx)
	if err != nil {
		rpcClient.Close()
		return nil, fmt.Errorf("eth_chainId: %w", err)
	}
	return &TxSender{RPC: rpcClient, Eth: ethClient, ChainID: chainID, PrivKey: priv, From: from, GasLimit: gasLimit}, nil
}

func (s *TxSender) Close() {
	if s.RPC != nil {
		s.RPC.Close()
	}
}

func (s *TxSender) SendContractCreation(ctx context.Context, nonce uint64, bytecode []byte, gasLimit uint64) (*types.Transaction, error) {
	return s.send(ctx, nonce, nil, nil, bytecode, gasLimit)
}

func (s *TxSender) SendCall(ctx context.Context, nonce uint64, to common.Address, data []byte, gasLimit uint64) (*types.Transaction, error) {
	return s.send(ctx, nonce, &to, nil, data, gasLimit)
}

func (s *TxSender) SendCallValue(ctx context.Context, nonce uint64, to common.Address, value *big.Int, data []byte, gasLimit uint64) (*types.Transaction, error) {
	return s.send(ctx, nonce, &to, value, data, gasLimit)
}

func (s *TxSender) send(ctx context.Context, nonce uint64, to *common.Address, value *big.Int, data []byte, gasLimit uint64) (*types.Transaction, error) {
	if value == nil {
		value = new(big.Int)
	}
	if gasLimit == 0 {
		gasLimit = s.GasLimit
	}

	var tx *types.Transaction
	header, err := s.Eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("latest header: %w", err)
	}
	if header.BaseFee != nil {
		tip, err := s.Eth.SuggestGasTipCap(ctx)
		if err != nil || tip == nil || tip.Sign() == 0 {
			tip = big.NewInt(1_000_000)
		}
		feeCap := new(big.Int).Mul(header.BaseFee, big.NewInt(2))
		feeCap.Add(feeCap, tip)
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:   s.ChainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: feeCap,
			Gas:       gasLimit,
			To:        to,
			Value:     value,
			Data:      data,
		})
	} else {
		gasPrice, err := s.Eth.SuggestGasPrice(ctx)
		if err != nil {
			return nil, fmt.Errorf("suggest gas price: %w", err)
		}
		tx = types.NewTx(&types.LegacyTx{
			Nonce:    nonce,
			GasPrice: gasPrice,
			Gas:      gasLimit,
			To:       to,
			Value:    value,
			Data:     data,
		})
	}

	signed, err := types.SignTx(tx, types.LatestSignerForChainID(s.ChainID), s.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("sign tx nonce %d: %w", nonce, err)
	}
	if err := s.Eth.SendTransaction(ctx, signed); err != nil {
		return nil, fmt.Errorf("send tx nonce %d: %w", nonce, err)
	}
	return signed, nil
}

// WaitReceipt polls for a transaction receipt until ctx is done.
func WaitReceipt(ctx context.Context, client *ethclient.Client, txHash common.Hash, pollInterval time.Duration) (*types.Receipt, error) {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err == nil {
			return receipt, nil
		}
		if !errors.Is(err, ethereum.NotFound) {
			return nil, fmt.Errorf("transaction receipt %s: %w", txHash, err)
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for receipt %s: %w", txHash, ctx.Err())
		case <-ticker.C:
		}
	}
}

// WaitRPCReceipt polls for a transaction receipt with raw JSON-RPC decoding.
func WaitRPCReceipt(ctx context.Context, rpcClient Caller, txHash common.Hash, pollInterval time.Duration) (*RPCReceipt, error) {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		var raw json.RawMessage
		err := rpcClient.CallContext(ctx, &raw, "eth_getTransactionReceipt", txHash)
		if err != nil {
			return nil, fmt.Errorf("eth_getTransactionReceipt(%s): %w", txHash, err)
		}
		if len(raw) != 0 && string(raw) != "null" {
			var receipt RPCReceipt
			if err := json.Unmarshal(raw, &receipt); err != nil {
				return nil, fmt.Errorf("unmarshal receipt %s: %w", txHash, err)
			}
			return &receipt, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("wait for receipt %s: %w", txHash, ctx.Err())
		case <-ticker.C:
		}
	}
}

func SortBlockNumsByTxCount[T any](blockTxs map[uint64][]T) []uint64 {
	blocks := make([]uint64, 0, len(blockTxs))
	for blockNum := range blockTxs {
		blocks = append(blocks, blockNum)
	}
	sort.Slice(blocks, func(i, j int) bool {
		if len(blockTxs[blocks[i]]) == len(blockTxs[blocks[j]]) {
			return blocks[i] < blocks[j]
		}
		return len(blockTxs[blocks[i]]) > len(blockTxs[blocks[j]])
	})
	return blocks
}

func DecodeHexBytes(input string) ([]byte, error) {
	trimmed := strings.TrimPrefix(input, "0x")
	if len(trimmed)%2 != 0 {
		trimmed = "0" + trimmed
	}
	return hex.DecodeString(trimmed)
}
