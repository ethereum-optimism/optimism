package sdm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type parityFrame struct {
	Type         string          `json:"type"`
	Action       json.RawMessage `json:"action"`
	Result       json.RawMessage `json:"result,omitempty"`
	Error        string          `json:"error,omitempty"`
	Subtraces    *uint64         `json:"subtraces"`
	TraceAddress []uint64        `json:"traceAddress"`
}
type localizedParity struct {
	parityFrame
	BlockHash           *common.Hash `json:"blockHash"`
	BlockNumber         *uint64      `json:"blockNumber"`
	TransactionHash     *common.Hash `json:"transactionHash"`
	TransactionPosition *uint64      `json:"transactionPosition"`
}
type parityReplay struct {
	Output    *hexutil.Bytes `json:"output"`
	Trace     []parityFrame  `json:"trace"`
	StateDiff map[common.Address]struct {
		Balance json.RawMessage `json:"balance"`
	} `json:"stateDiff"`
	TransactionHash *common.Hash `json:"transactionHash"`
}

type replayRecorder func(string, error)

func replayAddresses(header *replayHeader, tx rpcTransactionDetails) []common.Address {
	return []common.Address{tx.From, header.Miner, predeploys.BaseFeeVaultAddr, predeploys.L1FeeVaultAddr, predeploys.OperatorFeeVaultAddr}
}

func checkExtendedRPC(ctx context.Context, rpc Caller, blockNum uint64, header *replayHeader, receipts []*RPCReceipt, oracle *replayBalances, options RPCReplayOptions, record replayRecorder) {
	checkParityRPC(ctx, rpc, blockNum, header, receipts, oracle, record)
	reverted := false
	for i, tx := range header.Transactions {
		if ctx.Err() != nil {
			record("extended replay context", ctx.Err())
			return
		}
		if uint64(tx.Type) == optypes.DepositTxType || uint64(tx.Type) == optypes.PostExecTxType {
			continue
		}
		label := fmt.Sprintf("tx[%d] %s", i, tx.Hash)
		addresses := replayAddresses(header, tx)
		reverted = checkJSAndOTS(ctx, rpc, tx, receipts[i], label, options, record) || reverted
		if receipts[i].OPGasRefund != nil {
			checkCallMany(ctx, rpc, oracle, i, addresses, record)
		}
	}
	if options.CheckOTS && !reverted {
		record("ots_getTransactionError reverted-transaction coverage", replayCoverageGap("selected block has no failed transaction; successful-transaction null responses are still checked"))
	}
	record("debug_intermediateRoots canonical transaction roots", checkIntermediateRoots(ctx, rpc, header))
	record("debug_subscribe/traceChain", checkTraceChain(ctx, rpc, blockNum, header.Hash, receipts))
	checkRawDiagnostics(ctx, rpc, header, record)
	for i, receipt := range receipts {
		if receipt.OPGasRefund != nil {
			checkBoundaryCalls(ctx, rpc, header, header.Transactions[i], record)
			break
		}
	}
}

func checkParityRPC(ctx context.Context, rpc Caller, blockNum uint64, header *replayHeader, receipts []*RPCReceipt, oracle *replayBalances, record replayRecorder) {
	expected := make([][]parityFrame, len(receipts))
	for i, tx := range header.Transactions {
		if ctx.Err() != nil {
			record("parity context", ctx.Err())
			return
		}
		label := fmt.Sprintf("tx[%d] %s", i, tx.Hash)
		var replay *parityReplay
		err := rpc.CallContext(ctx, &replay, "trace_replayTransaction", tx.Hash, []string{"trace", "stateDiff"})
		if err == nil {
			err = checkParityReplay(ctx, oracle, i, header, receipts[i], replay)
		}
		record(label+" trace_replayTransaction/stateDiff", err)
		// Even an unsettled state diff can provide frame trees for wrapper consistency.
		if replay != nil && checkParityFrames(replay.Trace) == nil {
			expected[i] = replay.Trace
		}
		for _, method := range []string{"trace_transaction", "trace_get"} {
			var traces []localizedParity
			var err error
			if method == "trace_get" {
				var trace *localizedParity
				err = rpc.CallContext(ctx, &trace, method, tx.Hash, []uint64{0})
				if trace != nil {
					traces = append(traces, *trace)
				}
			} else {
				err = rpc.CallContext(ctx, &traces, method, tx.Hash)
			}
			if err == nil {
				want := expected[i]
				if method == "trace_get" && len(want) > 0 {
					want = want[:1]
				}
				err = checkLocalized(traces, want, header.Hash, blockNum, i, tx.Hash)
			}
			record(label+" "+method, err)
		}
	}
	var blockReplays []*parityReplay
	err := rpc.CallContext(ctx, &blockReplays, "trace_replayBlockTransactions", header.Hash, []string{"trace", "stateDiff"})
	if err == nil && len(blockReplays) != len(receipts) {
		err = fmt.Errorf("got %d results, want %d", len(blockReplays), len(receipts))
	}
	if err != nil {
		record("trace_replayBlockTransactions", err)
	} else {
		for i, replay := range blockReplays {
			err := checkParityReplay(ctx, oracle, i, header, receipts[i], replay)
			if replay == nil || replay.TransactionHash == nil || *replay.TransactionHash != header.Transactions[i].Hash {
				err = errors.Join(err, errors.New("missing/wrong transaction hash"))
			}
			if replay != nil {
				err = errors.Join(err, compareParity(replay.Trace, expected[i]))
			}
			record(fmt.Sprintf("trace_replayBlockTransactions tx[%d]", i), err)
		}
	}
	for _, method := range []string{"trace_block", "trace_filter"} {
		var traces []localizedParity
		var arg any = header.Hash
		if method == "trace_filter" {
			arg = map[string]any{"fromBlock": hexutil.EncodeUint64(blockNum), "toBlock": hexutil.EncodeUint64(blockNum)}
		}
		err := rpc.CallContext(ctx, &traces, method, arg)
		if err == nil {
			err = checkLocalizedBlock(traces, expected, header, blockNum)
		}
		record(method+" canonical frame trees", err)
		if method == "trace_filter" && err == nil && len(traces) > 1 {
			var page []localizedParity
			err = rpc.CallContext(ctx, &page, method, map[string]any{"fromBlock": hexutil.EncodeUint64(blockNum), "toBlock": hexutil.EncodeUint64(blockNum), "after": 1, "count": 1})
			if err == nil && !sameReplayJSON(page, traces[1:2]) {
				err = errors.New("pagination differs from unpaginated trace_filter")
			}
			record("trace_filter pagination", err)
		}
	}
}

func checkParityFrames(frames []parityFrame) error {
	if len(frames) == 0 || frames[0].TraceAddress == nil || len(frames[0].TraceAddress) != 0 {
		return errors.New("missing root parity frame")
	}
	paths := make(map[string]uint64)
	children := make(map[string]uint64)
	for _, frame := range frames {
		if frame.TraceAddress == nil || frame.Subtraces == nil || len(frame.Action) == 0 {
			return errors.New("incomplete parity frame")
		}
		key := fmt.Sprint(frame.TraceAddress)
		if _, exists := paths[key]; exists {
			return errors.New("duplicate parity trace address")
		}
		if len(frame.TraceAddress) > 0 {
			parent := fmt.Sprint(frame.TraceAddress[:len(frame.TraceAddress)-1])
			if _, exists := paths[parent]; !exists {
				return errors.New("parity child precedes/misses parent")
			}
			if frame.TraceAddress[len(frame.TraceAddress)-1] != children[parent] {
				return errors.New("non-contiguous parity child index")
			}
			children[parent]++
		}
		paths[key] = *frame.Subtraces
		if frame.Error == "" && (frame.Type == "call" || frame.Type == "create") {
			var result struct {
				GasUsed *hexutil.Uint64 `json:"gasUsed"`
			}
			if err := json.Unmarshal(frame.Result, &result); err != nil {
				return err
			}
			if result.GasUsed == nil {
				return errors.New("parity frame omits gasUsed")
			}
			// Parity frame gas is inspector gas, not net transaction receipt gas.
		}
	}
	for path, count := range paths {
		if children[path] != count {
			return errors.New("parity subtrace count mismatch")
		}
	}
	return nil
}

func compareParity(actual, want []parityFrame) error {
	if len(want) == 0 {
		return errors.New("no valid single-transaction frame reference")
	}
	if !sameReplayJSON(actual, want) {
		return errors.New("parity frame tree differs from single-transaction replay")
	}
	return nil
}

// Compare raw embedded JSON without depending on object field order or losing
// integer precision in RPC identifiers/counts.
func sameReplayJSON(a, b any) bool {
	decode := func(value any) (any, error) {
		encoded, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		var result any
		err = decoder.Decode(&result)
		return result, err
	}
	av, ae := decode(a)
	bv, be := decode(b)
	return ae == nil && be == nil && reflect.DeepEqual(av, bv)
}

func checkLocalized(traces []localizedParity, want []parityFrame, hash common.Hash, number uint64, index int, tx common.Hash) error {
	frames := make([]parityFrame, 0, len(traces))
	for _, trace := range traces {
		if trace.BlockHash == nil || *trace.BlockHash != hash || trace.BlockNumber == nil || *trace.BlockNumber != number || trace.TransactionHash == nil || *trace.TransactionHash != tx || trace.TransactionPosition == nil || *trace.TransactionPosition != uint64(index) {
			return errors.New("localized parity block/transaction identity mismatch")
		}
		frames = append(frames, trace.parityFrame)
	}
	return compareParity(frames, want)
}

func checkLocalizedBlock(traces []localizedParity, want [][]parityFrame, header *replayHeader, number uint64) error {
	offset := 0
	for i, frames := range want {
		if len(frames) == 0 || offset+len(frames) > len(traces) {
			return errors.New("missing canonical parity frames")
		}
		if err := checkLocalized(traces[offset:offset+len(frames)], frames, header.Hash, number, i, header.Transactions[i].Hash); err != nil {
			return err
		}
		offset += len(frames)
	}
	if offset != len(traces) {
		return errors.New("unexpected extra parity frames")
	}
	return nil
}

func checkParityReplay(ctx context.Context, oracle *replayBalances, index int, header *replayHeader, receipt *RPCReceipt, replay *parityReplay) error {
	if replay == nil || replay.Output == nil || replay.StateDiff == nil {
		return errors.New("missing parity replay/output/stateDiff")
	}
	if err := checkParityFrames(replay.Trace); err != nil {
		return err
	}
	if (replay.Trace[0].Error != "") != (receipt.Status == 0) {
		return errors.New("parity execution status differs from receipt")
	}
	tx := header.Transactions[index]
	if uint64(tx.Type) != optypes.DepositTxType && uint64(tx.Type) != optypes.PostExecTxType {
		if (tx.To == nil && replay.Trace[0].Type != "create") || (tx.To != nil && replay.Trace[0].Type != "call") {
			return errors.New("parity root action type differs from canonical transaction")
		}
		var action struct {
			From  common.Address  `json:"from"`
			To    *common.Address `json:"to"`
			Input hexutil.Bytes   `json:"input"`
			Init  hexutil.Bytes   `json:"init"`
		}
		if err := json.Unmarshal(replay.Trace[0].Action, &action); err != nil {
			return err
		}
		data := action.Input
		if tx.To == nil {
			data = action.Init
		}
		if action.From != tx.From || !reflect.DeepEqual(action.To, tx.To) || !bytes.Equal(data, tx.Input) {
			return errors.New("parity root action differs from canonical transaction")
		}
	}
	diff := replayDiff{Pre: make(map[common.Address]replayAccount), Post: make(map[common.Address]replayAccount)}
	addresses := replayAddresses(header, tx)
	for address, account := range replay.StateDiff {
		addresses = append(addresses, address)
		raw := bytes.TrimSpace(account.Balance)
		if bytes.Equal(raw, []byte(`"="`)) {
			continue
		}
		var delta map[string]json.RawMessage
		if err := json.Unmarshal(raw, &delta); err != nil || len(delta) != 1 {
			return errors.New("malformed parity balance delta")
		}
		for kind, value := range delta {
			if kind == "*" {
				var change struct{ From, To *hexutil.Big }
				if err := json.Unmarshal(value, &change); err != nil {
					return err
				}
				if change.From == nil || change.To == nil {
					return errors.New("missing parity balance from/to")
				}
				diff.Pre[address] = replayAccount{Balance: change.From}
				diff.Post[address] = replayAccount{Balance: change.To}
			} else if kind == "+" || kind == "-" {
				var balance *hexutil.Big
				if err := json.Unmarshal(value, &balance); err != nil {
					return err
				}
				if balance == nil {
					return errors.New("missing parity balance")
				}
				if kind == "+" {
					diff.Post[address] = replayAccount{Balance: balance}
				} else {
					diff.Pre[address] = replayAccount{Balance: balance}
				}
			} else {
				return errors.New("unknown parity balance delta")
			}
		}
	}
	return checkReplayDiff(ctx, oracle, index, addresses, diff)
}

// Same canonical account oracle is used for read-only boundary calls, but these
// APIs start AFTER the block's finalization, not at a transaction index.
func boundaryBalances(ctx context.Context, rpc Caller, header *replayHeader, addresses []common.Address) (*replayBalances, error) {
	oracle := &replayBalances{cache: make(map[balanceAt]*big.Int)}
	for _, address := range addresses {
		var balance *hexutil.Big
		if err := rpc.CallContext(ctx, &balance, "eth_getBalance", address, map[string]any{"blockHash": header.Hash, "requireCanonical": true}); err != nil {
			return nil, err
		}
		if balance == nil {
			return nil, errors.New("missing boundary balance")
		}
		oracle.cache[balanceAt{0, address}] = new(big.Int).Set((*big.Int)(balance))
	}
	return oracle, nil
}
