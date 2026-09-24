package batcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/builder"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
)

// Carrier pre-run (spec-sound-profile §E.3).
//
// On the public projection every non-deposit transaction must execute successfully, or its whole
// block is invalid and derivation replaces the rest of the span with deposit-only blocks. Every
// sequencer transaction in a span is one of the batcher's carriers (the claim, the per-block
// output records and the message replays), so the batcher simulates each of them before it asks
// for a proof, and refuses to publish a span in which any would fail.

// ErrCarrierPreRun marks a span whose carrier transactions would fail on the projection. It is
// fatal for the range: the batcher does not publish it and does not silently retry it. The error
// names the block, the transaction index and the revert data.
var ErrCarrierPreRun = errors.New("carrier pre-run failed")

// ProjectionCaller simulates a call on the public projection's execution client at a given
// block, with a storage state override (eth_call's third parameter, `stateDiff` form), and reads
// an account nonce at a given block (eth_getTransactionCount by block hash).
type ProjectionCaller interface {
	CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash, overrides StateOverrides) ([]byte, error)
	NonceAtHash(ctx context.Context, account common.Address, blockHash common.Hash) (uint64, error)
}

// StateOverrides are per-account storage overrides: address → slot → value.
type StateOverrides map[common.Address]map[common.Hash]common.Hash

// l1BlockBatcherHashSlot is L1Block's `batcherHash` storage slot (the same in L1Block and
// L1BlockCGT; snapshots/storageLayout). postClaim authorizes its caller against it.
var l1BlockBatcherHashSlot = common.BigToHash(big.NewInt(4))

// carrierOverrides is the L1-info state the span's blocks run under that the span parent may not
// have yet. Carriers are simulated at the span parent, but each executes after its own block's
// L1-info deposit, which writes L1Block's batcherHash from the L1 SystemConfig. At the projection
// genesis the parent has no batcher hash at all, so every first claim would revert
// ClaimRegistry_NotBatcher. The batcher only renders private blocks whose L1 info names its own
// signer (PrepareBlock), and the projection derives the same L1 info, so the batcher hash every
// span block carries is exactly the batcher's own. The override makes the simulation exact at
// every parent, including after a batcher rotation in the first block's epoch. No carrier reads any
// other L1-info field: the replays read only the block environment and their calldata.
func carrierOverrides(batcher common.Address) StateOverrides {
	return StateOverrides{predeploys.L1BlockAddr: {l1BlockBatcherHashSlot: common.BytesToHash(batcher.Bytes())}}
}

// rpcProjectionCaller is ProjectionCaller over a JSON-RPC client.
type rpcProjectionCaller struct{ rpc *rpc.Client }

func newRPCProjectionCaller(cl *rpc.Client) ProjectionCaller { return &rpcProjectionCaller{rpc: cl} }

func (c *rpcProjectionCaller) CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash common.Hash, overrides StateOverrides) ([]byte, error) {
	arg := map[string]any{"from": msg.From, "input": hexutil.Bytes(msg.Data)}
	if msg.To != nil {
		arg["to"] = msg.To
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.AccessList != nil {
		arg["accessList"] = msg.AccessList
	}
	type accountOverride struct {
		StateDiff map[common.Hash]common.Hash `json:"stateDiff"`
	}
	state := make(map[common.Address]accountOverride, len(overrides))
	for addr, diff := range overrides {
		state[addr] = accountOverride{StateDiff: diff}
	}
	var out hexutil.Bytes
	err := c.rpc.CallContext(ctx, &out, "eth_call", arg, rpc.BlockNumberOrHashWithHash(blockHash, false), state)
	return out, err
}

func (c *rpcProjectionCaller) NonceAtHash(ctx context.Context, account common.Address, blockHash common.Hash) (uint64, error) {
	var out hexutil.Uint64
	err := c.rpc.CallContext(ctx, &out, "eth_getTransactionCount", account, rpc.BlockNumberOrHashWithHash(blockHash, false))
	return uint64(out), err
}

// carrierCallTimeout bounds one simulated carrier call.
const carrierCallTimeout = 10 * time.Second

// checkCarrierGas is the static check: a carrier must declare at least its intrinsic gas and the
// EIP-7623 calldata floor, the Prague rules the projection runs under. A transaction below either
// is not even includable; one between them and its execution cost fails in the simulation.
//
// It is the same bound projection admission enforces (projection.MinTxGas), so a span failing it
// would be dropped on chain.
func checkCarrierGas(tx *types.Transaction) error {
	if tx.To() == nil {
		return errors.New("carrier is a contract creation")
	}
	need, err := projection.MinTxGas(tx.Data(), tx.AccessList())
	if err != nil {
		return err
	}
	if tx.Gas() < need {
		return fmt.Errorf("gas limit %d is below the %d intrinsic and calldata-floor gas", tx.Gas(), need)
	}
	return nil
}

// checkRangeCarrierGas applies checkCarrierGas to every carrier of a built range.
func checkRangeCarrierGas(built *builder.BuiltRange) error {
	return forEachCarrier(built, func(block uint64, index int, tx *types.Transaction) error {
		if err := checkCarrierGas(tx); err != nil {
			return fmt.Errorf("%w: block %d tx %d: %w", ErrCarrierPreRun, block, index, err)
		}
		return nil
	})
}

// preRunCarriers statically checks and then simulates every carrier of the candidate range, in
// order, with eth_call on the projection at the span parent. The replay contracts are stateless
// and postClaim reads only L1Block.batcherHash() and registry storage, so simulating each carrier
// at the parent, with L1Block's batcher hash overridden to the one the span's L1-info deposits
// write (carrierOverrides), is exact.
func preRunCarriers(ctx context.Context, caller ProjectionCaller, from common.Address, parent common.Hash, candidate *builder.BuiltRange) error {
	if err := checkRangeCarrierGas(candidate); err != nil {
		return err
	}
	if caller == nil {
		return fmt.Errorf("%w: no projection execution client to simulate carriers on", ErrCarrierPreRun)
	}
	if err := checkCarrierNonces(ctx, caller, from, parent, candidate); err != nil {
		return err
	}
	overrides := carrierOverrides(from)
	return forEachCarrier(candidate, func(block uint64, index int, tx *types.Transaction) error {
		callCtx, cancel := context.WithTimeout(ctx, carrierCallTimeout)
		defer cancel()
		_, err := caller.CallContractAtHash(callCtx, ethereum.CallMsg{
			From:       from,
			To:         tx.To(),
			Gas:        tx.Gas(),
			GasPrice:   new(big.Int),
			Value:      tx.Value(),
			Data:       tx.Data(),
			AccessList: tx.AccessList(),
		}, parent, overrides)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			// The job was cancelled (reset or shutdown): not a verdict on the span.
			return ctx.Err()
		}
		var data any
		var de rpc.DataError
		if errors.As(err, &de) {
			data = de.ErrorData()
		}
		var re rpc.Error
		if !errors.As(err, &re) && data == nil {
			// Transport failure: the carrier was never judged, so the range is retried.
			return fmt.Errorf("simulating block %d tx %d: %w", block, index, err)
		}
		return fmt.Errorf("%w: block %d tx %d to %s: %w (revert data: %v)", ErrCarrierPreRun, block, index, tx.To(), err, data)
	})
}

// checkCarrierNonces requires the carriers' nonces to be contiguous from the batcher's nonce at
// the span parent. A nonce gap or reuse makes a carrier an invalid transaction, which invalidates
// its projection block (ProjectionSequencerTxInvalid) and drops the rest of the span. Admission
// cannot catch it, because it has no state, and eth_call ignores nonces, so the simulation cannot
// either. A failure to read the nonce is not a verdict on the span and stays retryable.
func checkCarrierNonces(ctx context.Context, caller ProjectionCaller, from common.Address, parent common.Hash, candidate *builder.BuiltRange) error {
	callCtx, cancel := context.WithTimeout(ctx, carrierCallTimeout)
	defer cancel()
	next, err := caller.NonceAtHash(callCtx, from, parent)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("reading the batcher's nonce at the span parent %s: %w", parent, err)
	}
	return forEachCarrier(candidate, func(block uint64, index int, tx *types.Transaction) error {
		if tx.Nonce() != next {
			return fmt.Errorf("%w: block %d tx %d has nonce %d, want %d (contiguous from the batcher's nonce at the span parent %s)",
				ErrCarrierPreRun, block, index, tx.Nonce(), next, parent)
		}
		next++
		return nil
	})
}

// forEachCarrier visits every sequencer transaction of a built range in order.
func forEachCarrier(built *builder.BuiltRange, fn func(block uint64, index int, tx *types.Transaction) error) error {
	for _, blk := range built.Blocks {
		for i, raw := range blk.Txs {
			var tx types.Transaction
			if err := tx.UnmarshalBinary(raw); err != nil {
				return fmt.Errorf("%w: block %d tx %d does not decode: %w", ErrCarrierPreRun, blk.Number, i, err)
			}
			if err := fn(blk.Number, i, &tx); err != nil {
				return err
			}
		}
	}
	return nil
}
