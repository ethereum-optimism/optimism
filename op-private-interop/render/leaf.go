package render

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-private-interop/projection"
	"github.com/ethereum-optimism/optimism/op-private-interop/wire"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// MessageLeaves computes the sp1-private-projection-v1 message leaves (§C.3.7) of one private
// block from its rendered logs, in rendered order. It is the log-side twin of the leaves admission
// builds from published replay calldata; both give the same bytes.
//
//   - A messenger SentMessage is an INIT leaf over its interop payload hash
//     keccak256(topics ‖ data). The log must round-trip through DecodeSentMessage exactly.
//   - A CrossL2Inbox ExecutingMessage is an EXEC leaf over keccak256(data(160) ‖ topics[1]).
//   - A configured extra emitter's log consumes its rendered index but has no v1 leaf.
//
// Logs that cannot be rendered (bridge sender or target, oversize, malformed) are fatal, exactly as
// in RenderBlock.
func MessageLeaves(blockNumber uint64, logs []RenderedLog) ([]common.Hash, error) {
	var out []common.Hash
	for _, rl := range logs {
		act, err := actionFor(rl)
		if err != nil {
			return nil, err
		}
		var kind byte
		var hash common.Hash
		switch act.Kind {
		case ReplayExport:
			topics, data := wire.SentMessageLog(act.Export)
			if !slices.Equal(topics, rl.Log.Topics) || !bytes.Equal(data, rl.Log.Data) {
				return nil, fmt.Errorf("%w: private log index %d: SentMessage does not round-trip", ErrUnrenderableLog, rl.PrivateLogIndex)
			}
			kind, hash = projection.MessageKindInit, crypto.Keccak256Hash(messages.LogToMessagePayload(rl.Log))
		case ReplayImport:
			if len(rl.Log.Topics) != 2 || len(rl.Log.Data) != 160 {
				return nil, fmt.Errorf("%w: private log index %d: malformed ExecutingMessage", ErrUnrenderableLog, rl.PrivateLogIndex)
			}
			kind, hash = projection.MessageKindExec, crypto.Keccak256Hash(rl.Log.Data, rl.Log.Topics[1][:])
		default:
			continue
		}
		out = append(out, projection.MessageLeaf(blockNumber, rl.RenderedLogIndex, kind, hash))
	}
	return out, nil
}
