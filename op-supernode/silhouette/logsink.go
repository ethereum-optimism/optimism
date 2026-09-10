package silhouette

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// LogStore is the write side of one chain's interop log database, narrowed to what a sink that
// seals its own blocks needs. The supernode's LogsDB satisfies it as-is, structurally — neither
// package imports the other, which is what keeps the wire side of Silhouette independent of the
// supernode's internals (G2 D1).
//
// The narrowing is deliberate and not merely tidy. This interface has no read path for logs and no
// Contains: a sink WRITES the export set and never consults it. Everything that reads the database
// — the cross-safety judge, the frontier view — goes through the supernode's own LogsDB handle.
type LogStore interface {
	// LatestSealedBlock returns the latest sealed block ID, or false if no blocks are sealed.
	LatestSealedBlock() (eth.BlockID, bool)
	// FindSealedBlock returns the seal recorded at the given block number.
	FindSealedBlock(number uint64) (messages.BlockSeal, error)
	// AddLog stages one of a block's logs. Indices must start at 0 and be contiguous; the block is
	// written by the following SealBlock.
	AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *messages.ExecutingMessage) error
	// SealBlock writes the staged logs under the given block.
	SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error
	// Rewind removes every block after newHead. A target below the store's first block clears it.
	Rewind(newHead eth.BlockID) error
}

// maxBlockLogIndex bounds the log index a block export may claim. The LogsDB indexes a block's
// logs by position, so an exported index is materialised as that many slots (see exportedLogHashes);
// without a bound, one absurd index in one envelope would be an allocation of that size. It sits
// well above any block a real gas limit can produce — a bare LOG0 costs 375 gas — so it only ever
// fires on a malformed batch.
const maxBlockLogIndex = 1 << 17

// unexportedLogHash occupies the LogsDB slot of a log the export policy did not include.
//
// The LogsDB identifies a log by its POSITION within a block, while wire v2 names each exported
// log's own block-level index (proofbatch.LogExport.Index). Those two agree only when every log is
// exported. Under a curated policy they do not, and the gaps have to be occupied by something for
// the exported logs to land on their real indices — which is the whole point of carrying the index
// explicitly, since an executing message references this chain's REAL log indices.
//
// That something must be a value no real log can hash to. This is a domain-separated keccak of a
// fixed string, and the security argument is second-preimage resistance, stated plainly because it
// is the only thing holding:
//
//	a real log hash is messages.LogToLogHash(l) = keccak(addr ‖ keccak(payload)), a hash of 52
//	bytes. To make an executing message that references a poisoned slot, an attacker must present
//	an (address, payload) pair to CrossL2Inbox whose LogToLogHash equals this constant — a second
//	preimage of keccak. Publishing the constant therefore costs nothing: the checksum an executing
//	message carries is derived by the inbox FROM the payload the executor supplies, so an attacker
//	who cannot produce the payload cannot produce the message, whatever they know about the hash.
//
// This is load-bearing for the export policy itself, not merely for tidiness. Filling gaps with
// real-looking hashes would let a P insider reference logs P chose NOT to export, and filling them
// with predictable preimage-bearing filler would make them forgeable initiating messages — the two
// failure modes that make positional receipts ingestion unusable here (PLAN's LogsDB rule).
//
// Under the v2 default policy every log is exported and no slot is ever filled with it.
var unexportedLogHash = crypto.Keccak256Hash([]byte("silhouette:unexported-log"))

// ErrForeignHistory is returned when the store already holds a different block at a height this
// node's proofs commit to. It is not a recoverable condition: the store was built under a
// different anchor or a different chain.
var ErrForeignHistory = errors.New("log store holds foreign history")

// LogSink feeds one silhouette chain's exported logs into its interop log database, straight from
// the wire.
//
// This is the ingestion half of "a proven chain is a chain whose execution client is a verifier":
// the chain's initiating messages come out of the accepted proof batch, not out of receipts
// produced by re-executing it. Three properties are the design, and each is asserted by a test:
//
//   - EXPLICIT INDICES. Every log lands on the block-level index the wire names, with poison in the
//     gaps, so an executing message referencing P's real log index finds P's real log.
//   - REAL HASHES. A block is sealed under its real L2 block hash off the wire, so a proof-carried
//     chain's LogsDB is keyed exactly like a driven chain's, and no re-anchoring concept is needed
//     anywhere downstream.
//   - SYNCHRONOUS WITH ACCEPTANCE. Sealing happens inside batch acceptance, before the proven head
//     moves, so the head can never claim history the database does not hold. There is no polling
//     ingester, no queue and no cursor to disagree with — which is how the Cove-era C7 wedge class
//     (a served-but-unsealed block stranding a cursor forever) died. It stays dead by having no
//     component that could reintroduce it: retry-safety here is idempotent replay (see
//     sealProvenBlock), not a hand-off protocol.
type LogSink struct {
	log   log.Logger
	store LogStore
	mu    sync.Mutex
}

// NewLogSink builds a sink over a log store.
func NewLogSink(logger log.Logger, store LogStore) *LogSink {
	return &LogSink{log: logger, store: store}
}

// Accept seals an accepted batch's blocks. parentHash is the real hash of the block the batch
// extends — the proven-or-forced head — which the first sealed block chains onto.
//
// A batch is one transaction and one proof, so it lands whole or not at all: anything this call
// wrote is taken back before the error is returned, and the pipeline derives it again.
func (s *LogSink) Accept(blocks []proofbatch.BlockExport, parentHash common.Hash) error {
	if s == nil || s.store == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(blocks) == 0 {
		return nil
	}
	parent := parentHash
	wrote := false
	for _, blk := range blocks {
		logs, err := exportedLogHashes(blk)
		if err != nil {
			s.unwind(blocks, parentHash, wrote)
			return err
		}
		sealed, err := s.sealProvenBlock(blk, logs, parent)
		wrote = wrote || sealed
		if err != nil {
			s.unwind(blocks, parentHash, wrote)
			return err
		}
		parent = blk.Hash
	}
	return nil
}

// Rewind takes back the blocks a reset dropped, so the store never holds messages that nothing on
// L1 proves any more.
//
// This is the mirror of the fact store's own L1 rewind (G2 D5): an accepted batch whose carrier is
// no longer canonical was never posted, so both the facts it proved AND the messages it made
// referenceable have to go, together. Rewinding only the facts would leave a chain's exported logs
// referenceable by their real indices with no proof behind them.
func (s *LogSink) Rewind(target eth.BlockID) error {
	if s == nil || s.store == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rewind(target)
}

func (s *LogSink) rewind(target eth.BlockID) error {
	latest, ok := s.store.LatestSealedBlock()
	if !ok || latest.Number <= target.Number {
		return nil
	}
	s.log.Warn("unsealing proven blocks dropped by an L1 reorg", "from", latest, "to", target)
	if err := s.store.Rewind(target); err != nil {
		return fmt.Errorf("rewind log store to %s: %w", target, err)
	}
	return nil
}

// ReplaceWithEmpty mirrors the stock receipts ingester after Holocene invalidation: discard the
// invalid proof-carried suffix and seal the deposits-only replacement as a block with no exported
// logs. The replacement's imports are likewise empty and are reported by Container.
func (s *LogSink) ReplaceWithEmpty(parent eth.BlockID, block eth.BlockID, timestamp uint64) error {
	if s == nil || s.store == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if block.Number != parent.Number+1 {
		return fmt.Errorf("replacement block %s does not immediately follow parent %s", block, parent)
	}
	if latest, ok := s.store.LatestSealedBlock(); ok && latest.Number >= block.Number {
		seal, err := s.store.FindSealedBlock(block.Number)
		if err != nil {
			return fmt.Errorf("read replacement seal at block %d: %w", block.Number, err)
		}
		if seal.Hash == block.Hash {
			return nil
		}
	}
	if err := s.rewind(parent); err != nil {
		return err
	}
	if latest, ok := s.store.LatestSealedBlock(); ok && latest.Number == parent.Number && latest.Hash != parent.Hash {
		return fmt.Errorf("replacement parent %d is sealed as %s, expected %s", parent.Number, latest.Hash, parent.Hash)
	}
	if err := s.store.SealBlock(parent.Hash, block, timestamp); err != nil {
		return fmt.Errorf("seal empty replacement block %s: %w", block, err)
	}
	return nil
}

// unwind takes back whatever a failed Accept left behind.
//
// If the call sealed any block of the batch, the batch's parent is the target: half a batch is not
// a thing this node ever believed. If it sealed nothing, the store may still be holding logs staged
// for a block that was never sealed, and rewinding to its own head is how those are dropped —
// notably WITHOUT touching history this node did not write, which is what keeps a disagreement with
// a foreign store an error rather than a wipe.
func (s *LogSink) unwind(blocks []proofbatch.BlockExport, parentHash common.Hash, wrote bool) {
	target := eth.BlockID{Hash: parentHash, Number: blocks[0].Number - 1}
	if !wrote {
		latest, ok := s.store.LatestSealedBlock()
		if !ok {
			latest = eth.BlockID{}
		}
		target = latest
	}
	if err := s.store.Rewind(target); err != nil {
		s.log.Error("could not unwind a rejected proof batch; the store may hold staged logs",
			"target", target, "err", err)
	}
}

// sealProvenBlock writes one proven block into the log store, or checks it against what is already
// there. It reports whether it wrote.
//
// Proven history is re-derived from the configured L1 start block on every start rather than
// persisted, so a restarted node walks back over blocks the store already holds. Those are not
// re-sealed — they are COMPARED, which is what makes the replay a real re-check rather than a skip,
// and what makes re-serving a block idempotent instead of an error. A disagreement means the store
// was built under a different anchor or a different chain, and the honest response is to refuse
// rather than to write history that does not chain.
func (s *LogSink) sealProvenBlock(blk proofbatch.BlockExport, logs []common.Hash, parentHash common.Hash) (bool, error) {
	if latest, ok := s.store.LatestSealedBlock(); ok && blk.Number <= latest.Number {
		seal, err := s.store.FindSealedBlock(blk.Number)
		if err != nil {
			return false, fmt.Errorf("read sealed block %d while replaying proven history: %w", blk.Number, err)
		}
		if seal.Hash != blk.Hash {
			return false, fmt.Errorf("%w: block %d is sealed as %s but this node's proofs commit to %s "+
				"(a store built under a different anchor cannot chain: clear this chain's LogsDB "+
				"when moving the anchor block number)",
				ErrForeignHistory, blk.Number, seal.Hash, blk.Hash)
		}
		return false, nil
	}
	parentBlock := eth.BlockID{Hash: parentHash, Number: blk.Number - 1}
	for idx, logHash := range logs {
		// execMsg is nil for every log, and from wire v3 on that is a statement about POSITION, not
		// about existence.
		//
		// The LogsDB files a block's executing messages against the CrossL2Inbox log that carried
		// each one — position IS the key. Wire v3 carries a block's import list as an unordered SET
		// with no positions at all (PLAN G7, minimal-leak: no executing-tx information of any kind),
		// so there is no index to file them under, and inventing one would put a fabricated position
		// into the same-timestamp dependency graph. The import list therefore travels through the
		// FACT STORE (Fact.ExecMsgs) and the judge reads it from there, while this sink stays what it
		// was: the writer of the EXPORT set. See G7G D1.
		//
		// What has NOT changed is the export half: a proven chain's sealed logs are its initiating
		// messages, keyed exactly like a driven chain's, with poison in the gaps.
		if err := s.store.AddLog(logHash, parentBlock, uint32(idx), nil); err != nil {
			return false, fmt.Errorf("add log %d of block %d: %w", idx, blk.Number, err)
		}
	}
	if err := s.store.SealBlock(parentHash, eth.BlockID{Hash: blk.Hash, Number: blk.Number}, blk.Timestamp); err != nil {
		return false, fmt.Errorf("seal block %d: %w", blk.Number, err)
	}
	return true, nil
}

// exportedLogHashes turns a block's exported logs into the contiguous, position-indexed hash list
// the LogsDB seals, poisoning the gaps a curated export policy leaves.
//
// A v2 export names each log's own block-level index, which is what lets a curated policy export a
// subset of a block's logs without the survivors' indices shifting. The LogsDB, though, addresses a
// block's logs by position — it is the same store a driven chain fills from receipts, where
// position IS the index — so the gaps have to be materialised as slots no executing message can
// match. Under the default policy there are no gaps and this is a straight copy.
func exportedLogHashes(blk proofbatch.BlockExport) ([]common.Hash, error) {
	if len(blk.Logs) == 0 {
		return nil, nil
	}
	// CheckStructure has already required ascending, strictly increasing indices, so the last one
	// is the highest.
	highest := blk.Logs[len(blk.Logs)-1].Index
	if highest > maxBlockLogIndex {
		return nil, fmt.Errorf("block %d exports log index %d, above the %d this node will index",
			blk.Number, highest, maxBlockLogIndex)
	}
	logs := make([]common.Hash, highest+1)
	for i := range logs {
		logs[i] = unexportedLogHash
	}
	for _, l := range blk.Logs {
		logs[l.Index] = l.Hash
	}
	return logs, nil
}
