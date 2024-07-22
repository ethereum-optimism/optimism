package multithreaded

import (
	"errors"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

type ThreadedStackTracker struct {
	meta               *program.Metadata
	state              *State
	trackersByThreadId map[uint32]exec.TraceableStackTracker
}

var _ exec.TraceableStackTracker = (*ThreadedStackTracker)(nil)

func NewThreadedStackTracker(state *State, meta *program.Metadata) (*ThreadedStackTracker, error) {
	if meta == nil {
		return nil, errors.New("metadata is nil")
	}

	return &ThreadedStackTracker{
		state:              state,
		meta:               meta,
		trackersByThreadId: make(map[uint32]exec.TraceableStackTracker),
	}, nil
}

func (t *ThreadedStackTracker) PushStack(target uint32) {
	t.getCurrentTracker().PushStack(target)
}

func (t *ThreadedStackTracker) PopStack() {
	t.getCurrentTracker().PopStack()
}

func (t *ThreadedStackTracker) Traceback() {
	t.getCurrentTracker().Traceback()
}

func (t *ThreadedStackTracker) getCurrentTracker() exec.TraceableStackTracker {
	thread := t.state.getCurrentThread()
	tracker, exists := t.trackersByThreadId[thread.ThreadId]
	if !exists {
		tracker = exec.NewStackTrackerUnsafe(t.state, t.meta)
		t.trackersByThreadId[thread.ThreadId] = tracker
	}
	return tracker
}

func (t *ThreadedStackTracker) DropThread(threadId uint32) {
	delete(t.trackersByThreadId, threadId)
}
