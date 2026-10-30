package sdm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// The JS result consumes the executor-supplied result, not raw opcode gas totals.
const replayJSTracer = `{step:function(){},fault:function(){},result:function(ctx,db){return {gas:ctx.gasUsed,failed:!!ctx.error,output:toHex(ctx.output)}}}`

type executionCall struct {
	replayCallFrame
	From  common.Address  `json:"from"`
	To    common.Address  `json:"to"`
	Type  string          `json:"type"`
	Input hexutil.Bytes   `json:"input"`
	Value *hexutil.Big    `json:"value"`
	Calls []executionCall `json:"calls"`
}
type otsEntry struct {
	Type   string         `json:"type"`
	Depth  *uint64        `json:"depth"`
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Input  hexutil.Bytes  `json:"input"`
	Output hexutil.Bytes  `json:"output"`
	Value  *hexutil.Big   `json:"value"`
}

func checkJSAndOTS(ctx context.Context, rpc Caller, tx rpcTransactionDetails, receipt *RPCReceipt, label string, record replayRecorder) bool {
	var frame executionCall
	frameErr := rpc.CallContext(ctx, &frame, "debug_traceTransaction", tx.Hash, map[string]any{"tracer": "callTracer"})
	var js struct {
		Gas    json.RawMessage `json:"gas"`
		Failed *bool           `json:"failed"`
		Output *hexutil.Bytes  `json:"output"`
	}
	err := rpc.CallContext(ctx, &js, "debug_traceTransaction", tx.Hash, map[string]any{"tracer": replayJSTracer})
	if err == nil {
		gas, gasErr := traceGas(js.Gas)
		err = gasErr
		if err == nil && gas != uint64(receipt.GasUsed) {
			err = fmt.Errorf("JS gas %d, receipt gas %d", gas, receipt.GasUsed)
		}
		if js.Failed == nil || *js.Failed != (receipt.Status == 0) {
			err = errors.Join(err, errors.New("JS execution status differs from receipt"))
		}
		if frameErr != nil {
			err = errors.Join(err, frameErr)
		} else if js.Output == nil || frame.Output == nil || !bytes.Equal(*js.Output, *frame.Output) {
			err = errors.Join(err, errors.New("JS output differs from callTracer"))
		}
	}
	record(label+" debug_traceTransaction/JavaScript", err)

	var ots []otsEntry
	err = rpc.CallContext(ctx, &ots, "ots_traceTransaction", tx.Hash)
	if err == nil {
		if frameErr != nil {
			err = frameErr
		} else {
			err = compareOTS(ots, frame)
		}
	}
	record(label+" ots_traceTransaction", err)
	var output *hexutil.Bytes
	err = rpc.CallContext(ctx, &output, "ots_getTransactionError", tx.Hash)
	if err == nil {
		if receipt.Status != 0 {
			if output != nil {
				err = errors.New("successful transaction has OTS revert output")
			}
		} else if frameErr != nil {
			err = frameErr
		} else if strings.Contains(strings.ToLower(frame.Error), "revert") {
			if output == nil || frame.Output == nil || !bytes.Equal(*output, *frame.Output) {
				err = errors.New("OTS revert bytes differ from callTracer")
			}
		} else if output != nil {
			err = errors.New("halted (non-revert) transaction has OTS revert output")
		}
	}
	record(label+" ots_getTransactionError", err)
	return frameErr == nil && receipt.Status == 0 && strings.Contains(strings.ToLower(frame.Error), "revert")
}

func compareOTS(actual []otsEntry, root executionCall) error {
	var expected []otsEntry
	var visit func(executionCall, uint64)
	visit = func(frame executionCall, depth uint64) {
		entry := otsEntry{Type: frame.Type, Depth: &depth, From: frame.From, To: frame.To, Input: frame.Input, Value: frame.Value}
		if frame.Output != nil {
			entry.Output = *frame.Output
		}
		expected = append(expected, entry)
		for _, call := range frame.Calls {
			visit(call, depth+1)
		}
	}
	visit(root, 0)
	if len(actual) != len(expected) {
		return fmt.Errorf("OTS entries %d, callTracer frames %d", len(actual), len(expected))
	}
	for i, want := range expected {
		got := actual[i]
		if got.Depth == nil || *got.Depth != *want.Depth || !strings.EqualFold(got.Type, want.Type) || got.From != want.From || got.To != want.To || !bytes.Equal(got.Input, want.Input) || !bytes.Equal(got.Output, want.Output) || !(equalBigPtr(got.Value, want.Value) || (isZeroBig(got.Value) && isZeroBig(want.Value))) {
			return fmt.Errorf("OTS entry %d differs from callTracer frame", i)
		}
	}
	return nil
}

func checkCallMany(ctx context.Context, rpc Caller, oracle *replayBalances, index int, addresses []common.Address, record replayRecorder) {
	call, overrides := balanceProbe(addresses)
	// A counter proves that calls share state and that overrides apply only once,
	// rather than independently replaying the same synthetic call three times.
	for _, raw := range overrides {
		override := raw.(map[string]any)
		code := override["code"].(hexutil.Bytes)
		code = append(code[:len(code)-4], 0x5f, 0x54, 0x60, 0x01, 0x01, 0x80, 0x5f, 0x55, 0x60, byte(len(addresses)*32), 0x52)
		code = append(code, 0x60, byte((len(addresses)+1)*32), 0x5f, 0xf3)
		override["code"] = code
		override["state"] = map[common.Hash]common.Hash{} // isolated scratch storage
	}
	bundles := []any{map[string]any{"transactions": []any{call, call}}, map[string]any{"transactions": []any{call}}}
	state := map[string]any{"blockNumber": oracle.hash, "transactionIndex": index + 1}
	for _, method := range []string{"eth_callMany", "debug_traceCallMany"} {
		var results [][]json.RawMessage
		options := any(overrides)
		if method == "debug_traceCallMany" {
			options = map[string]any{"tracer": "callTracer", "stateOverrides": overrides}
		}
		err := rpc.CallContext(ctx, &results, method, bundles, state, options)
		if err == nil {
			err = checkManyResults(ctx, oracle, index, addresses, method, results)
		}
		record(fmt.Sprintf("tx[%d] %s settled prefix + shared overrides", index, method), err)
	}
}

func checkManyResults(ctx context.Context, oracle *replayBalances, index int, addresses []common.Address, method string, results [][]json.RawMessage) error {
	if len(results) != 2 || len(results[0]) != 2 || len(results[1]) != 1 {
		return errors.New("call-many bundle/result count mismatch")
	}
	counter := uint64(0)
	var failures []error
	for _, bundle := range results {
		for _, raw := range bundle {
			counter++
			var entry struct {
				Output *hexutil.Bytes  `json:"output"`
				Value  json.RawMessage `json:"value"`
				Error  string          `json:"error"`
			}
			if err := json.Unmarshal(raw, &entry); err != nil {
				return err
			}
			if entry.Error != "" {
				failures = append(failures, fmt.Errorf("call %d: %s", counter, entry.Error))
				continue
			}
			output := entry.Output
			if method == "eth_callMany" {
				if err := json.Unmarshal(entry.Value, &output); err != nil {
					return err
				}
			}
			if output == nil || len(*output) != (len(addresses)+1)*32 {
				return errors.New("call-many missing/truncated output")
			}
			balanceWords := (*output)[:len(addresses)*32]
			failures = append(failures, checkBalanceOutput(ctx, oracle, index, addresses, (*hexutil.Bytes)(&balanceWords)))
			count := new(big.Int).SetBytes((*output)[len(addresses)*32:])
			if count.Cmp(new(big.Int).SetUint64(counter)) != 0 {
				failures = append(failures, fmt.Errorf("call %d counter %s: overrides/state were not carried across calls/bundles", counter, count))
			}
		}
	}
	return errors.Join(failures...)
}

func checkBoundaryCalls(ctx context.Context, rpc Caller, header *replayHeader, tx rpcTransactionDetails, record replayRecorder) {
	addresses := replayAddresses(header, tx)
	oracle, err := boundaryBalances(ctx, rpc, header, addresses)
	if err != nil {
		record("trace_call boundary oracle", err)
		return
	}
	_, overrides := balanceProbe(addresses)
	var code hexutil.Bytes
	for _, override := range overrides {
		code = override.(map[string]any)["code"].(hexutil.Bytes)
	}
	// Creation calls return the probe's bytes without needing trace_callMany state
	// overrides (that endpoint has none). No signed transaction is submitted.
	call := map[string]any{"input": code, "gas": "0x100000", "gasPrice": "0x0"}
	var single *parityReplay
	err = rpc.CallContext(ctx, &single, "trace_call", call, []string{"trace"}, header.Hash)
	if err == nil {
		if single == nil {
			err = errors.New("missing trace_call result")
		} else {
			err = errors.Join(checkParityFrames(single.Trace), checkBalanceOutput(ctx, oracle, 0, addresses, single.Output))
		}
	}
	record("trace_call block-boundary balances", err)
	var many []*parityReplay
	err = rpc.CallContext(ctx, &many, "trace_callMany", []any{[]any{call, []string{"trace"}}, []any{call, []string{"trace"}}}, header.Hash)
	if err == nil {
		if len(many) != 2 {
			err = errors.New("trace_callMany result count mismatch")
		} else {
			for _, item := range many {
				if item == nil {
					err = errors.Join(err, errors.New("missing trace_callMany result"))
					continue
				}
				err = errors.Join(err, checkParityFrames(item.Trace), checkBalanceOutput(ctx, oracle, 0, addresses, item.Output))
			}
		}
	}
	record("trace_callMany block-boundary balances", err)
}
