package sdm

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rlp"
)

const SDMTxType = 0x7d

// PostExecPayloadVersion is the only PostExecPayload version the Go decoder accepts.
// Must stay in lock-step with POST_EXEC_PAYLOAD_VERSION in rust/op-alloy, which rejects
// unknown versions at decode time; Go accepting what Rust rejects is a cross-language
// drift hazard on any replay/verifier pipeline sitting between the two.
const PostExecPayloadVersion uint64 = 1

// SDMGasEntry is one per-transaction refund entry inside the SDM portion of a post-exec payload.
type SDMGasEntry struct {
	Index     uint64 `json:"index"`
	GasRefund uint64 `json:"gas_refund"`
}

// PostExecPayload is the decoded RLP payload carried by the synthetic post-exec tx.
// Contains the SDM gas refund entries and the L2 block number the payload is anchored to.
type PostExecPayload struct {
	Version          uint64        `json:"version"`
	BlockNumber      uint64        `json:"block_number,omitempty"`
	GasRefundEntries []SDMGasEntry `json:"gas_refund_entries"`
}

// DecodePayload decodes an RLP-encoded post-exec payload from the post-exec tx input.
func DecodePayload(input []byte) (*PostExecPayload, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty post-exec payload")
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
