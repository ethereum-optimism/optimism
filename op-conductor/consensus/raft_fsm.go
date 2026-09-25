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

// unsafeHeadTracker implements raft.FSM for storing unsafe head payload into raft consensus layer.
type unsafeHeadTracker struct {
	log        log.Logger
	mtx        sync.RWMutex
	unsafeHead *eth.ExecutionPayloadEnvelope
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
		t.unsafeHead = data
	}

	return nil
}

// Restore implements raft.FSM, it restores state from snapshot.
// An empty snapshot means that no unsafe head had been committed when the snapshot was taken, see snapshot.Persist.
func (t *unsafeHeadTracker) Restore(snapshot io.ReadCloser) error {
	var buf bytes.Buffer
	n, err := io.Copy(&buf, snapshot)
	snapshot.Close()
	if err != nil {
		return fmt.Errorf("error reading snapshot data: %w", err)
	}

	if n == 0 {
		t.mtx.Lock()
		defer t.mtx.Unlock()
		t.log.Info("restoring empty snapshot, no unsafe head has been committed yet")
		t.unsafeHead = nil
		return nil
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
	return nil
}

// Snapshot implements raft.FSM, it creates a snapshot of the current state.
func (t *unsafeHeadTracker) Snapshot() (raft.FSMSnapshot, error) {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return &snapshot{
		log:        t.log,
		unsafeHead: t.unsafeHead,
	}, nil
}

// UnsafeHead returns the latest unsafe head payload.
func (t *unsafeHeadTracker) UnsafeHead() *eth.ExecutionPayloadEnvelope {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.unsafeHead
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	log        log.Logger
	unsafeHead *eth.ExecutionPayloadEnvelope
}

// Persist implements raft.FSMSnapshot, it writes the snapshot to the given sink.
// A nil unsafe head is written as an empty snapshot instead of dereferencing the nil payload. Raft decides to
// snapshot from log growth, not from FSM content, so a cluster that has not committed any unsafe head yet, e.g. a
// freshly bootstrapped cluster whose sequencers are still stopped, must be able to snapshot and compact its log
// too. Restore treats an empty snapshot as "no unsafe head".
func (s *snapshot) Persist(sink raft.SnapshotSink) error {
	if s.unsafeHead == nil {
		s.log.Warn("persisting empty snapshot, no unsafe head has been committed yet")
		return sink.Close()
	}

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
