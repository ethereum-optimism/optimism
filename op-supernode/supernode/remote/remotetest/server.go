// Package remotetest provides a deterministic HTTP server implementing the remote-node
// protocol (see remote.HTTPAdapter), for use in tests. Production code never imports it;
// it lets a test point a real remote.HTTPAdapter at an httptest.Server that serves a
// fabricated finalized-block stream — "serve the right responses and you're good."
package remotetest

import (
	"encoding/binary"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/remote"
)

const (
	defaultBlockTime    = uint64(2)
	defaultMsgsPerBlock = 1
)

// Config configures a [Chain].
type Config struct {
	// ChainID of the simulated remote chain. Required.
	ChainID eth.ChainID
	// BlockTime in seconds, reported to the client. Defaults to 2 when zero.
	BlockTime uint64
	// MsgsPerBlock is the number of dummy initiating messages fabricated per block.
	// Defaults to 1 when zero.
	MsgsPerBlock int
	// StartTimestamp anchors the simulated chain's clock: the first served block has
	// timestamp StartTimestamp + BlockTime, and each later block adds BlockTime. Set
	// this to the interop activation timestamp so the first block lands exactly one
	// block past activation (the earliest referenceable point).
	StartTimestamp uint64
	// FirstBlock is the height served for the anchor request (after=0), i.e. where
	// this chain offers to start history. Defaults to 1. Set it high to simulate a
	// long-running chain whose finalized head is far past genesis — the case that
	// forces the supernode to anchor rather than replay from block 1.
	FirstBlock uint64
	// Head, when non-zero, is the highest finalized block number: requests for a block
	// beyond it get "block": null, so tests can exercise the "nothing new yet" path.
	// Zero means the chain is unbounded (always has the next block).
	Head uint64
}

// Chain is a deterministic fake remote chain. The same (ChainID, block, log index)
// always maps to the same log hash and timestamp, so [Chain.ExpectedChecksum] reproduces
// exactly the checksum the LogsDB recomputes when validating a referencing executing
// message.
type Chain struct {
	cfg Config
}

// New builds a Chain, applying defaults for unset fields.
func New(cfg Config) *Chain {
	if cfg.BlockTime == 0 {
		cfg.BlockTime = defaultBlockTime
	}
	if cfg.MsgsPerBlock <= 0 {
		cfg.MsgsPerBlock = defaultMsgsPerBlock
	}
	if cfg.FirstBlock == 0 {
		cfg.FirstBlock = 1
	}
	return &Chain{cfg: cfg}
}

func (c *Chain) BlockTime() uint64 { return c.cfg.BlockTime }

// FirstBlock is the height this chain serves for the anchor request (after=0).
func (c *Chain) FirstBlock() uint64 { return c.cfg.FirstBlock }

// Handler returns an http.Handler implementing GET /finalized?after={N}. Wrap it in
// httptest.NewServer and point a remote.HTTPAdapter at the server's URL.
func (c *Chain) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/finalized", func(w http.ResponseWriter, r *http.Request) {
		after, err := strconv.ParseUint(r.URL.Query().Get("after"), 10, 64)
		if err != nil {
			http.Error(w, "invalid 'after' query param", http.StatusBadRequest)
			return
		}
		resp := struct {
			BlockTime uint64                 `json:"blockTime"`
			Block     *remote.FinalizedBlock `json:"block"`
		}{BlockTime: c.cfg.BlockTime}

		// after == 0 is the anchor request: answer with FirstBlock, wherever that is.
		// Every later request is strictly contiguous.
		n := after + 1
		if after == 0 {
			n = c.cfg.FirstBlock
		}
		if n >= c.cfg.FirstBlock && (c.cfg.Head == 0 || n <= c.cfg.Head) {
			resp.Block = c.Block(n)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	return mux
}

// Block returns the deterministic finalized block at the given height (>= 1).
func (c *Chain) Block(n uint64) *remote.FinalizedBlock {
	blk := &remote.FinalizedBlock{
		Number:     n,
		Hash:       c.blockHash(n),
		ParentHash: c.parentHash(n),
		Timestamp:  c.blockTimestamp(n),
		Messages:   make([]remote.InitiatingMessage, 0, c.cfg.MsgsPerBlock),
	}
	for i := 0; i < c.cfg.MsgsPerBlock; i++ {
		blk.Messages = append(blk.Messages, remote.InitiatingMessage{
			LogIndex: uint32(i),
			LogHash:  c.logHash(n, uint32(i)),
		})
	}
	return blk
}

// ExpectedChecksum returns the message checksum an executing message must carry to
// reference the dummy initiating message at (blockNum, logIdx). It mirrors exactly what
// raftwallogdb.DB.Contains recomputes from the sealed log hash, block timestamp and
// chain ID.
func (c *Chain) ExpectedChecksum(blockNum uint64, logIdx uint32) messages.MessageChecksum {
	return messages.ChecksumArgs{
		BlockNumber: blockNum,
		LogIndex:    logIdx,
		Timestamp:   c.blockTimestamp(blockNum),
		ChainID:     c.cfg.ChainID,
		LogHash:     c.logHash(blockNum, logIdx),
	}.Checksum()
}

// blockTimestamp places the first served block one block time past StartTimestamp,
// whatever height that block sits at, so an anchored chain's timestamps stay in the
// same relation to activation as a genesis-rooted one's.
func (c *Chain) blockTimestamp(n uint64) uint64 {
	return c.cfg.StartTimestamp + (n-c.cfg.FirstBlock+1)*c.cfg.BlockTime
}

func (c *Chain) blockHash(n uint64) common.Hash {
	return c.tag("mock-remote-block", n, 0)
}

// parentHash links block n to its parent: the prior block's hash, so the LogsDB
// parent-linkage invariants hold. At or below the first served block the parent is a
// synthetic, non-zero hash standing in for history this chain does not serve.
func (c *Chain) parentHash(n uint64) common.Hash {
	if n <= c.cfg.FirstBlock {
		return c.tag("mock-remote-genesis", 0, 0)
	}
	return c.blockHash(n - 1)
}

// logHash is the canonical content hash the remote node serves and the LogsDB stores.
// It is derived from a fabricated (payloadHash, origin) preimage so a real CrossL2Inbox
// executing-message log — which carries payloadHash and origin separately — can reference
// it: messages.DecodeExecutingMessageLog recomputes exactly this hash.
func (c *Chain) logHash(n uint64, logIdx uint32) common.Hash {
	return messages.PayloadHashToLogHash(c.PayloadHash(n, logIdx), c.Origin(n, logIdx))
}

// Origin is the deterministic originating-contract address for the message at
// (blockNum, logIdx). A referencing executing message must carry this address.
func (c *Chain) Origin(blockNum uint64, logIdx uint32) common.Address {
	return common.BytesToAddress(c.tag("mock-remote-origin", blockNum, uint64(logIdx)).Bytes())
}

// PayloadHash is the deterministic initiating-message payload hash for (blockNum, logIdx).
// A referencing executing message carries this as the CrossL2Inbox event's indexed topic.
func (c *Chain) PayloadHash(blockNum uint64, logIdx uint32) common.Hash {
	return c.tag("mock-remote-payload", blockNum, uint64(logIdx))
}

// tag derives a deterministic, domain-separated hash from the chain ID and two numbers,
// so distinct (domain, n, k) tuples never collide.
func (c *Chain) tag(domain string, n, k uint64) common.Hash {
	cid := c.cfg.ChainID.Bytes32()
	buf := make([]byte, 0, len(domain)+len(cid)+16)
	buf = append(buf, domain...)
	buf = append(buf, cid[:]...)
	buf = binary.BigEndian.AppendUint64(buf, n)
	buf = binary.BigEndian.AppendUint64(buf, k)
	return crypto.Keccak256Hash(buf)
}
