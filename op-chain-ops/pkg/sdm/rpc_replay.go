package sdm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// RPCReplayResult reports canonical RPC replay separately from receipt conformance.
// An unavailable required method is a failure, never an implicit skip/pass.
type RPCReplayResult struct {
	BlockNumber          uint64           `json:"block_number"`
	BlockHash            common.Hash      `json:"block_hash"`
	RefundedTransactions int              `json:"refunded_transactions"`
	Checks               []RPCReplayCheck `json:"checks"`
	Complete             bool             `json:"complete"`
}

type RPCReplayCheck struct {
	Name    string `json:"name"`
	Error   string `json:"error,omitempty"`
	Skipped string `json:"skipped,omitempty"`
}

// Missing fixture coverage is distinct from an unavailable/failed RPC method.
// It never counts as a passed check, even when other checks succeed.
type replayCoverageGap string

func (g replayCoverageGap) Error() string { return string(g) }

type replayHeader struct {
	Hash         common.Hash             `json:"hash"`
	ParentHash   common.Hash             `json:"parentHash"`
	Miner        common.Address          `json:"miner"`
	Transactions []rpcTransactionDetails `json:"transactions"`
}

type replayCallFrame struct {
	GasUsed *hexutil.Uint64 `json:"gasUsed"`
	Output  *hexutil.Bytes  `json:"output"`
	Error   string          `json:"error"`
}

type replayAccount struct {
	Balance *hexutil.Big `json:"balance"`
}

type replayDiff struct {
	Pre  map[common.Address]replayAccount `json:"pre"`
	Post map[common.Address]replayAccount `json:"post"`
}

// ValidateRPCReplay checks generic RPC replay against canonical receipts and
// debug_accountAt's executor-backed transaction-boundary balances. It intentionally
// does not use debug_replaySDMBlock: that diagnostic replay omits settlement by
// design and cannot serve as an oracle for refund-dependent transactions.
//
// See the check-lagoon README for the endpoint matrix and fixture requirements.
// Complete is false when any endpoint fails or a required fixture is absent.
func ValidateRPCReplay(ctx context.Context, rpc Caller, blockNum uint64) (*RPCReplayResult, error) {
	validation, err := ValidatePostExecBlock(ctx, rpc, blockNum, ValidationOptions{CheckReceipts: true})
	if err != nil {
		return nil, err
	}
	if err := validatePayloadEntries(validation); err != nil {
		return nil, err
	}
	block := validation.Block
	var header *replayHeader
	if err := rpc.CallContext(ctx, &header, "eth_getBlockByHash", block.Hash, true); err != nil {
		return nil, err
	}
	if header == nil || header.Hash != block.Hash || len(header.Transactions) != len(block.Transactions) {
		return nil, errors.New("canonical replay block identity/transaction count mismatch")
	}
	receipts := make([]*RPCReceipt, len(block.Transactions))
	for i, tx := range block.Transactions {
		if header.Transactions[i].Hash != tx.Hash {
			return nil, fmt.Errorf("transaction identity mismatch at index %d", i)
		}
		receipt, err := GetRPCReceipt(ctx, rpc, tx.Hash)
		if err != nil {
			return nil, err
		}
		if receipt.TxHash != tx.Hash || receipt.BlockHash != block.Hash || uint64(receipt.TransactionIndex) != uint64(i) {
			return nil, fmt.Errorf("receipt identity mismatch at index %d", i)
		}
		receipts[i] = receipt
	}
	result := &RPCReplayResult{BlockNumber: blockNum, BlockHash: block.Hash, RefundedTransactions: len(validation.Payload.GasRefundEntries)}
	var failures []error
	record := func(name string, err error) {
		check := RPCReplayCheck{Name: name}
		if gap, skipped := err.(replayCoverageGap); skipped {
			check.Skipped = string(gap)
		} else if err != nil {
			check.Error = err.Error()
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
		}
		result.Checks = append(result.Checks, check)
	}
	oracle := replayBalances{rpc: rpc, hash: header.Hash, parent: header.ParentHash, cache: make(map[balanceAt]*big.Int)}
	for i, tx := range header.Transactions {
		if ctx.Err() != nil {
			record("context", ctx.Err())
			break
		}
		if uint64(tx.Type) == optypes.DepositTxType || uint64(tx.Type) == optypes.PostExecTxType {
			continue
		}
		label := fmt.Sprintf("tx[%d] %s", i, tx.Hash)
		var frame replayCallFrame
		err := rpc.CallContext(ctx, &frame, "debug_traceTransaction", tx.Hash, map[string]any{"tracer": "callTracer"})
		if err == nil {
			err = checkCallGas(frame, receipts[i])
		}
		record(label+" debug_traceTransaction/callTracer", err)

		var defaultFrame struct {
			Gas    json.RawMessage `json:"gas"`
			Failed *bool           `json:"failed"`
		}
		err = rpc.CallContext(ctx, &defaultFrame, "debug_traceTransaction", tx.Hash, map[string]any{"disableStorage": true, "disableStack": true, "enableMemory": false})
		if err == nil {
			var gas uint64
			gas, err = traceGas(defaultFrame.Gas)
			if err == nil && gas != uint64(receipts[i].GasUsed) {
				err = fmt.Errorf("gas %d, canonical receipt gasUsed %d", gas, receipts[i].GasUsed)
			}
			if err == nil && (defaultFrame.Failed == nil || *defaultFrame.Failed != (uint64(receipts[i].Status) == 0)) {
				err = errors.New("missing or incorrect execution status")
			}
		}
		record(label+" debug_traceTransaction/default", err)

		addresses := []common.Address{tx.From, header.Miner, predeploys.BaseFeeVaultAddr, predeploys.L1FeeVaultAddr, predeploys.OperatorFeeVaultAddr}
		var diff replayDiff
		err = rpc.CallContext(ctx, &diff, "debug_traceTransaction", tx.Hash, map[string]any{"tracer": "prestateTracer", "tracerConfig": map[string]any{"diffMode": true}})
		if err == nil {
			err = checkReplayDiff(ctx, &oracle, i, addresses, diff)
		}
		record(label+" debug_traceTransaction/prestateTracer balances", err)

		// Probe the state AFTER each refunded transaction, including earlier refunds.
		// Overrides are ephemeral; this neither sends a transaction nor changes chain state.
		if receipts[i].OPGasRefund != nil {
			record(label+" debug_traceCall settled prefix", checkReplayPrefix(ctx, rpc, &oracle, i, addresses))
		}
	}
	for _, method := range []string{"debug_traceBlockByHash", "debug_traceBlockByNumber"} {
		var blockID any = block.Hash
		if method == "debug_traceBlockByNumber" {
			blockID = hexutil.EncodeUint64(blockNum)
		}
		var frames []struct {
			Result json.RawMessage `json:"result"`
			Error  string          `json:"error"`
			TxHash *common.Hash    `json:"txHash"`
		}
		err := rpc.CallContext(ctx, &frames, method, blockID, map[string]any{"tracer": "callTracer"})
		if err == nil && len(frames) != len(receipts) {
			err = fmt.Errorf("got %d traces, want %d", len(frames), len(receipts))
		}
		if err != nil {
			record(method+"/callTracer", err)
			continue
		}
		for i, trace := range frames {
			var frame replayCallFrame
			err := json.Unmarshal(trace.Result, &frame)
			if trace.Error != "" {
				err = errors.New(trace.Error)
			}
			if trace.TxHash != nil && *trace.TxHash != block.Transactions[i].Hash {
				err = errors.New("trace transaction hash mismatch")
			}
			if err == nil {
				err = checkCallGas(frame, receipts[i])
			}
			record(fmt.Sprintf("%s/callTracer tx[%d]", method, i), err)
		}
	}
	checkExtendedRPC(ctx, rpc, blockNum, header, receipts, &oracle, record)
	// Number-based APIs may have observed a reorg; never certify a different block.
	canonical, err := GetBlockWithTxs(ctx, rpc, blockNum)
	if err == nil && canonical.Hash != block.Hash {
		err = fmt.Errorf("block %d reorged: %s -> %s", blockNum, block.Hash, canonical.Hash)
	}
	record("canonical block unchanged", err)
	result.Complete = !slices.ContainsFunc(result.Checks, func(c RPCReplayCheck) bool { return c.Error != "" || c.Skipped != "" })
	return result, errors.Join(failures...)
}

func checkCallGas(frame replayCallFrame, receipt *RPCReceipt) error {
	if frame.GasUsed == nil {
		return errors.New("trace omits gasUsed")
	}
	if *frame.GasUsed != receipt.GasUsed {
		return fmt.Errorf("gasUsed %d, canonical receipt gasUsed %d (opGasRefund %s)", *frame.GasUsed, receipt.GasUsed, formatUint64Ptr(receipt.OPGasRefund))
	}
	if (frame.Error != "") != (uint64(receipt.Status) == 0) {
		return errors.New("trace execution status differs from receipt")
	}
	return nil
}

// Default tracer gas is a JSON integer on reth and may be a hex quantity on
// other RPC implementations. Missing/null/fractional values are not zero gas.
func traceGas(raw json.RawMessage) (uint64, error) {
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, errors.New("trace omits gas")
	}
	var gas uint64
	if err := json.Unmarshal(raw, &gas); err == nil {
		return gas, nil
	}
	var quantity hexutil.Uint64
	if err := json.Unmarshal(raw, &quantity); err != nil {
		return 0, fmt.Errorf("invalid trace gas: %w", err)
	}
	return uint64(quantity), nil
}

type balanceAt struct {
	index   int
	address common.Address
}
type replayBalances struct {
	rpc          Caller
	hash, parent common.Hash
	cache        map[balanceAt]*big.Int
}

// debug_accountAt uses a block-global inclusive index and canonical executor
// settlement. For a transaction at index zero, its preceding boundary is the
// parent state. Null means the account does not exist (zero balance).
func (o *replayBalances) at(ctx context.Context, index int, address common.Address) (*big.Int, error) {
	key := balanceAt{index, address}
	if value, ok := o.cache[key]; ok {
		return value, nil
	}
	value := new(big.Int)
	if index < 0 {
		var balance *hexutil.Big
		if err := o.rpc.CallContext(ctx, &balance, "eth_getBalance", address, map[string]any{"blockHash": o.parent, "requireCanonical": true}); err != nil {
			return nil, err
		}
		if balance == nil {
			return nil, errors.New("eth_getBalance returned null")
		}
		value.Set((*big.Int)(balance))
	} else {
		var account *replayAccount
		if err := o.rpc.CallContext(ctx, &account, "debug_accountAt", o.hash, index, address); err != nil {
			return nil, err
		}
		if account != nil {
			if account.Balance == nil {
				return nil, errors.New("debug_accountAt omits balance")
			}
			value.Set((*big.Int)(account.Balance))
		}
	}
	o.cache[key] = value
	return value, nil
}

func checkReplayDiff(ctx context.Context, oracle *replayBalances, index int, required []common.Address, diff replayDiff) error {
	if diff.Pre == nil || diff.Post == nil {
		return errors.New("trace omits pre/post diff maps")
	}
	addresses := make(map[common.Address]struct{})
	for _, address := range required {
		addresses[address] = struct{}{}
	}
	for address := range diff.Pre {
		addresses[address] = struct{}{}
	}
	for address := range diff.Post {
		addresses[address] = struct{}{}
	}
	// Stable error ordering keeps JSON reports and regression tests reproducible.
	ordered := make([]common.Address, 0, len(addresses))
	for address := range addresses {
		ordered = append(ordered, address)
	}
	slices.SortFunc(ordered, func(a, b common.Address) int { return bytes.Compare(a[:], b[:]) })
	var failures []error
	for _, address := range ordered {
		before, err := oracle.at(ctx, index-1, address)
		if err != nil {
			return fmt.Errorf("canonical prestate %s: %w", address, err)
		}
		after, err := oracle.at(ctx, index, address)
		if err != nil {
			return fmt.Errorf("canonical poststate %s: %w", address, err)
		}
		pre, preExists := diff.Pre[address]
		post, postExists := diff.Post[address]
		if !preExists && postExists && before.Sign() != 0 {
			failures = append(failures, fmt.Errorf("%s trace implies creation but canonical pre-balance is %s", address, before))
		}
		if preExists && !postExists && after.Sign() != 0 {
			failures = append(failures, fmt.Errorf("%s trace implies deletion but canonical post-balance is %s", address, after))
		}
		if pre.Balance != nil && (*big.Int)(pre.Balance).Cmp(before) != 0 {
			failures = append(failures, fmt.Errorf("%s pre-balance %s, canonical %s", address, (*big.Int)(pre.Balance), before))
		}
		if post.Balance != nil && (*big.Int)(post.Balance).Cmp(after) != 0 {
			failures = append(failures, fmt.Errorf("%s post-balance %s, canonical %s", address, (*big.Int)(post.Balance), after))
		}
		if before.Cmp(after) == 0 {
			continue
		}
		// Created/deleted accounts legitimately omit one map entry. Other changed
		// balances must be present, including fee vaults omitted by a broken tracer.
		if pre.Balance == nil && (preExists || before.Sign() != 0) {
			failures = append(failures, fmt.Errorf("%s missing changed pre-balance", address))
		}
		if post.Balance == nil && (postExists || after.Sign() != 0) {
			failures = append(failures, fmt.Errorf("%s missing changed post-balance", address))
		}
	}
	return errors.Join(failures...)
}

func balanceProbe(addresses []common.Address) (map[string]any, map[common.Address]any) {
	// BALANCE probes avoid inferring canonical state from the trace under test.
	// Each address returns one 32-byte word; all five offsets fit PUSH1.
	probe := common.HexToAddress("0x000000000000000000000000000000000000dead")
	code := make(hexutil.Bytes, 0, len(addresses)*25+5)
	for i, address := range addresses {
		code = append(code, 0x73) // PUSH20 address
		code = append(code, address[:]...)
		code = append(code, 0x31, 0x60, byte(i*32), 0x52) // BALANCE, PUSH1 offset, MSTORE
	}
	code = append(code, 0x60, byte(len(addresses)*32), 0x5f, 0xf3) // size, PUSH0, RETURN
	return map[string]any{"to": probe, "gas": "0x100000", "gasPrice": "0x0"}, map[common.Address]any{probe: map[string]any{"code": code}}
}

func checkReplayPrefix(ctx context.Context, rpc Caller, oracle *replayBalances, index int, addresses []common.Address) error {
	call, overrides := balanceProbe(addresses)
	var frame replayCallFrame
	err := rpc.CallContext(ctx, &frame, "debug_traceCall", call, oracle.hash,
		map[string]any{"tracer": "callTracer", "txIndex": hexutil.Uint64(index + 1), "stateOverrides": overrides})
	if err != nil {
		return err
	}
	if frame.Error != "" {
		return errors.New(frame.Error)
	}
	return checkBalanceOutput(ctx, oracle, index, addresses, frame.Output)
}

func checkBalanceOutput(ctx context.Context, oracle *replayBalances, index int, addresses []common.Address, output *hexutil.Bytes) error {
	if output == nil || len(*output) != len(addresses)*32 {
		return errors.New("balance probe returned missing/truncated output")
	}
	var failures []error
	for i, address := range addresses {
		expected, err := oracle.at(ctx, index, address)
		if err != nil {
			return err
		}
		actual := new(big.Int).SetBytes((*output)[i*32 : (i+1)*32])
		if actual.Cmp(expected) != 0 {
			failures = append(failures, fmt.Errorf("%s prefix balance %s, canonical %s", address, actual, expected))
		}
	}
	return errors.Join(failures...)
}
