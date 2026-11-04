package sdm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	gethrpc "github.com/ethereum/go-ethereum/rpc"
)

type replayBlockFrame struct {
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
	TxHash *common.Hash    `json:"txHash"`
}
type chainReplay struct {
	Block  *hexutil.Uint64     `json:"block"`
	Hash   common.Hash         `json:"hash"`
	Traces []*replayBlockFrame `json:"traces"`
}
type replaySubscriber interface {
	Subscribe(context.Context, string, any, ...any) (ethereum.Subscription, error)
}
type gethReplaySubscriber interface {
	Subscribe(context.Context, string, any, ...any) (*gethrpc.ClientSubscription, error)
}

func checkTraceChain(ctx context.Context, rpc Caller, number uint64, hash common.Hash, receipts []*RPCReceipt) error {
	if number == 0 {
		return errors.New("traceChain requires a non-genesis block")
	}
	events := make(chan chainReplay, 1)
	args := []any{"traceChain", hexutil.EncodeUint64(number - 1), hexutil.EncodeUint64(number), map[string]any{"tracer": "callTracer"}}
	var sub ethereum.Subscription
	var err error
	switch subscriber := rpc.(type) {
	case replaySubscriber:
		sub, err = subscriber.Subscribe(ctx, "debug", events, args...)
	case gethReplaySubscriber:
		sub, err = subscriber.Subscribe(ctx, "debug", events, args...)
	default:
		return errors.New("traceChain requires a subscription-capable RPC client (use ws:// or wss://)")
	}
	if err != nil {
		return fmt.Errorf("traceChain subscription (requires ws:// or wss://): %w", err)
	}
	defer sub.Unsubscribe()
	select {
	case event := <-events:
		return checkChainReplay(event, number, hash, receipts)
	case err := <-sub.Err():
		if err == nil {
			return errors.New("traceChain ended without target block")
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func checkChainReplay(event chainReplay, number uint64, hash common.Hash, receipts []*RPCReceipt) error {
	if event.Block == nil || uint64(*event.Block) != number || event.Hash != hash {
		return errors.New("traceChain block identity mismatch")
	}
	if len(event.Traces) != len(receipts) {
		return errors.New("traceChain transaction/result count mismatch")
	}
	var failures []error
	for i, trace := range event.Traces {
		if trace == nil {
			failures = append(failures, fmt.Errorf("traceChain tx[%d] missing result", i))
			continue
		}
		if trace.Error != "" {
			failures = append(failures, fmt.Errorf("traceChain tx[%d]: %s", i, trace.Error))
			continue
		}
		if trace.TxHash == nil || *trace.TxHash != receipts[i].TxHash {
			failures = append(failures, fmt.Errorf("traceChain tx[%d] hash mismatch", i))
		}
		var frame replayCallFrame
		err := json.Unmarshal(trace.Result, &frame)
		if err == nil {
			err = checkCallGas(frame, receipts[i])
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("traceChain tx[%d]: %w", i, err))
		}
	}
	return errors.Join(failures...)
}

// These diagnostics intentionally do not require settled gas. Canonical and
// unvalidated blocks have distinct replay policies. Changing only header gasUsed
// must not route the raw block through canonical executor validation.
func checkRawDiagnostics(ctx context.Context, rpc Caller, header *replayHeader, record replayRecorder) {
	var raw hexutil.Bytes
	err := rpc.CallContext(ctx, &raw, "debug_getRawBlock", header.Hash)
	var altered hexutil.Bytes
	if err == nil {
		altered, err = invalidGasBlock(raw, header.Hash)
	}
	var baseline []replayBlockFrame
	if err == nil {
		err = rpc.CallContext(ctx, &baseline, "debug_traceBlock", raw, map[string]any{"tracer": "callTracer"})
	}
	if err == nil {
		err = checkDiagnosticFrames(baseline, len(header.Transactions))
	}
	record("debug_traceBlock raw diagnostic shape (not settled gas)", err)
	if err == nil {
		var invalid []replayBlockFrame
		err = rpc.CallContext(ctx, &invalid, "debug_traceBlock", altered, map[string]any{"tracer": "callTracer"})
		if err == nil && !sameReplayJSON(invalid, baseline) {
			err = errors.New("raw diagnostic changes when only header gasUsed is invalid")
		}
		record("debug_traceBlock invalid header gas diagnostic", err)
	}
	var bad []struct {
		Hash common.Hash   `json:"hash"`
		RLP  hexutil.Bytes `json:"rlp"`
	}
	err = rpc.CallContext(ctx, &bad, "debug_getBadBlocks")
	if err != nil {
		record("debug_traceBadBlock fixture lookup", err)
		return
	}
	if len(bad) == 0 {
		record("debug_traceBadBlock", replayCoverageGap("no retained bad block; canonical blocks are not a substitute for this fixture"))
		return
	}
	chosen := bad[0]
	if len(chosen.RLP) == 0 {
		record("debug_traceBadBlock", errors.New("bad-block fixture omits raw RLP"))
		return
	}
	var body []rlp.RawValue
	err = rlp.DecodeBytes(chosen.RLP, &body)
	if err != nil || len(body) < 3 || crypto.Keccak256Hash(body[0]) != chosen.Hash {
		record("debug_traceBadBlock", errors.New("bad-block raw RLP/header hash mismatch"))
		return
	}
	var transactions []rlp.RawValue
	if err := rlp.DecodeBytes(body[1], &transactions); err != nil {
		record("debug_traceBadBlock", err)
		return
	}
	var cached, uncached []replayBlockFrame
	err = rpc.CallContext(ctx, &cached, "debug_traceBadBlock", chosen.Hash, map[string]any{"tracer": "callTracer"})
	rawErr := rpc.CallContext(ctx, &uncached, "debug_traceBlock", chosen.RLP, map[string]any{"tracer": "callTracer"})
	err = errors.Join(err, rawErr)
	if err == nil {
		err = checkDiagnosticFrames(cached, len(transactions))
	}
	if err == nil && !sameReplayJSON(cached, uncached) {
		err = errors.New("cached bad-block trace differs from raw diagnostic")
	}
	record("debug_traceBadBlock "+chosen.Hash.Hex()+" raw diagnostic agreement", err)
}

func checkDiagnosticFrames(frames []replayBlockFrame, count int) error {
	if len(frames) != count {
		return errors.New("diagnostic trace count mismatch")
	}
	for i, trace := range frames {
		if trace.Error != "" {
			return fmt.Errorf("diagnostic tx[%d]: %s", i, trace.Error)
		}
		var frame replayCallFrame
		if err := json.Unmarshal(trace.Result, &frame); err != nil {
			return err
		}
		if frame.GasUsed == nil {
			return errors.New("diagnostic trace missing gasUsed")
		}
	}
	return nil
}

func invalidGasBlock(raw hexutil.Bytes, hash common.Hash) (hexutil.Bytes, error) {
	var body []rlp.RawValue
	if err := rlp.DecodeBytes(raw, &body); err != nil {
		return nil, err
	}
	if len(body) < 3 || crypto.Keccak256Hash(body[0]) != hash {
		return nil, errors.New("raw block header hash mismatch")
	}
	var header []rlp.RawValue
	if err := rlp.DecodeBytes(body[0], &header); err != nil {
		return nil, err
	}
	if len(header) < 15 {
		return nil, errors.New("raw block header too short")
	}
	var gasUsed uint64
	if err := rlp.DecodeBytes(header[10], &gasUsed); err != nil {
		return nil, err
	}
	if gasUsed == 0 {
		return nil, errors.New("fixture already has zero header gasUsed")
	}
	header[10] = rlp.RawValue{0x80} // deliberately incorrect gasUsed, leave all transactions intact
	encoded, err := rlp.EncodeToBytes(header)
	if err != nil {
		return nil, err
	}
	body[0] = encoded
	return rlp.EncodeToBytes(body)
}
