package testutil

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

// ExpectedState is a test utility that basically stores a copy of a state that can be explicitly mutated
// to define an expected post-state.  The post-state is then validated with ExpectedState.Validate(t, postState)
type ExpectedState struct {
	PreimageKey         common.Hash
	PreimageOffset      arch.Word
	Heap                arch.Word
	LLReservationStatus multithreaded.LLReservationStatus
	LLAddress           arch.Word
	LLOwnerThread       arch.Word
	ExitCode            uint8
	Exited              bool
	Step                uint64
	LastHint            hexutil.Bytes
	MemoryRoot          common.Hash
	threadExpectations  *threadExpectations
	expectedMemory      *memory.Memory
	// Remember some actions so we can analyze expectations
	memoryWrites []arch.Word
}

type ExpectedThreadState struct {
	ThreadId  arch.Word
	ExitCode  uint8
	Exited    bool
	PC        arch.Word
	NextPC    arch.Word
	HI        arch.Word
	LO        arch.Word
	Registers [32]arch.Word
	Dropped   bool
}

func NewExpectedState(t require.TestingT, state mipsevm.FPVMState) *ExpectedState {
	fromState := ToMTState(t, state)

	return &ExpectedState{
		// General Fields
		PreimageKey:         fromState.GetPreimageKey(),
		PreimageOffset:      fromState.GetPreimageOffset(),
		Heap:                fromState.GetHeap(),
		LLReservationStatus: fromState.LLReservationStatus,
		LLAddress:           fromState.LLAddress,
		LLOwnerThread:       fromState.LLOwnerThread,
		ExitCode:            fromState.GetExitCode(),
		Exited:              fromState.GetExited(),
		Step:                fromState.GetStep(),
		LastHint:            fromState.GetLastHint(),
		MemoryRoot:          fromState.GetMemory().MerkleRoot(),
		threadExpectations:  newThreadExpectations(fromState),
		expectedMemory:      fromState.Memory.Copy(),
	}
}

func newExpectedThreadState(fromThread *multithreaded.ThreadState) *ExpectedThreadState {
	return &ExpectedThreadState{
		ThreadId:  fromThread.ThreadId,
		ExitCode:  fromThread.ExitCode,
		Exited:    fromThread.Exited,
		PC:        fromThread.Cpu.PC,
		NextPC:    fromThread.Cpu.NextPC,
		HI:        fromThread.Cpu.HI,
		LO:        fromThread.Cpu.LO,
		Registers: fromThread.Registers,
		Dropped:   false,
	}
}

func (e *ExpectedState) ExpectedMemoryWrites() []arch.Word {
	return e.memoryWrites
}

func (e *ExpectedState) ExpectStep() {
	// Set some standard expectations for a normal step
	e.Step += 1
	e.PrestateActiveThread().PC += 4
	e.PrestateActiveThread().NextPC += 4
	e.threadExpectations.StepsSinceLastContextSwitch += 1
}

func (e *ExpectedState) ExpectMemoryReservationCleared() {
	e.LLReservationStatus = multithreaded.LLStatusNone
	e.LLAddress = 0
	e.LLOwnerThread = 0
}

func (e *ExpectedState) ExpectMemoryWriteUint32(t require.TestingT, addr arch.Word, val uint32) {
	// Track write expectations
	e.memoryWrites = append(e.memoryWrites, addr)

	// Align address to 4-byte boundaries
	addr = addr & ^arch.Word(3)

	// Set 4 bytes at addr
	data := testutil.Uint32ToBytes(val)
	err := e.expectedMemory.SetMemoryRange(addr, bytes.NewReader(data))
	require.NoError(t, err)

	e.MemoryRoot = e.expectedMemory.MerkleRoot()
}

func (e *ExpectedState) ExpectMemoryWrite(addr arch.Word, val arch.Word) {
	// Track write expectations
	e.memoryWrites = append(e.memoryWrites, addr)

	e.expectedMemory.SetWord(addr, val)
	e.MemoryRoot = e.expectedMemory.MerkleRoot()
}

func (e *ExpectedState) ExpectPreemption() {
	e.threadExpectations.ExpectPreemption()
}

func (e *ExpectedState) ExpectNewThread() *ExpectedThreadState {
	return e.threadExpectations.ExpectNewThread()
}

func (e *ExpectedState) ExpectPoppedThread() {
	e.threadExpectations.ExpectPop()
}

func (e *ExpectedState) ExpectTraverseRight(traverseRight bool) {
	e.threadExpectations.ExpectTraverseRight(traverseRight)
}

func (e *ExpectedState) ExpectNoContextSwitch() {
	e.threadExpectations.StepsSinceLastContextSwitch += 1
}

func (e *ExpectedState) ExpectContextSwitch() {
	e.threadExpectations.StepsSinceLastContextSwitch = 0
}

func (e *ExpectedState) ActiveThread() *ExpectedThreadState {
	return e.threadExpectations.activeThread()
}

func (e *ExpectedState) ActiveThreadId() arch.Word {
	return e.threadExpectations.ActiveThreadId
}

func (e *ExpectedState) ExpectActiveThreadId(expected arch.Word) {
	e.threadExpectations.ActiveThreadId = expected
}

func (e *ExpectedState) ExpectNextThreadId(expected arch.Word) {
	e.threadExpectations.NextThreadId = expected
}

func (e *ExpectedState) PrestateActiveThread() *ExpectedThreadState {
	return e.threadExpectations.PrestateActiveThread()
}

func (e *ExpectedState) Thread(threadId arch.Word) *ExpectedThreadState {
	return e.threadExpectations.ThreadById(threadId)
}

func (e *ExpectedState) Validate(t require.TestingT, state mipsevm.FPVMState) {
	actualState := ToMTState(t, state)

	require.Equalf(t, e.PreimageKey, actualState.GetPreimageKey(), "Expect preimageKey = %v", e.PreimageKey)
	require.Equalf(t, e.PreimageOffset, actualState.GetPreimageOffset(), "Expect preimageOffset = %v", e.PreimageOffset)
	require.Equalf(t, e.Heap, actualState.GetHeap(), "Expect heap = 0x%x", e.Heap)
	require.Equalf(t, e.LLReservationStatus, actualState.LLReservationStatus, "Expect LLReservationStatus = %v", e.LLReservationStatus)
	require.Equalf(t, e.LLAddress, actualState.LLAddress, "Expect LLAddress = 0x%x", e.LLAddress)
	require.Equalf(t, e.LLOwnerThread, actualState.LLOwnerThread, "Expect LLOwnerThread = %v", e.LLOwnerThread)
	require.Equalf(t, e.ExitCode, actualState.GetExitCode(), "Expect exitCode = 0x%x", e.ExitCode)
	require.Equalf(t, e.Exited, actualState.GetExited(), "Expect exited = %v", e.Exited)
	require.Equalf(t, e.Step, actualState.GetStep(), "Expect step = %d", e.Step)
	require.Equalf(t, e.LastHint, actualState.GetLastHint(), "Expect lastHint = %v", e.LastHint)
	require.Equalf(t, e.MemoryRoot, common.Hash(actualState.GetMemory().MerkleRoot()), "Expect memory root = %v", e.MemoryRoot)
	// Thread-related global fields
	e.threadExpectations.Validate(t, actualState)
}

type threadExpectations struct {
	ActiveThreadId              arch.Word
	StepsSinceLastContextSwitch uint64
	NextThreadId                arch.Word
	prestateActiveThread        *ExpectedThreadState
	// Cache the original value of the prestate active thread, so we can keep the original values before any updates
	prestateActiveThreadValue ExpectedThreadState
	traverseRight             bool
	left                      []*ExpectedThreadState
	right                     []*ExpectedThreadState
	popped                    []*ExpectedThreadState
}

func newThreadExpectations(state *multithreaded.State) *threadExpectations {
	left := expectedThreadStack(state.LeftThreadStack)
	right := expectedThreadStack(state.RightThreadStack)
	var prestateActiveThread *ExpectedThreadState
	if state.TraverseRight {
		prestateActiveThread = right[len(right)-1]
	} else {
		prestateActiveThread = left[len(left)-1]
	}

	return &threadExpectations{
		ActiveThreadId:              prestateActiveThread.ThreadId,
		StepsSinceLastContextSwitch: state.StepsSinceLastContextSwitch,
		NextThreadId:                state.NextThreadId,
		prestateActiveThread:        prestateActiveThread,
		prestateActiveThreadValue:   *prestateActiveThread,
		traverseRight:               state.TraverseRight,
		left:                        left,
		right:                       right,
		popped:                      make([]*ExpectedThreadState, 0),
	}
}

func (e *threadExpectations) Validate(t require.TestingT, state *multithreaded.State) {
	actualState := ToMTState(t, state)

	require.Equalf(t, e.StepsSinceLastContextSwitch, actualState.StepsSinceLastContextSwitch, "Expect StepsSinceLastContextSwitch = %v", e.StepsSinceLastContextSwitch)
	require.Equalf(t, e.traverseRight, actualState.TraverseRight, "Expect TraverseRight = %v", e.traverseRight)
	require.Equalf(t, e.NextThreadId, actualState.NextThreadId, "Expect NextThreadId = %v", e.NextThreadId)
	require.Equalf(t, e.threadCount(), actualState.ThreadCount(), "Expect thread count = %v", e.threadCount())

	// Check active thread
	activeThreadId := actualState.GetCurrentThread().ThreadId
	require.Equal(t, e.ActiveThreadId, activeThreadId)

	// Check stacks
	e.assertStackMatchesExpectations(t, e.left, actualState.LeftThreadStack, "left", activeThreadId)
	e.assertStackMatchesExpectations(t, e.right, actualState.RightThreadStack, "right", activeThreadId)
}

func (e *threadExpectations) assertStackMatchesExpectations(t require.TestingT, expectedStack []*ExpectedThreadState, actualStack []*multithreaded.ThreadState, label string, activeThreadId arch.Word) {
	require.Equalf(t, len(expectedStack), len(actualStack), "Expect %v stack size = %v", label, len(expectedStack))
	for i, expectedThread := range expectedStack {
		if i >= len(actualStack) {
			// Break to avoid unit test panics - should be unreachable for actual tests
			require.FailNow(t, "Missing thread")
			break
		}
		actualThread := actualStack[i]
		e.validateThread(t, expectedThread, actualThread, activeThreadId)
	}
}

func (e *threadExpectations) validateThread(t require.TestingT, expected *ExpectedThreadState, actual *multithreaded.ThreadState, activeThreadId arch.Word) {
	threadInfo := fmt.Sprintf("tid = %v, active = %v", actual.ThreadId, actual.ThreadId == activeThreadId)
	require.Equalf(t, expected.ThreadId, actual.ThreadId, "Expect ThreadId = 0x%x (%v)", expected.ThreadId, threadInfo)
	require.Equalf(t, expected.PC, actual.Cpu.PC, "Expect PC = 0x%x (%v)", expected.PC, threadInfo)
	require.Equalf(t, expected.NextPC, actual.Cpu.NextPC, "Expect nextPC = 0x%x (%v)", expected.NextPC, threadInfo)
	require.Equalf(t, expected.HI, actual.Cpu.HI, "Expect HI = 0x%x (%v)", expected.HI, threadInfo)
	require.Equalf(t, expected.LO, actual.Cpu.LO, "Expect LO = 0x%x (%v)", expected.LO, threadInfo)
	require.Equalf(t, expected.Registers, actual.Registers, "Expect registers to match (%v)", threadInfo)
	require.Equalf(t, expected.ExitCode, actual.ExitCode, "Expect exitCode = %v (%v)", expected.ExitCode, threadInfo)
	require.Equalf(t, expected.Exited, actual.Exited, "Expect exited = %v (%v)", expected.Exited, threadInfo)
	require.Equalf(t, expected.Dropped, false, "Thread should not be dropped")
}

func (e *threadExpectations) ExpectPreemption() {
	e.StepsSinceLastContextSwitch = 0
	if e.traverseRight {
		lastEl := len(e.right) - 1
		preempted := e.right[lastEl]
		e.right = e.right[:lastEl]
		e.left = append(e.left, preempted)
		e.traverseRight = len(e.right) > 0
	} else {
		lastEl := len(e.left) - 1
		preempted := e.left[lastEl]
		e.left = e.left[:lastEl]
		e.right = append(e.right, preempted)
		e.traverseRight = len(e.left) == 0
	}
	e.updateActiveThreadId()
}

func (e *threadExpectations) ExpectNewThread() *ExpectedThreadState {
	e.StepsSinceLastContextSwitch = 0
	newThreadId := e.NextThreadId
	e.NextThreadId += 1

	// Copy expectations from prestate active thread's original value (before changing any expectations)
	newThread := &ExpectedThreadState{}
	*newThread = e.prestateActiveThreadValue

	newThread.ThreadId = newThreadId
	if e.traverseRight {
		e.right = append(e.right, newThread)
	} else {
		e.left = append(e.left, newThread)
	}

	e.ActiveThreadId = newThreadId
	return newThread
}

func (e *threadExpectations) ExpectPop() {
	e.StepsSinceLastContextSwitch = 0
	var popped *ExpectedThreadState
	if e.traverseRight {
		lastEl := len(e.right) - 1
		popped = e.right[lastEl]
		popped.Dropped = true
		e.right = e.right[:lastEl]
		e.traverseRight = len(e.right) > 0
	} else {
		lastEl := len(e.left) - 1
		popped = e.left[lastEl]
		popped.Dropped = true
		e.left = e.left[:lastEl]
		e.traverseRight = len(e.left) == 0
	}
	e.popped = append(e.popped, popped)

	e.updateActiveThreadId()
}

func (e *threadExpectations) ExpectTraverseRight(traverseRight bool) {
	e.traverseRight = traverseRight
}

func (e *threadExpectations) PrestateActiveThread() *ExpectedThreadState {
	return e.prestateActiveThread
}

func (e *threadExpectations) ThreadById(threadId arch.Word) *ExpectedThreadState {
	for _, thread := range e.allThreads() {
		if thread.ThreadId == threadId {
			return thread
		}
	}
	return nil
}

func (e *threadExpectations) allThreads() []*ExpectedThreadState {
	var allThreads []*ExpectedThreadState
	allThreads = append(allThreads, e.right...)
	allThreads = append(allThreads, e.left...)
	allThreads = append(allThreads, e.popped...)

	return allThreads
}

func (e *threadExpectations) updateActiveThreadId() {
	activeStack := e.activeStack()
	e.ActiveThreadId = activeStack[len(activeStack)-1].ThreadId
}

func (e *threadExpectations) threadCount() int {
	return e.leftStackSize() + e.rightStackSize()
}

func (e *threadExpectations) rightStackSize() int {
	return len(e.right)
}

func (e *threadExpectations) leftStackSize() int {
	return len(e.left)
}

func (e *threadExpectations) activeStack() []*ExpectedThreadState {
	if e.traverseRight {
		return e.right
	} else {
		return e.left
	}
}

func (e *threadExpectations) activeThread() *ExpectedThreadState {
	lastEl := len(e.activeStack()) - 1
	if lastEl < 0 {
		return nil
	}
	return e.activeStack()[lastEl]
}

func expectedThreadStack(threadStack []*multithreaded.ThreadState) []*ExpectedThreadState {
	expectedThreads := make([]*ExpectedThreadState, 0, len(threadStack))
	for _, threadState := range threadStack {
		expectedThreads = append(expectedThreads, newExpectedThreadState(threadState))
	}

	return expectedThreads
}
