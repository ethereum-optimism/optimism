package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
)

type ExpectationMutator func(t *testing.T, e *ExpectedState)

func TestValidate_shouldCatchMutations(t *testing.T) {
	states := []*multithreaded.State{
		randomStateWithMultipleThreads(0),
		randomStateWithMultipleThreads(1),
		randomStateWithMultipleThreads(2),
	}
	var emptyHash [32]byte
	someThread := RandomThread(123)

	cases := []struct {
		name string
		mut  ExpectationMutator
	}{
		{name: "PreimageKey", mut: func(t *testing.T, e *ExpectedState) { e.PreimageKey = emptyHash }},
		{name: "PreimageOffset", mut: func(t *testing.T, e *ExpectedState) { e.PreimageOffset += 1 }},
		{name: "Heap", mut: func(t *testing.T, e *ExpectedState) { e.Heap += 1 }},
		{name: "LLReservationStatus", mut: func(t *testing.T, e *ExpectedState) {
			e.LLReservationStatus = e.LLReservationStatus + 1
		}},
		{name: "LLAddress", mut: func(t *testing.T, e *ExpectedState) { e.LLAddress += 1 }},
		{name: "LLOwnerThread", mut: func(t *testing.T, e *ExpectedState) { e.LLOwnerThread += 1 }},
		{name: "ExitCode", mut: func(t *testing.T, e *ExpectedState) { e.ExitCode += 1 }},
		{name: "Exited", mut: func(t *testing.T, e *ExpectedState) { e.Exited = !e.Exited }},
		{name: "Step", mut: func(t *testing.T, e *ExpectedState) { e.Step += 1 }},
		{name: "LastHint", mut: func(t *testing.T, e *ExpectedState) { e.LastHint = []byte{7, 8, 9, 10} }},
		{name: "MemoryRoot", mut: func(t *testing.T, e *ExpectedState) { e.MemoryRoot = emptyHash }},
		{name: "StepsSinceLastContextSwitch", mut: func(t *testing.T, e *ExpectedState) {
			e.threadExpectations.StepsSinceLastContextSwitch += 1
		}},
		{name: "TraverseRight", mut: func(t *testing.T, e *ExpectedState) {
			e.threadExpectations.traverseRight = !e.threadExpectations.traverseRight
		}},
		{name: "NextThreadId", mut: func(t *testing.T, e *ExpectedState) { e.threadExpectations.NextThreadId += 1 }},
		{name: "ActiveThreadId", mut: func(t *testing.T, e *ExpectedState) {
			e.threadExpectations.ActiveThreadId += 1
		}},
		{name: "Empty thread expectations", mut: func(t *testing.T, e *ExpectedState) {
			e.threadExpectations.left = []*ExpectedThreadState{}
			e.threadExpectations.right = []*ExpectedThreadState{}
		}},
		{name: "Missing single thread expectation", mut: func(t *testing.T, e *ExpectedState) {
			if len(e.threadExpectations.left) > 0 {
				e.threadExpectations.left = e.threadExpectations.left[:len(e.threadExpectations.left)-1]
			} else {
				e.threadExpectations.right = e.threadExpectations.right[:len(e.threadExpectations.right)-1]
			}
		}},
		{name: "Extra thread expectation", mut: func(t *testing.T, e *ExpectedState) {
			e.threadExpectations.left = append(e.threadExpectations.left, newExpectedThreadState(someThread))
		}},
		{name: "Active threadId", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().ThreadId += 1
		}},
		{name: "Active thread exitCode", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().ExitCode += 1
		}},
		{name: "Active thread exited", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().Exited = !e.ActiveThread().Exited
		}},
		{name: "Active thread PC", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().PC += 1
		}},
		{name: "Active thread NextPC", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().NextPC += 1
		}},
		{name: "Active thread HI", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().HI += 1
		}},
		{name: "Active thread LO", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().LO += 1
		}},
		{name: "Active thread Registers", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().Registers[0] += 1
		}},
		{name: "Active thread dropped", mut: func(t *testing.T, e *ExpectedState) {
			e.ActiveThread().Dropped = true
		}},
		{name: "Inactive threadId", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).ThreadId += 1
		}},
		{name: "Inactive thread exitCode", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).ExitCode += 1
		}},
		{name: "Inactive thread exited", mut: func(t *testing.T, e *ExpectedState) {
			thread := findInactiveThread(t, e)
			thread.Exited = !thread.Exited
		}},
		{name: "Inactive thread PC", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).PC += 1
		}},
		{name: "Inactive thread NextPC", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).NextPC += 1
		}},
		{name: "Inactive thread HI", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).HI += 1
		}},
		{name: "Inactive thread LO", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).LO += 1
		}},
		{name: "Inactive thread Registers", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).Registers[0] += 1
		}},
		{name: "Inactive thread dropped", mut: func(t *testing.T, e *ExpectedState) {
			findInactiveThread(t, e).Dropped = true
		}},
	}
	for _, c := range cases {
		for i, state := range states {
			testName := fmt.Sprintf("%v (state #%v)", c.name, i)
			t.Run(testName, func(t *testing.T) {
				expected := NewExpectedState(t, state)
				c.mut(t, expected)

				// We should detect the change and fail
				mockT := &MockTestingT{}
				expected.Validate(mockT, state)
				mockT.RequireFailed(t)
			})
		}

	}
}

func TestValidate_shouldPassUnchangedExpectations(t *testing.T) {
	states := []*multithreaded.State{
		RandomState(10),
		RandomState(11),
		RandomState(12),
	}

	for i, state := range states {
		testName := fmt.Sprintf("State #%v", i)
		t.Run(testName, func(t *testing.T) {
			expected := NewExpectedState(t, state)

			mockT := &MockTestingT{}
			expected.Validate(mockT, state)
			mockT.RequireNoFailure(t)
		})
	}
}

func TestExpectNewThread_DoesNotInheritChangedExpectations(t *testing.T) {
	state := RandomState(123)
	expected := NewExpectedState(t, state)

	// Make some changes to the active thread
	origHI := expected.ActiveThread().HI
	expected.ActiveThread().HI = 123

	// Create a new thread
	newThread := expected.ExpectNewThread()

	// New thread should not carry over changes to the original thread
	require.Equal(t, origHI, newThread.HI)
}

func findInactiveThread(t *testing.T, e *ExpectedState) *ExpectedThreadState {
	threads := e.threadExpectations.allThreads()
	activeThread := e.ActiveThread()
	for _, thread := range threads {
		if thread.ThreadId != activeThread.ThreadId {
			return thread
		}
	}
	t.Error("No inactive thread found")
	t.FailNow()
	return nil
}

func randomStateWithMultipleThreads(seed int64) *multithreaded.State {
	state := RandomState(int(seed))
	if state.ThreadCount() == 1 {
		// Make sure we have at least 2 threads
		SetupThreads(seed+100, state, state.TraverseRight, 1, 1)
	}
	return state
}

type MockTestingT struct {
	errCount int
}

var _ require.TestingT = (*MockTestingT)(nil)

func (m *MockTestingT) Errorf(format string, args ...interface{}) {
	m.errCount += 1
}

func (m *MockTestingT) FailNow() {
	m.errCount += 1
}

func (m *MockTestingT) RequireFailed(t require.TestingT) {
	require.Greater(t, m.errCount, 0, "Should have tracked a failure")
}

func (m *MockTestingT) RequireNoFailure(t require.TestingT) {
	require.Equal(t, m.errCount, 0, "Should not have tracked a failure")
}
