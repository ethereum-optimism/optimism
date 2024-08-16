package multithreaded

import (
	"errors"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

type ThreadedStackTracker interface {
	exec.TraceableStackTracker
	DropThread(threadId uint64)
}

type NoopThreadedStackTracker struct {
	exec.NoopStackTracker
}

var _ ThreadedStackTracker = (*ThreadedStackTrackerImpl)(nil)

func (n *NoopThreadedStackTracker) DropThread(threadId uint64) {}

type ThreadedStackTrackerImpl struct {
	meta               *program.Metadata
	state              *State
	trackersByThreadId map[uint64]exec.TraceableStackTracker
}

var _ ThreadedStackTracker = (*ThreadedStackTrackerImpl)(nil)

func NewThreadedStackTracker(state *State, meta *program.Metadata) (*ThreadedStackTrackerImpl, error) {
	if meta == nil {
		return nil, errors.New("metadata is nil")
	}

	return &ThreadedStackTrackerImpl{
		state:              state,
		meta:               meta,
		trackersByThreadId: make(map[uint64]exec.TraceableStackTracker),
	}, nil
}

func (t *ThreadedStackTrackerImpl) PushStack(caller uint64, target uint64) {
	t.getCurrentTracker().PushStack(caller, target)
}

func (t *ThreadedStackTrackerImpl) PopStack() {
	t.getCurrentTracker().PopStack()
}

func (t *ThreadedStackTrackerImpl) Traceback() {
	t.getCurrentTracker().Traceback()
}

func (t *ThreadedStackTrackerImpl) getCurrentTracker() exec.TraceableStackTracker {
	thread := t.state.getCurrentThread()
	tracker, exists := t.trackersByThreadId[thread.ThreadId]
	if !exists {
		tracker = exec.NewStackTrackerUnsafe(t.state, t.meta)
		t.trackersByThreadId[thread.ThreadId] = tracker
	}
	return tracker
}

func (t *ThreadedStackTrackerImpl) DropThread(threadId uint64) {
	delete(t.trackersByThreadId, threadId)
}
