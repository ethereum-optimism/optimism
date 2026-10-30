package sdm

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

var fullReplayOptions = RPCReplayOptions{RequireJS: true, CheckOTS: true}

type rpcReplayFixture struct {
	block          *RPCBlock
	header         replayHeader
	receipts       map[common.Hash]*RPCReceipt
	states         []map[common.Address]int64 // parent, then after each block-global tx
	mutate         func(string, []any, any) any
	unavailable    string
	calls          map[string]int
	accountIndices []int
	extended       *extendedReplayFixture
}

func replayFixture(t *testing.T) *rpcReplayFixture {
	t.Helper()
	sender := common.HexToAddress("0x1234")
	payload, err := rlp.EncodeToBytes(optypes.PostExecPayload{Version: 1, BlockNumber: 42,
		GasRefundEntries: []optypes.SDMGasEntry{{Index: 1, GasRefund: 7}, {Index: 3, GasRefund: 11}}})
	require.NoError(t, err)
	f := &rpcReplayFixture{receipts: make(map[common.Hash]*RPCReceipt), calls: make(map[string]int)}
	f.block = &RPCBlock{Number: 42, Hash: common.HexToHash("0xbeef"), GasUsed: 342}
	f.header = replayHeader{Hash: f.block.Hash, ParentHash: common.HexToHash("0xcafe"), Miner: predeploys.SequencerFeeVaultAddr}
	// Deposit, refunded user, unrefunded user, later refunded user, PostExec.
	// Reusing the sender makes both prefix state and transaction boundaries matter.
	gas := []uint64{100, 93, 60, 89, 0}
	refunds := []uint64{0, 7, 0, 11, 0}
	kinds := []uint64{optypes.DepositTxType, 2, 2, 2, optypes.PostExecTxType}
	f.states = append(f.states, map[common.Address]int64{sender: 100000})
	for i, kind := range kinds {
		tx := RPCTransaction{Hash: common.BigToHash(big.NewInt(int64(i + 1))), Type: hexutil.Uint64(kind)}
		if kind == optypes.PostExecTxType {
			tx.Input = payload
		}
		f.block.Transactions = append(f.block.Transactions, tx)
		f.header.Transactions = append(f.header.Transactions, rpcTransactionDetails{RPCTransaction: tx, From: sender})
		receipt := &RPCReceipt{TxHash: tx.Hash, BlockHash: f.block.Hash, TransactionIndex: hexutil.Uint64(i), Type: hexutil.Uint64(kind), GasUsed: hexutil.Uint64(gas[i]), Status: 1}
		if refunds[i] != 0 {
			receipt.OPGasRefund = uint64Ptr(refunds[i])
		}
		f.receipts[tx.Hash] = receipt
		state := make(map[common.Address]int64)
		for addr, balance := range f.states[i] {
			state[addr] = balance
		}
		if kind == 2 {
			state[sender] -= int64(gas[i])*2 + 5
			state[predeploys.OperatorFeeVaultAddr] += 5
			state[predeploys.BaseFeeVaultAddr] += int64(gas[i])
			state[predeploys.SequencerFeeVaultAddr] += int64(gas[i])
		}
		f.states = append(f.states, state)
	}
	initExtendedFixture(t, f)
	return f
}

func (f *rpcReplayFixture) CallContext(ctx context.Context, result any, method string, args ...any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.calls[method]++
	if method == f.unavailable {
		return errors.New("method not found")
	}
	var value any
	switch method {
	case "eth_getBlockByNumber":
		value = f.block
	case "eth_getBlockByHash":
		value = f.header
		if args[0] == f.header.ParentHash {
			value = map[string]any{"hash": f.header.ParentHash, "stateRoot": f.extended.parentRoot}
		}
	case "eth_getTransactionReceipt":
		value = f.receipts[args[0].(common.Hash)]
	case "eth_getBalance":
		state := 0
		if args[1].(map[string]any)["blockHash"] == f.header.Hash {
			state = len(f.states) - 1
		}
		value = bigPtr(f.states[state][args[0].(common.Address)])
	case "debug_accountAt":
		if args[0] != f.header.Hash {
			return errors.New("wrong accountAt block hash")
		}
		index := args[1].(int)
		f.accountIndices = append(f.accountIndices, index)
		value = fixtureAccount(f.states[index+1][args[2].(common.Address)])
	case "debug_traceTransaction":
		receipt := f.receipts[args[0].(common.Hash)]
		index := int(receipt.TransactionIndex)
		opts := args[1].(map[string]any)
		switch opts["tracer"] {
		case "callTracer":
			value = f.fixtureCall(index)
		case "prestateTracer":
			diff := replayDiff{Pre: make(map[common.Address]replayAccount), Post: make(map[common.Address]replayAccount)}
			for addr, after := range f.states[index+1] {
				before := f.states[index][addr]
				if before != after {
					diff.Pre[addr] = replayAccount{Balance: bigPtr(before)}
					diff.Post[addr] = replayAccount{Balance: bigPtr(after)}
				}
			}
			value = diff
		default:
			value = map[string]any{"gas": uint64(receipt.GasUsed), "failed": receipt.Status == 0, "output": f.fixtureCall(index).Output}
		}
	case "debug_traceBlockByHash", "debug_traceBlockByNumber":
		frames := make([]map[string]any, 0, len(f.block.Transactions))
		for i, tx := range f.block.Transactions {
			frames = append(frames, map[string]any{"txHash": tx.Hash, "result": f.fixtureCall(i)})
		}
		value = frames
	case "debug_traceCall":
		if args[1] != f.header.Hash {
			return errors.New("prefix probe not hash pinned")
		}
		opts := args[2].(map[string]any)
		prefix := int(opts["txIndex"].(hexutil.Uint64))
		overrides := opts["stateOverrides"].(map[common.Address]any)
		var code hexutil.Bytes
		for _, override := range overrides {
			code = override.(map[string]any)["code"].(hexutil.Bytes)
		}
		count := (len(code) - 4) / 25
		output := make(hexutil.Bytes, 32*count)
		for i := 0; i < count; i++ {
			address := common.BytesToAddress(code[i*25+1 : i*25+21])
			big.NewInt(f.states[prefix][address]).FillBytes(output[i*32 : (i+1)*32])
		}
		value = replayCallFrame{Output: &output}
	default:
		var err error
		value, err = f.extendedValue(method, args)
		if err != nil {
			return err
		}
	}
	// Mutations operate on actual wire-shaped JSON, also testing missing fields.
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if f.mutate != nil {
		var wire any
		if err := json.Unmarshal(encoded, &wire); err != nil {
			return err
		}
		encoded, err = json.Marshal(f.mutate(method, args, wire))
		if err != nil {
			return err
		}
	}
	return json.Unmarshal(encoded, result)
}

func TestValidateRPCReplay(t *testing.T) {
	f := replayFixture(t)
	result, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
	require.NoError(t, err)
	require.Equal(t, 2, result.RefundedTransactions)
	require.NotEmpty(t, result.Checks)
	for _, check := range result.Checks {
		require.Empty(t, check.Error, check.Name)
	}
	for _, method := range []string{"debug_traceTransaction", "debug_traceBlockByHash", "debug_traceBlockByNumber", "debug_traceCall", "debug_accountAt"} {
		require.Positive(t, f.calls[method], method)
	}
	require.Zero(t, f.calls["debug_replaySDMBlock"], "policy-free replay is not a canonical state oracle")
	require.Contains(t, f.accountIndices, 0, "first user's prestate is after the deposit")
	require.Contains(t, f.accountIndices, 2, "later user's prestate must include earlier settlement")
	require.Contains(t, f.accountIndices, 3)
}

func TestValidateRPCReplayRegressions(t *testing.T) {
	for _, test := range []struct {
		name, want string
		mutate     func(string, []any, any) any
	}{
		{"unsettled call gas", "gasUsed 100, canonical receipt gasUsed 93", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[0] == common.HexToHash("0x2") && a[1].(map[string]any)["tracer"] == "callTracer" {
				v.(map[string]any)["gasUsed"] = "0x64"
			}
			return v
		}},
		{"unsettled default gas", "gas 100, canonical receipt gasUsed 93", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[0] == common.HexToHash("0x2") && a[1].(map[string]any)["tracer"] == nil {
				v.(map[string]any)["gas"] = 100
			}
			return v
		}},
		{"later prefix prestate", "pre-balance", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[0] == common.HexToHash("0x4") && a[1].(map[string]any)["tracer"] == "prestateTracer" {
				pre := v.(map[string]any)["pre"].(map[string]any)
				pre[common.HexToAddress("0x1234").Hex()].(map[string]any)["balance"] = "0x1"
			}
			return v
		}},
		{"unsettled sender diff", "post-balance", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[1].(map[string]any)["tracer"] == "prestateTracer" {
				post := v.(map[string]any)["post"].(map[string]any)
				post[common.HexToAddress("0x1234").Hex()].(map[string]any)["balance"] = "0x1"
			}
			return v
		}},
		{"omitted fee vault diff", "missing changed", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[1].(map[string]any)["tracer"] == "prestateTracer" {
				delete(v.(map[string]any)["pre"].(map[string]any), predeploys.BaseFeeVaultAddr.Hex())
				delete(v.(map[string]any)["post"].(map[string]any), predeploys.BaseFeeVaultAddr.Hex())
			}
			return v
		}},
		{"unsettled synthetic call prefix", "prefix balance", func(m string, _ []any, v any) any {
			if m == "debug_traceCall" {
				v.(map[string]any)["output"] = "0x" + strings.Repeat("00", 160)
			}
			return v
		}},
		{"missing trace gas", "omits gasUsed", func(m string, a []any, v any) any {
			if m == "debug_traceTransaction" && a[1].(map[string]any)["tracer"] == "callTracer" {
				delete(v.(map[string]any), "gasUsed")
			}
			return v
		}},
		{"block trace failure", "execution failed", func(m string, _ []any, v any) any {
			if m == "debug_traceBlockByHash" {
				v.([]any)[3].(map[string]any)["error"] = "execution failed"
			}
			return v
		}},
		{"block cumulative not per transaction", "canonical receipt gasUsed 89", func(m string, _ []any, v any) any {
			if m == "debug_traceBlockByNumber" {
				v.([]any)[3].(map[string]any)["result"].(map[string]any)["gasUsed"] = "0x156"
			}
			return v
		}},
		{"missing block trace", "got 4 traces, want 5", func(m string, _ []any, v any) any {
			if m == "debug_traceBlockByHash" {
				return v.([]any)[:4]
			}
			return v
		}},
		{"wrong block trace identity", "transaction hash mismatch", func(m string, _ []any, v any) any {
			if m == "debug_traceBlockByHash" {
				v.([]any)[1].(map[string]any)["txHash"] = common.HexToHash("0xff").Hex()
			}
			return v
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := replayFixture(t)
			f.mutate = test.mutate
			result, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
			require.ErrorContains(t, err, test.want)
			require.NotNil(t, result, "partial evidence must survive failures")
			require.Positive(t, f.calls["debug_traceBlockByNumber"], "failure must not short circuit other endpoint families")
		})
	}
}

func TestRPCReplayUnavailableIsFailure(t *testing.T) {
	for _, method := range []string{"debug_traceTransaction", "debug_traceBlockByNumber", "debug_traceBlockByHash", "debug_traceCall", "debug_accountAt"} {
		t.Run(method, func(t *testing.T) {
			f := replayFixture(t)
			f.unavailable = method
			result, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
			require.ErrorContains(t, err, "method not found")
			require.NotNil(t, result)
		})
	}
}

func TestRPCReplayReorgAndReceiptIdentity(t *testing.T) {
	t.Run("reorg", func(t *testing.T) {
		f := replayFixture(t)
		f.mutate = func(m string, _ []any, v any) any {
			if m == "eth_getBlockByNumber" && f.calls[m] > 1 {
				v.(map[string]any)["hash"] = common.HexToHash("0xff").Hex()
			}
			return v
		}
		_, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
		require.ErrorContains(t, err, "reorged")
	})
	t.Run("receipt identity", func(t *testing.T) {
		f := replayFixture(t)
		f.receipts[common.HexToHash("0x2")].BlockHash = common.Hash{}
		_, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
		require.ErrorContains(t, err, "receipt identity")
	})
	t.Run("no refund payload", func(t *testing.T) {
		f := replayFixture(t)
		f.block.Transactions = f.block.Transactions[:4]
		_, err := ValidateRPCReplay(context.Background(), f, 42, fullReplayOptions)
		require.ErrorContains(t, err, "does not contain post-exec")
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := ValidateRPCReplay(ctx, replayFixture(t), 42, fullReplayOptions)
		require.ErrorIs(t, err, context.Canceled)
	})
}

func TestReplayDiffAccountLifecycle(t *testing.T) {
	address := common.HexToAddress("0x1234")
	for _, test := range []struct {
		name          string
		before, after int64
		pre, post     *hexutil.Big
		want          string
	}{
		{"unchanged omitted", 10, 10, nil, nil, ""},
		{"created", 0, 10, nil, bigPtr(10), ""},
		{"deleted", 10, 0, bigPtr(10), nil, ""},
		{"changed explicit", 10, 9, bigPtr(10), bigPtr(9), ""},
		{"missing change", 10, 9, nil, nil, "missing changed"},
		{"false creation", 10, 10, nil, bigPtr(10), "implies creation"},
		{"false deletion", 10, 10, bigPtr(10), nil, "implies deletion"},
	} {
		t.Run(test.name, func(t *testing.T) {
			oracle := replayBalances{cache: map[balanceAt]*big.Int{
				{0, address}: big.NewInt(test.before), {1, address}: big.NewInt(test.after),
			}}
			diff := replayDiff{Pre: make(map[common.Address]replayAccount), Post: make(map[common.Address]replayAccount)}
			if test.pre != nil {
				diff.Pre[address] = replayAccount{Balance: test.pre}
			}
			if test.post != nil {
				diff.Post[address] = replayAccount{Balance: test.post}
			}
			err := checkReplayDiff(context.Background(), &oracle, 1, []common.Address{address}, diff)
			if test.want != "" {
				require.ErrorContains(t, err, test.want)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestTraceGasEncoding(t *testing.T) {
	for _, raw := range []string{"24981", `"0x6195"`} {
		gas, err := traceGas(json.RawMessage(raw))
		require.NoError(t, err)
		require.Equal(t, uint64(24981), gas)
	}
	for _, raw := range []string{"", "null", "-1", "1.2", `"nope"`, "18446744073709551616"} {
		_, err := traceGas(json.RawMessage(raw))
		require.Error(t, err, raw)
	}
}

func TestReplayBalanceParentAndAccountLifecycle(t *testing.T) {
	f := replayFixture(t)
	oracle := replayBalances{rpc: f, hash: f.header.Hash, parent: f.header.ParentHash, cache: make(map[balanceAt]*big.Int)}
	sender := common.HexToAddress("0x1234")
	value, err := oracle.at(context.Background(), -1, sender)
	require.NoError(t, err)
	require.Equal(t, int64(100000), value.Int64())
	require.Equal(t, 1, f.calls["eth_getBalance"])
	_, err = oracle.at(context.Background(), -1, sender)
	require.NoError(t, err)
	require.Equal(t, 1, f.calls["eth_getBalance"])
	// Explicit zero is valid; omitted balance on an existing account is not.
	f.mutate = func(m string, _ []any, v any) any {
		if m == "debug_accountAt" {
			return map[string]any{}
		}
		return v
	}
	_, err = oracle.at(context.Background(), 1, sender)
	require.ErrorContains(t, err, "omits balance")
	f.mutate = func(m string, _ []any, v any) any {
		if m == "debug_accountAt" {
			return nil
		}
		return v
	}
	value, err = oracle.at(context.Background(), 1, sender)
	require.NoError(t, err)
	require.Zero(t, value.Sign())
}
