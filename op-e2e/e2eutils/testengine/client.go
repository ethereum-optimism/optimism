package testengine

import (
	"context"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ipc"
)

// Client is a thin typed client for an op-reth-test-engine process: the Engine API methods the
// tests drive directly, the engine's optest_ namespace, and the raw RPC client for everything else.
type Client struct {
	proc *ipc.Proc
	rpc  *rpc.Client
}

// Spawn starts the engine binary at binPath over the op-geth-style genesis JSON at genesisPath and
// dials its socket. The engine's stderr is forwarded to logw. The returned Client owns the process
// and must be closed.
func Spawn(binPath, genesisPath string, logw io.Writer) (*Client, error) {
	proc, err := ipc.Spawn(binPath, socketFlag, []string{"--genesis", genesisPath}, logw)
	if err != nil {
		return nil, err
	}
	return &Client{proc: proc, rpc: proc.Client()}, nil
}

// RPC returns the raw RPC client, for the eth_ namespace and for go-ethereum's and op-node's
// clients. It is valid until Close.
func (c *Client) RPC() *rpc.Client {
	return c.rpc
}

// Close stops the engine process. It is idempotent.
func (c *Client) Close() {
	c.proc.Close()
}

// ForkchoiceUpdated calls method, a version of engine_forkchoiceUpdated; attrs may be nil.
func (c *Client) ForkchoiceUpdated(ctx context.Context, method eth.EngineAPIMethod, state eth.ForkchoiceState, attrs *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	var res eth.ForkchoiceUpdatedResult
	if err := c.rpc.CallContext(ctx, &res, string(method), state, attrs); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetPayload calls method, a version of engine_getPayload.
func (c *Client) GetPayload(ctx context.Context, method eth.EngineAPIMethod, id eth.PayloadID) (*eth.ExecutionPayloadEnvelope, error) {
	var envelope eth.ExecutionPayloadEnvelope
	if err := c.rpc.CallContext(ctx, &envelope, string(method), id); err != nil {
		return nil, err
	}
	return &envelope, nil
}

// NewPayload calls method, a version of engine_newPayload, with the parameters that version takes,
// as op-node sends them: V3 adds empty blob versioned hashes and parentBeaconBlockRoot, V4 empty
// execution requests.
func (c *Client) NewPayload(ctx context.Context, method eth.EngineAPIMethod, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	var params []any
	switch method {
	case eth.NewPayloadV2:
		params = []any{payload}
	case eth.NewPayloadV3:
		params = []any{payload, []common.Hash{}, parentBeaconBlockRoot}
	case eth.NewPayloadV4:
		params = []any{payload, []common.Hash{}, parentBeaconBlockRoot, []hexutil.Bytes{}}
	default:
		return nil, fmt.Errorf("unsupported newPayload method %q", method)
	}
	var status eth.PayloadStatusV1
	if err := c.rpc.CallContext(ctx, &status, string(method), params...); err != nil {
		return nil, err
	}
	return &status, nil
}

// IncludeTxResult is the result of including a transaction in the block being built.
type IncludeTxResult struct {
	TxHash  common.Hash `json:"txHash"`
	GasUsed uint64      `json:"gasUsed"`
}

// IncludeTx executes the encoded transaction raw into the block being built (optest_includeTx). It
// returns a nil result if the block is force-empty, which drops the transaction.
func (c *Client) IncludeTx(ctx context.Context, raw hexutil.Bytes) (*IncludeTxResult, error) {
	var res *IncludeTxResult
	err := c.rpc.CallContext(ctx, &res, "optest_includeTx", raw)
	return res, err
}

// RemainingBlockGas returns the gas still available in the block being built
// (optest_remainingBlockGas).
func (c *Client) RemainingBlockGas(ctx context.Context) (uint64, error) {
	var gas uint64
	err := c.rpc.CallContext(ctx, &gas, "optest_remainingBlockGas")
	return gas, err
}

// ForcedEmpty reports whether the block being built is force-empty (optest_forcedEmpty).
func (c *Client) ForcedEmpty(ctx context.Context) (bool, error) {
	var forced bool
	err := c.rpc.CallContext(ctx, &forced, "optest_forcedEmpty")
	return forced, err
}

// SetForceEmpty sets the force-empty flag of the block being built (optest_setForceEmpty).
func (c *Client) SetForceEmpty(ctx context.Context, forced bool) error {
	var ok bool
	return c.rpc.CallContext(ctx, &ok, "optest_setForceEmpty", forced)
}

// IncludeNextTxResult is the result of including the next parked transaction from an account in
// the block being built: exactly one of TxHash, Skipped and NoTx is set.
type IncludeNextTxResult struct {
	TxHash  *common.Hash `json:"txHash"`
	GasUsed uint64       `json:"gasUsed"`
	// Skipped reports that the block is force-empty, which drops the transaction.
	Skipped bool `json:"skipped"`
	// NoTx reports that no parked transaction from the account has the next nonce.
	NoTx bool `json:"noTx"`
}

// IncludeNextTx executes the parked transaction from `from` with the account's next nonce into the
// block being built (optest_includeNextTx). eth_sendRawTransaction parks transactions.
func (c *Client) IncludeNextTx(ctx context.Context, from common.Address) (*IncludeNextTxResult, error) {
	var res IncludeNextTxResult
	if err := c.rpc.CallContext(ctx, &res, "optest_includeNextTx", from); err != nil {
		return nil, err
	}
	return &res, nil
}
