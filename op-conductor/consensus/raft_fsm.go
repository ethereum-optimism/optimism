package consensus

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/hashicorp/raft"
	"github.com/klauspost/compress/s2"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ raft.FSM = (*unsafeHeadTracker)(nil)

// unsafeHeadTracker implements raft.FSM for storing unsafe head payload into raft consensus layer.
type unsafeHeadTracker struct {
	log        log.Logger
	metrics    ConsensusMetrics
	mtx        sync.RWMutex
	unsafeHead *eth.ExecutionPayloadEnvelope
}

func NewUnsafeHeadTracker(log log.Logger, metrics ConsensusMetrics) *unsafeHeadTracker {
	return &unsafeHeadTracker{
		log:     log,
		metrics: metrics,
	}
}

// Apply implements raft.FSM, it applies the latest change (latest unsafe head payload) to FSM.
func (t *unsafeHeadTracker) Apply(l *raft.Log) interface{} {
	start := time.Now()
	defer func() {
		if t.metrics != nil {
			t.metrics.RecordFSMApplyDuration(time.Since(start).Seconds())
		}
	}()

	if len(l.Data) == 0 {
		return fmt.Errorf("log data is nil or empty")
	}

	data, err := decodePayloadEnvelope(l.Data)
	if err != nil {
		return err
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.log.Debug("applying new unsafe head", "number", uint64(data.ExecutionPayload.BlockNumber), "hash", data.ExecutionPayload.BlockHash)
	if t.unsafeHead == nil || t.unsafeHead.ExecutionPayload.BlockNumber < data.ExecutionPayload.BlockNumber {
		t.unsafeHead = data
	}

	return nil
}

// Restore implements raft.FSM, it restores state from snapshot.
func (t *unsafeHeadTracker) Restore(snapshot io.ReadCloser) error {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, snapshot)
	snapshot.Close()
	if err != nil {
		return fmt.Errorf("error reading snapshot data: %w", err)
	}

	data, err := decodePayloadEnvelope(buf.Bytes())
	if err != nil {
		return err
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

// decodePayloadEnvelope decompresses (if needed) and SSZ-decodes a payload
// envelope. It tries BlockV4 first, falling back to BlockV3.
func decodePayloadEnvelope(raw []byte) (*eth.ExecutionPayloadEnvelope, error) {
	raw = maybeDecompress(raw)

	data := &eth.ExecutionPayloadEnvelope{}
	if err := data.UnmarshalSSZ(eth.BlockV4, uint32(len(raw)), bytes.NewReader(raw)); err != nil {
		if err := data.UnmarshalSSZ(eth.BlockV3, uint32(len(raw)), bytes.NewReader(raw)); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// maybeDecompress attempts S2/Snappy decompression. If the data is not
// S2/Snappy-encoded it is returned as-is. Raw SSZ data will never be
// valid S2 framing, so try/fallback is safe.
func maybeDecompress(data []byte) []byte {
	decoded, err := s2.Decode(nil, data)
	if err != nil {
		return data // not compressed, return original
	}
	return decoded
}
