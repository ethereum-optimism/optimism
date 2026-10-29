package dsl

import (
	optypes "github.com/ethereum-optimism/optimism/op-core/types"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// WaitForDeposit waits until the deposit an L1 transaction made through this chain's portal has
// executed on this chain, and returns its L2 receipt without asserting its status.
func (el *L2ELNode) WaitForDeposit(portal common.Address, l1Receipt *types.Receipt) *types.Receipt {
	var deposit *optypes.DepositTx
	for _, l := range l1Receipt.Logs {
		if l.Address != portal {
			continue
		}
		if tx, err := derive.UnmarshalDepositLogEvent(l); err == nil {
			deposit = tx
			break
		}
	}
	el.require.NotNil(deposit, "expected a TransactionDeposited event from portal %s in L1 tx %s", portal, l1Receipt.TxHash)
	el.log.Info("Waiting for deposit to execute on L2",
		"l1Tx", l1Receipt.TxHash, "l1Block", l1Receipt.BlockNumber, "l2Tx", deposit.Hash())
	el.WaitL1OriginReached(eth.Unsafe, bigs.Uint64Strict(l1Receipt.BlockNumber), 120)
	return el.WaitForReceipt(deposit.Hash())
}

// CallFrame is one call in a callTracer trace of a transaction.
type CallFrame struct {
	From   common.Address `json:"from"`
	To     common.Address `json:"to"`
	Input  hexutil.Bytes  `json:"input"`
	Output hexutil.Bytes  `json:"output"`
	Error  string         `json:"error"`
	Calls  []CallFrame    `json:"calls"`
}

// Find returns the first frame, depth first from this one, that matches.
func (f CallFrame) Find(match func(CallFrame) bool) (CallFrame, bool) {
	if match(f) {
		return f, true
	}
	for _, c := range f.Calls {
		if found, ok := c.Find(match); ok {
			return found, true
		}
	}
	return CallFrame{}, false
}

// TraceCalls traces a transaction on this chain with the callTracer, return data included.
func (el *L2ELNode) TraceCalls(txHash common.Hash) CallFrame {
	el.log.Info("Tracing transaction", "tx", txHash)
	var trace CallFrame
	err := el.EthClient().RPC().CallContext(el.ctx, &trace, "debug_traceTransaction", txHash,
		map[string]any{
			"enableReturnData": true,
			"tracer":           "callTracer",
			"tracerConfig":     map[string]any{},
		})
	el.require.NoError(err, "failed to trace tx %s", txHash)
	return trace
}
