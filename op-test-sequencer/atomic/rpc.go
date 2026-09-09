package atomic

import (
	"bytes"
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/client"
)

// RPCExecutor discovers and replays calls using op-geth's debug_traceCall.
// Parent must pin a block hash. This adapter simulates an empty next-block
// prefix; it is unsuitable when preceding transactions or system updates affect
// application execution. The resulting payloads must still be executed and
// checked by the normal engines and interop verifier.
type RPCExecutor struct {
	RPC                client.RPC
	Parent             common.Hash
	Number, Timestamp  uint64
	MaxDiscoveryPasses int
}

type callFrame struct {
	Type   string         `json:"type"`
	To     common.Address `json:"to"`
	Input  hexutil.Bytes  `json:"input"`
	Output hexutil.Bytes  `json:"output"`
	Error  string         `json:"error"`
	Calls  []callFrame    `json:"calls"`
	Logs   []callLog      `json:"logs"`
}

type callLog struct {
	Address  common.Address `json:"address"`
	Topics   []common.Hash  `json:"topics"`
	Data     hexutil.Bytes  `json:"data"`
	Position hexutil.Uint   `json:"position"`
}

func (e *RPCExecutor) trace(ctx context.Context, tx Transaction) (callFrame, error) {
	var frame callFrame
	var header *types.Header
	if err := e.RPC.CallContext(ctx, &header, "eth_getBlockByHash", e.Parent, false); err != nil {
		return frame, err
	}
	if header == nil || header.BaseFee == nil {
		return frame, fmt.Errorf("missing pinned parent/base fee")
	}
	price := new(big.Int).Add(header.BaseFee, big.NewInt(1))
	err := e.RPC.CallContext(ctx, &frame, "debug_traceCall", map[string]any{
		"from": tx.From, "to": tx.To, "data": hexutil.Bytes(tx.Data), "gas": hexutil.Uint64(tx.Gas), "accessList": tx.AccessList, "gasPrice": (*hexutil.Big)(price),
	}, rpc.BlockNumberOrHashWithHash(e.Parent, true), map[string]any{
		"tracer": "callTracer", "tracerConfig": map[string]any{"withLog": true},
		"blockOverrides": map[string]any{"number": hexutil.Uint64(e.Number), "time": hexutil.Uint64(e.Timestamp)},
	})
	return frame, err
}

func (e *RPCExecutor) Discover(ctx context.Context, tx Transaction) (Execution, error) {
	// Each cold checksum causes a normal inbox revert. Add the requested key
	// only to the next speculative attempt. No bytecode override is necessary,
	// and the final builder-produced transaction has its own complete access list.
	tx.AccessList = append(types.AccessList(nil), tx.AccessList...)
	seen := make(map[common.Hash]bool)
	for _, tuple := range tx.AccessList {
		if tuple.Address == predeploys.CrossL2InboxAddr {
			for _, key := range tuple.StorageKeys {
				seen[key] = true
			}
		}
	}
	for range e.MaxDiscoveryPasses {
		frame, err := e.trace(ctx, tx)
		if err != nil {
			return Execution{}, err
		}
		var added []common.Hash
		var visit func(callFrame)
		visit = func(f callFrame) {
			if f.Type == "CALL" {
				if checksum, ok := requestedChecksum(f.To, f.Input); ok && !seen[checksum] {
					seen[checksum] = true
					added = append(added, checksum)
				}
			}
			for _, child := range f.Calls {
				visit(child)
			}
		}
		visit(frame)
		if len(added) == 0 {
			return executionFromFrame(frame)
		}
		tx.AccessList = append(tx.AccessList, types.AccessTuple{Address: predeploys.CrossL2InboxAddr, StorageKeys: added})
	}
	return Execution{}, fmt.Errorf("exceeded %d RPC discovery passes", e.MaxDiscoveryPasses)
}

func (e *RPCExecutor) Replay(ctx context.Context, tx Transaction) (Execution, error) {
	frame, err := e.trace(ctx, tx)
	if err != nil {
		return Execution{}, err
	}
	return executionFromFrame(frame)
}

func executionFromFrame(frame callFrame) (Execution, error) {
	out := Execution{Output: frame.Output, Reverted: frame.Error != ""}
	var visit func(callFrame) error
	visit = func(f callFrame) error {
		if f.Error != "" {
			return nil
		}
		idx := 0
		for position := 0; position <= len(f.Calls); position++ {
			for idx < len(f.Logs) && uint64(f.Logs[idx].Position) == uint64(position) {
				log := f.Logs[idx]
				out.Logs = append(out.Logs, &types.Log{Address: log.Address, Topics: log.Topics, Data: log.Data})
				idx++
			}
			if position < len(f.Calls) {
				if err := visit(f.Calls[position]); err != nil {
					return err
				}
			}
		}
		if idx != len(f.Logs) {
			return fmt.Errorf("invalid callTracer log positions")
		}
		return nil
	}
	err := visit(frame)
	return out, err
}

func requestedChecksum(to common.Address, input []byte) (common.Hash, bool) {
	if to != predeploys.CrossL2InboxAddr || len(input) != 196 || !bytes.Equal(input[:4], validateSelector) {
		return common.Hash{}, false
	}
	id := Identifier{Origin: common.BytesToAddress(input[4:36]), BlockNumber: new(big.Int).SetBytes(input[36:68]), LogIndex: new(big.Int).SetBytes(input[68:100]), Timestamp: new(big.Int).SetBytes(input[100:132]), ChainId: new(big.Int).SetBytes(input[132:164])}
	if !id.BlockNumber.IsUint64() || !id.Timestamp.IsUint64() || id.LogIndex.BitLen() > 32 {
		return common.Hash{}, false
	}
	msg := message(id, common.BytesToHash(input[164:196]))
	return common.Hash(msg.Checksum()), true
}
