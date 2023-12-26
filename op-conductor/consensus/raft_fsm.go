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
	mtx        sync.RWMutex
	unsafeHead unsafeHeadData
}

// Apply implements raft.FSM, it applies the latest change (latest unsafe head payload) to FSM.
func (t *unsafeHeadTracker) Apply(l *raft.Log) interface{} {
	if l.Data == nil || len(l.Data) == 0 {
		return fmt.Errorf("log data is nil or empty")
	}
	var data unsafeHeadData
	if err := data.UnmarshalSSZ(bytes.NewReader(l.Data)); err != nil {
		return err
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()
	if t.unsafeHead.payload.BlockNumber < data.payload.BlockNumber {
		t.unsafeHead = data
	}

	return nil
}

// Restore implements raft.FSM, it restores state from snapshot.
func (t *unsafeHeadTracker) Restore(snapshot io.ReadCloser) error {
	var data unsafeHeadData
	if err := data.UnmarshalSSZ(snapshot); err != nil {
		return fmt.Errorf("error unmarshalling snapshot: %w", err)
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
		unsafeHead: t.unsafeHead,
	}, nil
}

// UnsafeHead returns the latest unsafe head payload.
func (t *unsafeHeadTracker) UnsafeHead() eth.ExecutionPayload {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.unsafeHead.payload
}

var _ raft.FSMSnapshot = (*snapshot)(nil)

type snapshot struct {
	log        log.Logger
	unsafeHead unsafeHeadData
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

// unsafeHeadData wraps the execution payload with the block version, and provides ease of use interfaces to marshal/unmarshal it.
type unsafeHeadData struct {
	version eth.BlockVersion
	payload eth.ExecutionPayload
}

func (e *unsafeHeadData) MarshalSSZ(w io.Writer) (int, error) {
	vb := byte(e.version)
	n1, err := w.Write([]byte{vb})
	if err != nil {
		return n1, err
	}

	n2, err := e.payload.MarshalSSZ(w)
	if err != nil {
		return n1 + n2, err
	}

	return n1 + n2, nil
}

func (e *unsafeHeadData) UnmarshalSSZ(r io.Reader) error {
	bs, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if len(bs) < 1 {
		return fmt.Errorf("data is too short to contain version information")
	}

	vb, data := bs[0], bs[1:]
	e.version = eth.BlockVersion(vb)
	if err = e.payload.UnmarshalSSZ(e.version, uint32(len(data)), bytes.NewReader(data)); err != nil {
		return err
	}
	return nil
}
