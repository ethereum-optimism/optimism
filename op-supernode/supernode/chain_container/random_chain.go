package chain_container

// Random L1/L2 chain data for fuzzing the interop phases. A RandomChainManager
// generates valid chains; each RandomChain child implements
// virtual_node.VirtualNode and the engine_controller l2Provider set.

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type L2Block struct {
	Ref      eth.L2BlockRef
	Payload  *eth.ExecutionPayloadEnvelope
	ExecMsgs map[uint32]*supervisortypes.Message
}

// SafeHeadEntry is one SafeDB row: the L1 block at which an L2 safe head became
// safe. Sparse, ascending by L1 number.
type SafeHeadEntry struct {
	L1 eth.BlockID
	L2 eth.BlockID
}
