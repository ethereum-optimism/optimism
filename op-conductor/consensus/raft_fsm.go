package consensus

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ raft.FSM = (*unsafeHeadTracker)(nil)

const (
	// maxRetainedUnsafePayloads bounds the payload history every member keeps for bringing its
	// own op-node up to the consensus unsafe head. A leadership handover only ever leaves a node
	// a few blocks behind; a larger gap is a real sync problem that leader transfer, not payload
	// replay, has to resolve.
	maxRetainedUnsafePayloads = 16
	// maxRetainedUnsafePayloadsMemory bounds that same history by size, because a payload's
	// footprint is dominated by its transactions: 16 calldata-heavy blocks on a high gas limit
	// chain would otherwise be tens of megabytes of steady-state RSS in a sidecar. op-node bounds
	// its own unsafe payload buffer the same way.
	maxRetainedUnsafePayloadsMemory = 64 * 1024 * 1024
	// retainedPayloadFixedCost approximates everything in a payload other than its transaction
	// calldata, and retainedPayloadTxOverhead the per-transaction slice header. Both mirror
	// op-node's payloadMemSize.
	retainedPayloadFixedCost  = 1000
	retainedPayloadTxOverhead = 24
)

// unsafeHeadTracker implements raft.FSM for storing unsafe head payload into raft consensus layer.
type unsafeHeadTracker struct {
	log        log.Logger
	mtx        sync.RWMutex
	unsafeHead *eth.ExecutionPayloadEnvelope

	// retained holds the most recently applied payloads, each the parent of the next, ending at
	// unsafeHead. Every member applies every committed payload, so this gives each conductor
	// the blocks its own op-node is missing after a handover without waiting for p2p gossip
	// or L1 derivation. It is a local cache, not replicated state: it is left out of
	// snapshots, and a restarted or restored member simply starts with no history.
	retained []*eth.ExecutionPayloadEnvelope
	// retainedSize is the approximate memory footprint of retained, see retainedPayloadSize.
	retainedSize uint64
}

func NewUnsafeHeadTracker(log log.Logger) *unsafeHeadTracker {
	return &unsafeHeadTracker{
		log: log,
	}
}

// Apply implements raft.FSM, it applies the latest change (latest unsafe head payload) to FSM.
func (t *unsafeHeadTracker) Apply(l *raft.Log) interface{} {
	if len(l.Data) == 0 {
		return fmt.Errorf("log data is nil or empty")
	}

	data := &eth.ExecutionPayloadEnvelope{}
	// There is no good way to know which version, so try both. Start with the most recent version
	if err := data.UnmarshalSSZ(eth.BlockV4, uint32(len(l.Data)), bytes.NewReader(l.Data)); err != nil {
		// Try v3 if v4 fails and return an error if v3 fails
		if err := data.UnmarshalSSZ(eth.BlockV3, uint32(len(l.Data)), bytes.NewReader(l.Data)); err != nil {
			return err
		}
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.log.Debug("applying new unsafe head", "number", uint64(data.ExecutionPayload.BlockNumber), "hash", data.ExecutionPayload.BlockHash.Hex())
	if t.unsafeHead == nil || t.unsafeHead.ExecutionPayload.BlockNumber < data.ExecutionPayload.BlockNumber {
		t.retain(data)
		t.unsafeHead = data
	}

	return nil
}

// Restore implements raft.FSM, it restores state from snapshot.
func (t *unsafeHeadTracker) Restore(snapshot io.ReadCloser) error {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, snapshot)
	snapshot.Close()
	if err != nil {
		return fmt.Errorf("error reading snapshot data: %w", err)
	}

	data := &eth.ExecutionPayloadEnvelope{}
	// There is no good way to know which version, so try both. Start with the most recent version
	if err := data.UnmarshalSSZ(eth.BlockV4, uint32(n), bytes.NewReader(buf.Bytes())); err != nil {
		// Try v3 if v4 fails and return an error if v3 fails
		if err := data.UnmarshalSSZ(eth.BlockV3, uint32(n), bytes.NewReader(buf.Bytes())); err != nil {
			return err
		}
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.unsafeHead = data
	clear(t.retained)
	t.retained = t.retained[:0]
	t.retainedSize = 0
	return nil
}

// Snapshot implements raft.FSM, it creates a snapshot of the current state.
func (t *unsafeHeadTracker) Snapshot() (raft.FSMSnapshot, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return &snapshot{
		unsafeHead: t.unsafeHead,
	}, nil
}

// UnsafeHead returns the latest unsafe head payload.
func (t *unsafeHeadTracker) UnsafeHead() *eth.ExecutionPayloadEnvelope {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.unsafeHead
}

// retain appends payload to the gap-fill history. A payload that is not a direct child of the
// last retained one (after a restart, a restore, or a competing block at the same height being
// dropped by Apply) discards the history rather than leaving a hole, or a fork, that a replay
// could not fill.
//
// Callers must hold t.mtx.
func (t *unsafeHeadTracker) retain(payload *eth.ExecutionPayloadEnvelope) {
	last := len(t.retained) - 1
	if last < 0 || !isChildOf(t.retained[last], payload) {
		clear(t.retained)
		t.retained = append(t.retained[:0], payload)
		t.retainedSize = retainedPayloadSize(payload)
		return
	}

	t.retained = append(t.retained, payload)
	t.retainedSize += retainedPayloadSize(payload)
	// Drop from the front until the history is back inside both bounds, keeping the newest
	// payload whatever its size: it is the one a handover is most likely to need.
	for len(t.retained) > 1 && (len(t.retained) > maxRetainedUnsafePayloads || t.retainedSize > maxRetainedUnsafePayloadsMemory) {
		t.retainedSize -= retainedPayloadSize(t.retained[0])
		n := copy(t.retained, t.retained[1:])
		t.retained[n] = nil
		t.retained = t.retained[:n]
	}
}

func isChildOf(parent, child *eth.ExecutionPayloadEnvelope) bool {
	return parent.ExecutionPayload.BlockNumber+1 == child.ExecutionPayload.BlockNumber &&
		parent.ExecutionPayload.BlockHash == child.ExecutionPayload.ParentHash
}

// retainedPayloadSize approximates a payload's memory footprint, which is dominated by its
// transaction calldata. Mirrors op-node's payloadMemSize.
func retainedPayloadSize(payload *eth.ExecutionPayloadEnvelope) uint64 {
	size := uint64(retainedPayloadFixedCost)
	for _, tx := range payload.ExecutionPayload.Transactions {
		size += uint64(len(tx)) + retainedPayloadTxOverhead
	}
	return size
}

// UnsafePayloadsAfter returns the retained payloads with a block number greater than number, each
// the parent of the next up to the latest unsafe payload, or nil when the history no longer
// reaches back that far.
func (t *unsafeHeadTracker) UnsafePayloadsAfter(number uint64) []*eth.ExecutionPayloadEnvelope {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	for i, payload := range t.retained {
		if uint64(payload.ExecutionPayload.BlockNumber) == number+1 {
			// Copy: Apply mutates the backing array in place.
			return append([]*eth.ExecutionPayloadEnvelope(nil), t.retained[i:]...)
		}
	}
	return nil
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	log        log.Logger
	unsafeHead *eth.ExecutionPayloadEnvelope
}

// Persist implements raft.FSMSnapshot, it writes the snapshot to the given sink.
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := s.unsafeHead.MarshalSSZ(sink); err != nil {
		if cerr := sink.Cancel(); cerr != nil {
			s.log.Error("error cancelling snapshot sink", "error", cerr)
		}
		return fmt.Errorf("error writing data to sink: %w", err)
	}

	return sink.Close()
}

// Release implements raft.FSMSnapshot.
// We don't really need to do anything within Release as the snapshot is not gonna change after creation, and we don't hold any reference to closable resources.
func (s *snapshot) Release() {}
