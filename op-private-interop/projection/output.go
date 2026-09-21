package projection

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// PrivateOutput uses the standard OP output-v0 commitment. The caller must use
// this only for Isthmus-active private payloads: withdrawalsRoot then commits to
// the message-passer storage trie. An absent root is never replaced with zero.
func PrivateOutput(payload *eth.ExecutionPayload) (common.Hash, error) {
	if payload == nil || payload.WithdrawalsRoot == nil {
		return common.Hash{}, fmt.Errorf("private output requires an Isthmus withdrawals root")
	}
	return common.Hash(eth.OutputRoot(&eth.OutputV0{
		StateRoot: payload.StateRoot, MessagePasserStorageRoot: eth.Bytes32(*payload.WithdrawalsRoot),
		BlockHash: payload.BlockHash,
	})), nil
}

// MetadataOnly identifies checkpoint calls for sequencer-drift scheduling. It
// does not replace whole-span admission, which checks placement and envelopes.
// Any message replay is application activity, even when its payload is empty.
func MetadataOnly(txs []hexutil.Bytes) bool {
	for _, raw := range txs {
		var tx types.Transaction
		if tx.UnmarshalBinary(raw) != nil || tx.Type() != types.DynamicFeeTxType || tx.To() == nil || *tx.To() != predeploys.ClaimRegistryAddr {
			return false
		}
		if _, err := wire.DecodeOutput(tx.Data()); err == nil {
			continue
		}
		if _, err := wire.DecodeClaim(tx.Data()); err != nil {
			return false
		}
	}
	return true
}
