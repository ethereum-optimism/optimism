package types

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
)

// PostExecPayloadVersion is the only PostExecPayload version the Go decoder accepts.
// Must stay in lock-step with POST_EXEC_PAYLOAD_VERSION in rust/op-alloy.
const PostExecPayloadVersion uint64 = 1

// SDMGasEntry is one per-transaction gas refund entry. Index is block-global, so it counts
// the deposit transactions that derivation prepends ahead of the batched ones.
type SDMGasEntry struct {
	Index     uint64 `json:"index"`
	GasRefund uint64 `json:"gas_refund"`
}

// PostExecPayload is the payload RLP-encoded into the Data of a [PostExecTx].
// Field order must match op-alloy's PostExecPayload.
type PostExecPayload struct {
	Version               uint64        `json:"version"`
	BlockNumber           uint64        `json:"block_number,omitempty"`
	SelectedBaseFeePerGas uint64        `json:"selected_base_fee_per_gas"`
	GasRefundEntries      []SDMGasEntry `json:"gas_refund_entries"`
}

// DecodePostExecPayload decodes the bytes that follow the [PostExecTxType] type byte,
// mirroring op-alloy's framing and version checks: one well-formed RLP value with no
// trailing bytes, at a supported version.
//
// This validates the encoding only. Whether a payload is *correct* for its block — that
// entries stay in bounds, target non-deposit transactions, and claim no more gas than
// execution actually earned — is settled by the execution layer, which is the only place
// holding the full transaction list and the gas accounting.
func DecodePostExecPayload(input []byte) (*PostExecPayload, error) {
	if len(input) == 0 {
		return nil, errors.New("empty post-exec payload")
	}

	var payload PostExecPayload
	if err := rlp.DecodeBytes(input, &payload); err != nil {
		return nil, fmt.Errorf("decode post-exec payload: %w", err)
	}
	if payload.Version != PostExecPayloadVersion {
		return nil, fmt.Errorf(
			"unsupported post-exec payload version %d (expected %d)",
			payload.Version, PostExecPayloadVersion,
		)
	}
	return &payload, nil
}
