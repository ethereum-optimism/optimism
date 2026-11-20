package sdm

import (
	"context"
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestExtendedReplayCompleteAndFixtureGaps(t *testing.T) {
	f := replayFixture(t)
	result, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
	require.NoError(t, err)
	require.False(t, result.Complete)
	gaps := 0
	for _, check := range result.Checks {
		if check.Skipped != "" {
			gaps++
		}
	}
	require.Equal(t, 2, gaps, "reverted tx and cached bad block must not be silent passes")
	f.extended.bad = true
	f.receipts[common.HexToHash("0x3")].Status = 0
	result, err = ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
	require.NoError(t, err)
	require.True(t, result.Complete)
	for _, method := range []string{"trace_replayTransaction", "trace_replayBlockTransactions", "trace_transaction", "trace_get", "trace_block", "trace_filter", "trace_call", "trace_callMany", "eth_callMany", "debug_traceCallMany", "debug_intermediateRoots", "debug_executionWitness", "ots_traceTransaction", "ots_getTransactionError", "debug_traceBadBlock", "debug_traceBlock"} {
		require.Positive(t, f.calls[method], method)
	}
}

func TestExtendedReplayRegressions(t *testing.T) {
	for _, test := range []struct {
		name, want string
		mutate     func(string, []any, any) any
	}{
		{"parity unsettled balance", "post-balance", func(m string, a []any, v any) any {
			if m == "trace_replayTransaction" && a[0] == common.HexToHash("0x2") {
				v.(map[string]any)["stateDiff"].(map[string]any)[common.HexToAddress("0x1234").Hex()].(map[string]any)["balance"].(map[string]any)["*"].(map[string]any)["to"] = "0x1"
			}
			return v
		}},
		{"block parity unsettled balance", "post-balance", func(m string, _ []any, v any) any {
			if m == "trace_replayBlockTransactions" {
				v.([]any)[3].(map[string]any)["stateDiff"].(map[string]any)[common.HexToAddress("0x1234").Hex()].(map[string]any)["balance"].(map[string]any)["*"].(map[string]any)["to"] = "0x1"
			}
			return v
		}},
		{"parity unknown delta", "unknown parity balance", func(m string, a []any, v any) any {
			if m == "trace_replayTransaction" && a[0] == common.HexToHash("0x2") {
				v.(map[string]any)["stateDiff"].(map[string]any)[common.HexToAddress("0x1234").Hex()].(map[string]any)["balance"] = map[string]any{"?": "0x1"}
			}
			return v
		}},
		{"localized missing hash", "identity mismatch", func(m string, _ []any, v any) any {
			if m == "trace_transaction" {
				delete(v.([]any)[0].(map[string]any), "blockHash")
			}
			return v
		}},
		{"trace get wrong tree", "frame tree differs", func(m string, _ []any, v any) any {
			if m == "trace_get" {
				v.(map[string]any)["result"].(map[string]any)["gasUsed"] = "0xff"
			}
			return v
		}},
		{"trace block reordered", "identity mismatch", func(m string, _ []any, v any) any {
			if m == "trace_block" {
				x := v.([]any)
				x[1], x[2] = x[2], x[1]
			}
			return v
		}},
		{"trace filter pagination", "pagination differs", func(m string, a []any, v any) any {
			if m == "trace_filter" {
				if _, ok := a[0].(map[string]any)["after"]; ok {
					return []any{}
				}
			}
			return v
		}},
		{"call-many resets state", "counter", func(m string, _ []any, v any) any {
			if m == "eth_callMany" {
				rows := v.([]any)[0].([]any)
				rows[1] = rows[0]
			}
			return v
		}},
		{"trace-call-many truncates bundle", "count mismatch", func(m string, _ []any, v any) any {
			if m == "debug_traceCallMany" {
				return v.([]any)[:1]
			}
			return v
		}},
		{"JS raw gas", "JS gas 100", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[1].(map[string]any)["tracer"] == replayJSTracer {
				v.(map[string]any)["gas"] = 100
			}
			return v
		}},
		{"OTS wrong depth", "OTS entry", func(m string, _ []any, v any) any {
			if m == "ots_traceTransaction" {
				v.([]any)[0].(map[string]any)["depth"] = 9
			}
			return v
		}},
		{"OTS spurious revert", "successful transaction", func(m string, _ []any, v any) any {
			if m == "ots_getTransactionError" {
				return "0x01"
			}
			return v
		}},
		{"middle root wrong last correct", "root after tx[1]", func(m string, _ []any, v any) any {
			if m == "debug_intermediateRoots" {
				v.([]any)[1] = common.HexToHash("0x1234").Hex()
			}
			return v
		}},
		{"missing witness", "witness-backed parent trie", func(m string, _ []any, v any) any {
			if m == "debug_executionWitness" {
				v.(map[string]any)["state"] = []any{}
			}
			return v
		}},
		{"missing witness keys", "no account key", func(m string, _ []any, v any) any {
			if m == "debug_executionWitness" {
				v.(map[string]any)["keys"] = []any{}
			}
			return v
		}},
		{"incomplete canonical account", "incomplete canonical account", func(m string, _ []any, v any) any {
			if m == "debug_accountAt" && v != nil {
				delete(v.(map[string]any), "codeHash")
			}
			return v
		}},
		{"missing raw diagnostic frame", "diagnostic trace count", func(m string, _ []any, v any) any {
			if m == "debug_traceBlock" {
				return []any{}
			}
			return v
		}},
		{"invalid bad-block fixture", "header hash mismatch", func(m string, _ []any, v any) any {
			if m == "debug_getBadBlocks" {
				return []any{map[string]any{"hash": common.Hash{}, "rlp": "0x80"}}
			}
			return v
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := replayFixture(t)
			f.mutate = test.mutate
			r, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
			require.ErrorContains(t, err, test.want)
			require.NotNil(t, r)
			require.False(t, r.Complete)
		})
	}
}

func TestExtendedMethodsUnavailableFail(t *testing.T) {
	for _, method := range []string{"trace_replayTransaction", "trace_replayBlockTransactions", "trace_transaction", "trace_get", "trace_block", "trace_filter", "trace_call", "trace_callMany", "eth_callMany", "debug_traceCallMany", "ots_traceTransaction", "ots_getTransactionError", "debug_intermediateRoots", "debug_executionWitness", "debug_traceBlock", "debug_getBadBlocks"} {
		t.Run(method, func(t *testing.T) {
			f := replayFixture(t)
			f.unavailable = method
			r, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
			require.ErrorContains(t, err, "method not found")
			require.False(t, r.Complete)
		})
	}
}

func TestTraceChainResultAlignment(t *testing.T) {
	f := replayFixture(t)
	receipts := make([]*RPCReceipt, 0, len(f.header.Transactions))
	event := chainReplay{Block: uint64Ptr(42), Hash: f.header.Hash}
	for i, tx := range f.header.Transactions {
		receipts = append(receipts, f.receipts[tx.Hash])
		raw, err := json.Marshal(f.fixtureCall(i))
		require.NoError(t, err)
		event.Traces = append(event.Traces, &replayBlockFrame{TxHash: ptr.New(tx.Hash), Result: raw})
	}
	require.NoError(t, checkChainReplay(event, 42, f.header.Hash, receipts))
	event.Traces[1] = &replayBlockFrame{Error: "executor failed", TxHash: ptr.New(receipts[1].TxHash)}
	event.Traces[2] = nil
	err := checkChainReplay(event, 42, f.header.Hash, receipts)
	require.ErrorContains(t, err, "tx[1]: executor failed")
	require.ErrorContains(t, err, "tx[2] missing result")
	event.Hash = common.Hash{}
	require.ErrorContains(t, checkChainReplay(event, 42, f.header.Hash, receipts), "identity mismatch")
}

func TestRawDiagnosticDoesNotRequireCanonicalGas(t *testing.T) {
	f := replayFixture(t)
	f.mutate = func(m string, _ []any, v any) any {
		if m == "debug_traceBlock" {
			for _, row := range v.([]any) {
				row.(map[string]any)["result"].(map[string]any)["gasUsed"] = "0xffff"
			}
		}
		return v
	}
	var failures []error
	checkRawDiagnostics(context.Background(), f, &f.header, func(_ string, err error) {
		if err != nil {
			if _, gap := err.(replayCoverageGap); !gap {
				failures = append(failures, err)
			}
		}
	})
	require.Empty(t, failures, "raw diagnostic gas is intentionally not canonical receipt gas")
}

func TestOTSOptionalZeroValue(t *testing.T) {
	frame := executionCall{Type: "STATICCALL", From: common.HexToAddress("0x1"), To: common.HexToAddress("0x2")}
	entry := otsEntry{Type: "STATICCALL", Depth: ptr.New(uint64(0)), From: frame.From, To: frame.To, Value: bigPtr(0)}
	require.NoError(t, compareOTS([]otsEntry{entry}, frame))
	entry.Value = bigPtr(1)
	require.ErrorContains(t, compareOTS([]otsEntry{entry}, frame), "differs")
}

func TestReplayJSONPrecisionAndOrder(t *testing.T) {
	require.True(t, sameReplayJSON(json.RawMessage(`{"b":2,"a":1}`), json.RawMessage(`{"a":1,"b":2}`)))
	require.False(t, sameReplayJSON(json.RawMessage(`9007199254740992`), json.RawMessage(`9007199254740993`)))
}

func TestParityFrameGasIsNotReceiptGas(t *testing.T) {
	f := replayFixture(t)
	r := f.fixtureParity(1)
	require.Equal(t, hexutil.Uint64(93), f.receipts[common.HexToHash("0x2")].GasUsed)
	require.Contains(t, string(r.Trace[0].Result), "0x5")
	oracle := replayBalances{rpc: f, hash: f.header.Hash, parent: f.header.ParentHash, cache: make(map[balanceAt]*big.Int)}
	require.NoError(t, checkParityReplay(context.Background(), &oracle, 1, &f.header, f.receipts[common.HexToHash("0x2")], r))
}
