package chain_container

// Random L1/L2 chain data for fuzzing the interop phases. A RandomChainManager
// generates valid chains; each RandomChain child implements
// virtual_node.VirtualNode and the engine_controller l2Provider set.

import (
	"sort"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type L2Block struct {
	Ref      eth.L2BlockRef
	Payload  *eth.ExecutionPayloadEnvelope
	ExecMsgs map[uint32]*supervisortypes.Message
}

// Output mirrors engine_controller.OutputV0AtBlockNumber's payload branch.
func (b *L2Block) Output() *eth.OutputV0 {
	p := b.Payload.ExecutionPayload
	var msgPasser eth.Bytes32
	if p.WithdrawalsRoot != nil {
		msgPasser = eth.Bytes32(*p.WithdrawalsRoot)
	}
	return &eth.OutputV0{
		StateRoot:                p.StateRoot,
		MessagePasserStorageRoot: msgPasser,
		BlockHash:                p.BlockHash,
	}
}

// Receipts encodes the executing messages as logs the interop decoder accepts.
func (b *L2Block) Receipts() gethtypes.Receipts {
	keys := make([]uint32, 0, len(b.ExecMsgs))
	for k := range b.ExecMsgs {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	logs := make([]*gethtypes.Log, 0, len(keys))
	for _, k := range keys {
		topics, data := b.ExecMsgs[k].EncodeEvent()
		logs = append(logs, &gethtypes.Log{
			Address: params.InteropCrossL2InboxAddress,
			Topics:  topics,
			Data:    data,
		})
	}
	return gethtypes.Receipts{{
		Type:   gethtypes.DynamicFeeTxType,
		Status: gethtypes.ReceiptStatusSuccessful,
		Logs:   logs,
	}}
}

// SafeHeadEntry is one SafeDB row: the L1 block at which an L2 safe head became
// safe. Sparse, ascending by L1 number.
type SafeHeadEntry struct {
	L1 eth.BlockID
	L2 eth.BlockID
}
