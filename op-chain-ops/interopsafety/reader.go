// Package interopsafety independently checks the cross-safe chains that op-supernodes
// report. It re-validates every CrossL2Inbox executing message from a start block up
// to each supernode's cross-safe head against the referenced initiating log, checks
// that all supernodes agree on the cross-safe chain, and tracks cross-safe progress.
//
// It trusts no supernode decision: each supernode's view is read from that supernode
// and its own ELs.
package interopsafety

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// ChainReader reads one chain as one supernode sees it: the supernode's sync status
// for the chain, and blocks and logs from the EL that supernode uses.
type ChainReader interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
	// Header returns the hash and timestamp of the canonical block at num.
	Header(ctx context.Context, num uint64) (common.Hash, uint64, error)
	// BlockLogs returns every log of the canonical block at num, with block-wide indices.
	BlockLogs(ctx context.Context, num uint64) ([]*ethTypes.Log, error)
	// ExecutingMessageLogs returns the CrossL2Inbox ExecutingMessage logs in [from, to].
	ExecutingMessageLogs(ctx context.Context, from, to uint64) ([]*ethTypes.Log, error)
	// HasReceipt reports whether tx is in the canonical chain.
	HasReceipt(ctx context.Context, tx common.Hash) (bool, error)
	// TxTypes returns the transaction types of the canonical block at num.
	TxTypes(ctx context.Context, num uint64) ([]uint64, error)
}

// RPCReader is a ChainReader backed by a supernode's per-chain rollup RPC and an EL RPC.
type RPCReader struct {
	Rollup  *gethrpc.Client
	EL      *gethrpc.Client
	Timeout time.Duration
}

var _ ChainReader = (*RPCReader)(nil)

func (r *RPCReader) call(ctx context.Context, c *gethrpc.Client, out any, method string, args ...any) error {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	return c.CallContext(ctx, out, method, args...)
}

func (r *RPCReader) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	var ss eth.SyncStatus
	if err := r.call(ctx, r.Rollup, &ss, "optimism_syncStatus"); err != nil {
		return nil, err
	}
	return &ss, nil
}

type rpcHeader struct {
	Hash      common.Hash    `json:"hash"`
	Timestamp hexutil.Uint64 `json:"timestamp"`
}

func (r *RPCReader) Header(ctx context.Context, num uint64) (common.Hash, uint64, error) {
	var h *rpcHeader
	if err := r.call(ctx, r.EL, &h, "eth_getBlockByNumber", hexutil.Uint64(num).String(), false); err != nil {
		return common.Hash{}, 0, err
	}
	if h == nil {
		return common.Hash{}, 0, fmt.Errorf("block %d not found", num)
	}
	return h.Hash, uint64(h.Timestamp), nil
}

func (r *RPCReader) BlockLogs(ctx context.Context, num uint64) ([]*ethTypes.Log, error) {
	hash, _, err := r.Header(ctx, num)
	if err != nil {
		return nil, err
	}
	var logs []*ethTypes.Log
	err = r.call(ctx, r.EL, &logs, "eth_getLogs", map[string]any{"blockHash": hash})
	return logs, err
}

func (r *RPCReader) ExecutingMessageLogs(ctx context.Context, from, to uint64) ([]*ethTypes.Log, error) {
	var logs []*ethTypes.Log
	err := r.call(ctx, r.EL, &logs, "eth_getLogs", map[string]any{
		"fromBlock": hexutil.Uint64(from).String(),
		"toBlock":   hexutil.Uint64(to).String(),
		"address":   predeploys.CrossL2InboxAddr,
		"topics":    [][]common.Hash{{messages.ExecutingMessageEventTopic}},
	})
	return logs, err
}

func (r *RPCReader) HasReceipt(ctx context.Context, tx common.Hash) (bool, error) {
	var rcpt json.RawMessage
	if err := r.call(ctx, r.EL, &rcpt, "eth_getTransactionReceipt", tx); err != nil {
		return false, err
	}
	return len(rcpt) > 0 && string(rcpt) != "null", nil
}

func (r *RPCReader) TxTypes(ctx context.Context, num uint64) ([]uint64, error) {
	var blk *struct {
		Transactions []struct {
			Type hexutil.Uint64 `json:"type"`
		} `json:"transactions"`
	}
	if err := r.call(ctx, r.EL, &blk, "eth_getBlockByNumber", hexutil.Uint64(num).String(), true); err != nil {
		return nil, err
	}
	if blk == nil {
		return nil, fmt.Errorf("block %d not found", num)
	}
	types := make([]uint64, len(blk.Transactions))
	for i, tx := range blk.Transactions {
		types[i] = uint64(tx.Type)
	}
	return types, nil
}
