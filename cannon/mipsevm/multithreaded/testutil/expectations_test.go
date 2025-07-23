package testutil

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
)

type ExpectationMutator func(e *ExpectedState, st *multithreaded.State)

func TestValidate_shouldCatchMutations(t *testing.T) {
	states := []*multithreaded.State{
		RandomState(0),
		RandomState(1),
		RandomState(2),
	}
	var emptyHash [32]byte
	someThread := RandomThread(123)

	cases := []struct {
		name string
		mut  ExpectationMutator
	}{
		{name: "PreimageKey", mut: func(e *ExpectedState, st *multithreaded.State) { e.PreimageKey = emptyHash }},
		{name: "PreimageOffset", mut: func(e *ExpectedState, st *multithreaded.State) { e.PreimageOffset += 1 }},
		{name: "Heap", mut: func(e *ExpectedState, st *multithreaded.State) { e.Heap += 1 }},
		{name: "LLReservationStatus", mut: func(e *ExpectedState, st *multithreaded.State) { e.LLReservationStatus = e.LLReservationStatus + 1 }},
		{name: "LLAddress", mut: func(e *ExpectedState, st *multithreaded.State) { e.LLAddress += 1 }},
		{name: "LLOwnerThread", mut: func(e *ExpectedState, st *multithreaded.State) { e.LLOwnerThread += 1 }},
		{name: "ExitCode", mut: func(e *ExpectedState, st *multithreaded.State) { e.ExitCode += 1 }},
		{name: "Exited", mut: func(e *ExpectedState, st *multithreaded.State) { e.Exited = !e.Exited }},
		{name: "Step", mut: func(e *ExpectedState, st *multithreaded.State) { e.Step += 1 }},
		{name: "LastHint", mut: func(e *ExpectedState, st *multithreaded.State) { e.LastHint = []byte{7, 8, 9, 10} }},
		{name: "MemoryRoot", mut: func(e *ExpectedState, st *multithreaded.State) { e.MemoryRoot = emptyHash }},
		{name: "StepsSinceLastContextSwitch", mut: func(e *ExpectedState, st *multithreaded.State) { e.threadExpectations.StepsSinceLastContextSwitch += 1 }},
		{name: "TraverseRight", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.traverseRight = !e.threadExpectations.traverseRight
		}},
		{name: "NextThreadId", mut: func(e *ExpectedState, st *multithreaded.State) { e.threadExpectations.NextThreadId += 1 }},
		{name: "ActiveThreadId", mut: func(e *ExpectedState, st *multithreaded.State) { e.threadExpectations.ActiveThreadId += 1 }},
		{name: "Empty thread expectations", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.left = []*ExpectedThreadState{}
			e.threadExpectations.right = []*ExpectedThreadState{}
		}},
		{name: "Missing single thread expectation", mut: func(e *ExpectedState, st *multithreaded.State) {
			if len(e.threadExpectations.left) > 0 {
				e.threadExpectations.left = e.threadExpectations.left[:len(e.threadExpectations.left)-1]
			} else {
				e.threadExpectations.right = e.threadExpectations.right[:len(e.threadExpectations.right)-1]
			}
		}},
		{name: "Extra thread expectation", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.left = append(e.threadExpectations.left, newExpectedThreadState(someThread))
		}},
		{name: "Active threadId", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.ThreadId += 1
		}},
		{name: "Active thread exitCode", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.ExitCode += 1
		}},
		{name: "Active thread exited", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.Exited = !st.GetCurrentThread().Exited
		}},
		{name: "Active thread PC", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.PC += 1
		}},
		{name: "Active thread NextPC", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.NextPC += 1
		}},
		{name: "Active thread HI", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.HI += 1
		}},
		{name: "Active thread LO", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.LO += 1
		}},
		{name: "Active thread Registers", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.Registers[0] += 1
		}},
		{name: "Active thread dropped", mut: func(e *ExpectedState, st *multithreaded.State) {
			e.threadExpectations.prestateActiveThread.Dropped = true
		}},
		{name: "Inactive threadId", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).ThreadId += 1
		}},
		{name: "Inactive thread exitCode", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).ExitCode += 1
		}},
		{name: "Inactive thread exited", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).Exited = !FindNextThread(st).Exited
		}},
		{name: "Inactive thread PC", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).PC += 1
		}},
		{name: "Inactive thread NextPC", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).NextPC += 1
		}},
		{name: "Inactive thread HI", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).HI += 1
		}},
		{name: "Inactive thread LO", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).LO += 1
		}},
		{name: "Inactive thread Registers", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).Registers[0] += 1
		}},
		{name: "Inactive thread dropped", mut: func(e *ExpectedState, st *multithreaded.State) {
			findInactiveThread(e).Dropped = true
		}},
	}
	for _, c := range cases {
		for i, state := range states {
			testName := fmt.Sprintf("%v (state #%v)", c.name, i)
			t.Run(testName, func(t *testing.T) {
				expected := NewExpectedState(t, state)
				c.mut(expected, state)

				// We should detect the change and fail
				mockT := &MockTestingT{}
				expected.Validate(mockT, state)
				mockT.RequireFailed(t)
			})
		}

	}
}

func findInactiveThread(e *ExpectedState) *ExpectedThreadState {
	threads := e.threadExpectations.allThreads()
	idx := int(len(threads) / 2)
	return threads[idx]
}

func TestValidate_shouldPassUnchangedExpectations(t *testing.T) {
	states := []*multithreaded.State{
		RandomState(0),
		RandomState(1),
		RandomState(2),
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
