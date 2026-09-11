// Package remote provides the plugin boundary for ingesting initiating messages
// from chains that the supernode does NOT drive.
//
// A driven chain is represented by an in-process virtual node that runs a full
// op-node and seals that chain's logs into a LogsDB. A *remote* chain, by
// contrast, is represented only by an [Adapter]: a small plugin that exposes the
// chain's FINALIZED initiating messages, already normalized into the canonical
// form the interop verifier needs. The supernode ingests those messages into a
// LogsDB so the driven chains in the interop set can reference them as executing
// messages — without the supernode running a consensus client for the remote
// chain.
//
// This lets an operator plug in any external chain node software. The production
// Adapter is [HTTPAdapter]: point it at a URL and the supernode polls that endpoint
// for the chain's finalized initiating messages, so the wire protocol — not a Go
// type — is the real plugin boundary (serve the right responses in any language and
// you are done). The remotetest package provides a deterministic test server for it.
package remote

import (
	"context"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// InitiatingMessage is a single initiating (cross-chain) message emitted by a
// remote chain, normalized to the minimum the interop verifier needs to make it
// referenceable by an executing message on another chain. It is also the JSON wire
// type a remote node serves (see [HTTPAdapter]).
//
// LogHash is the canonical content hash of the originating log — identical to
// what messages.LogToLogHash produces for a real EVM log. The verifier derives
// the referenceable checksum from (LogHash, block number, LogIndex, block
// timestamp, chain ID).
type InitiatingMessage struct {
	// LogIndex is the position of this message's log within its block. Indices
	// must be contiguous and start at 0: the verifier indexes messages at exactly
	// these positions.
	LogIndex uint32 `json:"logIndex"`
	// LogHash is the canonical content hash of the originating log.
	LogHash common.Hash `json:"logHash"`
}

// FinalizedBlock is one finalized block's worth of initiating messages from a
// remote chain, and the JSON wire type a remote node serves. Because remote chains
// are ingested finalized-only, blocks are assumed irreversible: there is no
// reorg/rewind path for them.
type FinalizedBlock struct {
	Number     uint64      `json:"number"`
	Hash       common.Hash `json:"hash"`
	ParentHash common.Hash `json:"parentHash"`
	Timestamp  uint64      `json:"timestamp"`
	// Messages are the initiating messages in this block, ordered by ascending
	// LogIndex starting at 0. May be empty (a block with no cross-chain messages).
	Messages []InitiatingMessage `json:"messages"`
}

// Adapter is the plugin boundary: an implementation wraps any external chain node
// software and exposes that chain's finalized initiating messages. An adapter is
// only ever consulted by a single ingester goroutine, so implementations need not
// be safe for concurrent use.
type Adapter interface {
	// ChainID identifies the remote chain. It must be a member of the interop
	// dependency set for its messages to be referenceable.
	ChainID() eth.ChainID

	// BlockTime is the remote chain's block time in seconds. The verifier uses it
	// for the interop activation invariant (a message must be at least one block
	// past activation on its own chain).
	BlockTime() uint64

	// NextFinalized returns the next finalized block after block number `after`
	// (i.e. block number after+1), or ok=false if that block is not finalized yet.
	//
	// after == 0 is the ANCHOR request: the supernode has no local history for this
	// chain and asks where to start. The adapter may answer with a block at any
	// height >= 1 — its chain's current finalized head, for instance — and that block
	// becomes the anchor of the locally recorded history. It is the adapter's choice,
	// and it must be stable across adapter restarts once the supernode has sealed
	// history rooted at it. Block 0 is the implicit genesis and carries no messages,
	// so it is never a valid answer.
	//
	// For after > 0 the adapter MUST return exactly block after+1 (or ok=false until
	// it is finalized): the supernode enforces strict contiguity from the anchor
	// onward and rejects any other height.
	NextFinalized(ctx context.Context, after uint64) (blk *FinalizedBlock, ok bool, err error)

	// Close releases any resources held by the adapter.
	Close() error
}
