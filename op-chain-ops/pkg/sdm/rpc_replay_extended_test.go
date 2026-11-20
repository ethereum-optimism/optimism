package sdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/ptr"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/stretchr/testify/require"
)

type extendedReplayFixture struct {
	parentRoot common.Hash
	roots      []common.Hash
	witness    replayWitness
	raw        hexutil.Bytes
	client     *gethrpc.Client
	bad        bool
}

func fixtureAccount(balance int64) *canonicalReplayAccount {
	if balance == 0 {
		return nil
	}
	return &canonicalReplayAccount{Balance: bigPtr(balance), Nonce: uint64Ptr(0), StorageRoot: ptr.New(types.EmptyRootHash), CodeHash: ptr.New(types.EmptyCodeHash)}
}
func initExtendedFixture(t *testing.T, f *rpcReplayFixture) {
	t.Helper()
	e := &extendedReplayFixture{}
	f.extended = e
	disk := rawdb.NewMemoryDatabase()
	db := triedb.NewDatabase(disk, triedb.HashDefaults)
	defer disk.Close()
	defer db.Close()
	tree := trie.NewEmpty(db)
	for address, balance := range f.states[0] {
		encoded, err := encodeCanonicalAccount(fixtureAccount(balance))
		require.NoError(t, err)
		require.NoError(t, tree.Update(crypto.Keccak256(address[:]), encoded))
	}
	e.parentRoot = tree.Hash()
	proof := memorydb.New()
	defer proof.Close()
	addresses := map[common.Address]struct{}{}
	for _, state := range f.states {
		for address := range state {
			addresses[address] = struct{}{}
		}
	}
	for address := range addresses {
		e.witness.Keys = append(e.witness.Keys, append(hexutil.Bytes{}, address[:]...))
		require.NoError(t, tree.Prove(crypto.Keccak256(address[:]), proof))
	}
	it := proof.NewIterator(nil, nil)
	for it.Next() {
		e.witness.State = append(e.witness.State, append(hexutil.Bytes{}, it.Value()...))
	}
	it.Release()
	for _, state := range f.states[1:] {
		for address := range addresses {
			encoded, err := encodeCanonicalAccount(fixtureAccount(state[address]))
			require.NoError(t, err)
			if encoded == nil {
				require.NoError(t, tree.Delete(crypto.Keccak256(address[:])))
			} else {
				require.NoError(t, tree.Update(crypto.Keccak256(address[:]), encoded))
			}
		}
		e.roots = append(e.roots, tree.Hash())
	}
	header := &types.Header{ParentHash: f.header.ParentHash, Number: big.NewInt(42), GasUsed: uint64(f.block.GasUsed), GasLimit: 1000000}
	encoded, err := rlp.EncodeToBytes(header)
	require.NoError(t, err)
	txs, err := rlp.EncodeToBytes(make([][]byte, len(f.block.Transactions)))
	require.NoError(t, err)
	e.raw, err = rlp.EncodeToBytes([]rlp.RawValue{encoded, txs, {0xc0}})
	require.NoError(t, err)
	f.block.Hash = crypto.Keccak256Hash(encoded)
	f.header.Hash = f.block.Hash
	for _, r := range f.receipts {
		r.BlockHash = f.block.Hash
	}
	server := gethrpc.NewServer()
	require.NoError(t, server.RegisterName("debug", &fixtureDebugSubscription{f}))
	e.client = gethrpc.DialInProc(server)
	t.Cleanup(func() { e.client.Close(); server.Stop() })
}

func (f *rpcReplayFixture) fixtureCall(i int) executionCall {
	tx := f.header.Transactions[i]
	receipt := f.receipts[tx.Hash]
	output := hexutil.Bytes{}
	frame := executionCall{replayCallFrame: replayCallFrame{GasUsed: uint64Ptr(uint64(receipt.GasUsed)), Output: &output}, From: tx.From, Type: "CREATE", Input: tx.Input, Value: bigPtr(0)}
	if tx.To != nil {
		frame.To = *tx.To
		frame.Type = "CALL"
	}
	if receipt.Status == 0 {
		frame.Error = "execution reverted"
		output = hexutil.Bytes{0xde, 0xad}
	}
	return frame
}
func (f *rpcReplayFixture) fixtureParity(i int) *parityReplay {
	tx := f.header.Transactions[i]
	frame := f.fixtureCall(i)
	action := map[string]any{"from": tx.From, "init": tx.Input}
	kind := "create"
	if tx.To != nil {
		kind = "call"
		action = map[string]any{"from": tx.From, "to": tx.To, "input": tx.Input}
	}
	a, _ := json.Marshal(action)
	r, _ := json.Marshal(map[string]any{"gasUsed": "0x5", "output": frame.Output})
	replay := &parityReplay{Output: frame.Output, Trace: []parityFrame{{Type: kind, Action: a, Result: r, Error: frame.Error, Subtraces: ptr.New(uint64(0)), TraceAddress: []uint64{}}}, TransactionHash: ptr.New(tx.Hash)}
	// Build wire-format balance deltas independently from prestate-tracer encoding.
	balances := make(map[common.Address]map[string]any)
	for address, after := range f.states[i+1] {
		before := f.states[i][address]
		if before != after {
			balances[address] = map[string]any{"balance": map[string]any{"*": map[string]any{"from": hexutil.EncodeBig(big.NewInt(before)), "to": hexutil.EncodeBig(big.NewInt(after))}}}
		}
	}
	encoded, _ := json.Marshal(balances)
	_ = json.Unmarshal(encoded, &replay.StateDiff)
	return replay
}
func (f *rpcReplayFixture) localized(i int) localizedParity {
	return localizedParity{parityFrame: f.fixtureParity(i).Trace[0], BlockHash: ptr.New(f.header.Hash), BlockNumber: ptr.New(uint64(42)), TransactionHash: ptr.New(f.header.Transactions[i].Hash), TransactionPosition: ptr.New(uint64(i))}
}
func fixtureProbe(code hexutil.Bytes, state map[common.Address]int64, count int) hexutil.Bytes {
	out := make(hexutil.Bytes, count*32)
	for i := 0; i < count; i++ {
		address := common.BytesToAddress(code[i*25+1 : i*25+21])
		big.NewInt(state[address]).FillBytes(out[i*32 : (i+1)*32])
	}
	return out
}
func (f *rpcReplayFixture) extendedValue(method string, args []any) (any, error) {
	index := func() int { return int(f.receipts[args[0].(common.Hash)].TransactionIndex) }
	switch method {
	case "trace_replayTransaction":
		return f.fixtureParity(index()), nil
	case "trace_transaction":
		return []localizedParity{f.localized(index())}, nil
	case "trace_get":
		return f.localized(index()), nil
	case "trace_replayBlockTransactions":
		var values []*parityReplay
		for i := range f.header.Transactions {
			values = append(values, f.fixtureParity(i))
		}
		return values, nil
	case "trace_block", "trace_filter":
		var values []localizedParity
		for i := range f.header.Transactions {
			values = append(values, f.localized(i))
		}
		if method == "trace_filter" {
			if _, ok := args[0].(map[string]any)["after"]; ok {
				return values[1:2], nil
			}
		}
		return values, nil
	case "ots_traceTransaction":
		call := f.fixtureCall(index())
		return []otsEntry{{Type: call.Type, Depth: ptr.New(uint64(0)), From: call.From, To: call.To, Input: call.Input, Output: *call.Output, Value: call.Value}}, nil
	case "ots_getTransactionError":
		if f.receipts[args[0].(common.Hash)].Status == 0 {
			return f.fixtureCall(index()).Output, nil
		}
		return nil, nil
	case "eth_callMany", "debug_traceCallMany":
		state := args[1].(map[string]any)
		prefix := state["transactionIndex"].(int)
		var overrides map[common.Address]any
		if method == "eth_callMany" {
			overrides = args[2].(map[common.Address]any)
		} else {
			overrides = args[2].(map[string]any)["stateOverrides"].(map[common.Address]any)
		}
		var code hexutil.Bytes
		for _, override := range overrides {
			code = override.(map[string]any)["code"].(hexutil.Bytes)
		}
		count := (len(code) - 15) / 25
		value := fixtureProbe(code, f.states[prefix], count)
		var bundles [][]any
		counter := int64(0)
		for _, size := range []int{2, 1} {
			var bundle []any
			for range size {
				counter++
				output := append(hexutil.Bytes{}, value...)
				output = append(output, big.NewInt(counter).FillBytes(make([]byte, 32))...)
				if method == "eth_callMany" {
					bundle = append(bundle, map[string]any{"value": output})
				} else {
					bundle = append(bundle, map[string]any{"output": output, "value": "0x0"})
				}
			}
			bundles = append(bundles, bundle)
		}
		return bundles, nil
	case "trace_call", "trace_callMany":
		var call map[string]any
		if method == "trace_call" {
			call = args[0].(map[string]any)
		} else {
			call = args[0].([]any)[0].([]any)[0].(map[string]any)
		}
		code := call["input"].(hexutil.Bytes)
		out := fixtureProbe(code, f.states[len(f.states)-1], (len(code)-4)/25)
		result := f.fixtureParity(1)
		result.Output = &out
		if method == "trace_callMany" {
			return []*parityReplay{result, result}, nil
		}
		return result, nil
	case "debug_intermediateRoots":
		return f.extended.roots, nil
	case "debug_executionWitness":
		return f.extended.witness, nil
	case "debug_getRawBlock":
		return f.extended.raw, nil
	case "debug_traceBlock", "debug_traceBadBlock":
		var traces []replayBlockFrame
		for _, tx := range f.header.Transactions {
			raw, _ := json.Marshal(replayCallFrame{GasUsed: uint64Ptr(uint64(f.receipts[tx.Hash].GasUsed))})
			traces = append(traces, replayBlockFrame{Result: raw, TxHash: ptr.New(tx.Hash)})
		}
		return traces, nil
	case "debug_getBadBlocks":
		if f.extended.bad {
			raw, err := invalidGasBlock(f.extended.raw, f.header.Hash)
			if err != nil {
				return nil, err
			}
			var body []rlp.RawValue
			if err := rlp.DecodeBytes(raw, &body); err != nil {
				return nil, err
			}
			return []any{map[string]any{"hash": crypto.Keccak256Hash(body[0]), "rlp": raw}}, nil
		}
		return []any{}, nil
	}
	return nil, fmt.Errorf("unexpected RPC %s", method)
}

func TestCanonicalAccountExistence(t *testing.T) {
	absent, err := encodeCanonicalAccount(nil)
	require.NoError(t, err)
	require.Nil(t, absent)
	empty, err := encodeCanonicalAccount(&canonicalReplayAccount{Balance: bigPtr(0), Nonce: uint64Ptr(0), StorageRoot: ptr.New(types.EmptyRootHash), CodeHash: ptr.New(types.EmptyCodeHash)})
	require.NoError(t, err)
	require.NotEmpty(t, empty)
}

func (f *rpcReplayFixture) Subscribe(ctx context.Context, namespace string, ch any, args ...any) (*gethrpc.ClientSubscription, error) {
	return f.extended.client.Subscribe(ctx, namespace, ch, args...)
}

// op-service/client wrappers return ethereum.Subscription, while the raw geth
// client returns *rpc.ClientSubscription. Both are used by the real CLI.
type wrappedReplayFixture struct{ *rpcReplayFixture }

func (f wrappedReplayFixture) Subscribe(ctx context.Context, namespace string, channel any, args ...any) (ethereum.Subscription, error) {
	return f.rpcReplayFixture.Subscribe(ctx, namespace, channel, args...)
}
func TestTraceChainWrappedClientAndCancellation(t *testing.T) {
	f := replayFixture(t)
	var receipts []*RPCReceipt
	for _, tx := range f.header.Transactions {
		receipts = append(receipts, f.receipts[tx.Hash])
	}
	require.NoError(t, checkTraceChain(context.Background(), wrappedReplayFixture{f}, 42, f.header.Hash, receipts))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, checkTraceChain(ctx, wrappedReplayFixture{f}, 42, f.header.Hash, receipts), context.Canceled)
}

type fixtureDebugSubscription struct{ f *rpcReplayFixture }

func (api *fixtureDebugSubscription) TraceChain(ctx context.Context, start, end string, opts map[string]any) (*gethrpc.Subscription, error) {
	notifier, ok := gethrpc.NotifierFromContext(ctx)
	if !ok {
		return nil, errors.New("no notifier")
	}
	sub := notifier.CreateSubscription()
	f := api.f
	event := chainReplay{Block: uint64Ptr(42), Hash: f.header.Hash}
	for i, tx := range f.header.Transactions {
		raw, _ := json.Marshal(f.fixtureCall(i))
		event.Traces = append(event.Traces, &replayBlockFrame{Result: raw, TxHash: ptr.New(tx.Hash)})
	}
	go func() { _ = notifier.Notify(sub.ID, event) }()
	return sub, nil
}
