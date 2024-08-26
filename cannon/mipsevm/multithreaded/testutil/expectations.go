package testutil

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
)

type ExpectedMTState struct {
	PreimageKey    common.Hash
	PreimageOffset uint32
	Heap           uint32
	ExitCode       uint8
	Exited         bool
	Step           uint64
	LastHint       hexutil.Bytes
	MemoryRoot     common.Hash
	// Threading-related expectations
	StepsSinceLastContextSwitch uint64
	Wakeup                      uint32
	TraverseRight               bool
	NextThreadId                uint32
	ThreadCount                 int
	RightStackSize              int
	LeftStackSize               int
	prestateActiveThreadId      uint32
	prestateActiveThreadOrig    ExpectedThreadState // Cached for internal use
	ActiveThreadId              uint32
	threadExpectations          map[uint32]*ExpectedThreadState
}

type ExpectedThreadState struct {
	ThreadId         uint32
	ExitCode         uint8
	Exited           bool
	FutexAddr        uint32
	FutexVal         uint32
	FutexTimeoutStep uint64
	PC               uint32
	NextPC           uint32
	HI               uint32
	LO               uint32
	Registers        [32]uint32
	Dropped          bool
}

func NewExpectedMTState(fromState *multithreaded.State) *ExpectedMTState {
	currentThread := fromState.GetCurrentThread()

	expectedThreads := make(map[uint32]*ExpectedThreadState)
	for _, t := range GetAllThreads(fromState) {
		expectedThreads[t.ThreadId] = newExpectedThreadState(t)
	}

	return &ExpectedMTState{
		// General Fields
		PreimageKey:    fromState.GetPreimageKey(),
		PreimageOffset: fromState.GetPreimageOffset(),
		Heap:           fromState.GetHeap(),
		ExitCode:       fromState.GetExitCode(),
		Exited:         fromState.GetExited(),
		Step:           fromState.GetStep(),
		LastHint:       fromState.GetLastHint(),
		MemoryRoot:     fromState.GetMemory().MerkleRoot(),
		// Thread-related global fields
		StepsSinceLastContextSwitch: fromState.StepsSinceLastContextSwitch,
		Wakeup:                      fromState.Wakeup,
		TraverseRight:               fromState.TraverseRight,
		NextThreadId:                fromState.NextThreadId,
		ThreadCount:                 fromState.ThreadCount(),
		RightStackSize:              len(fromState.RightThreadStack),
		LeftStackSize:               len(fromState.LeftThreadStack),
		// ThreadState expectations
		prestateActiveThreadId:   currentThread.ThreadId,
		prestateActiveThreadOrig: *newExpectedThreadState(currentThread), // Cache prestate thread for internal use
		ActiveThreadId:           currentThread.ThreadId,
		threadExpectations:       expectedThreads,
	}
}

func newExpectedThreadState(fromThread *multithreaded.ThreadState) *ExpectedThreadState {
	return &ExpectedThreadState{
		ThreadId:         fromThread.ThreadId,
		ExitCode:         fromThread.ExitCode,
		Exited:           fromThread.Exited,
		FutexAddr:        fromThread.FutexAddr,
		FutexVal:         fromThread.FutexVal,
		FutexTimeoutStep: fromThread.FutexTimeoutStep,
		PC:               fromThread.Cpu.PC,
		NextPC:           fromThread.Cpu.NextPC,
		HI:               fromThread.Cpu.HI,
		LO:               fromThread.Cpu.LO,
		Registers:        fromThread.Registers,
		Dropped:          false,
	}
}

func (e *ExpectedMTState) ExpectStep() {
	// Set some standard expectations for a normal step
	e.Step += 1
	e.PrestateActiveThread().PC += 4
	e.PrestateActiveThread().NextPC += 4
	e.StepsSinceLastContextSwitch += 1
}

func (e *ExpectedMTState) ExpectPreemption(preState *multithreaded.State) {
	e.ActiveThreadId = FindNextThread(preState).ThreadId
	e.StepsSinceLastContextSwitch = 0
	if preState.TraverseRight {
		e.TraverseRight = e.RightStackSize > 1
		e.RightStackSize -= 1
		e.LeftStackSize += 1
	} else {
		e.TraverseRight = e.LeftStackSize == 1
		e.LeftStackSize -= 1
		e.RightStackSize += 1
	}
}

func (e *ExpectedMTState) ExpectNewThread() *ExpectedThreadState {
	newThreadId := e.NextThreadId
	e.NextThreadId += 1
	e.ThreadCount += 1

	// Clone expectations from prestate active thread's original state (bf changing any expectations)
	newThread := &ExpectedThreadState{}
	*newThread = e.prestateActiveThreadOrig

	newThread.ThreadId = newThreadId
	e.threadExpectations[newThreadId] = newThread

	return newThread
}

func (e *ExpectedMTState) ActiveThread() *ExpectedThreadState {
	return e.threadExpectations[e.ActiveThreadId]
}

func (e *ExpectedMTState) PrestateActiveThread() *ExpectedThreadState {
	return e.threadExpectations[e.prestateActiveThreadId]
}

func (e *ExpectedMTState) Thread(threadId uint32) *ExpectedThreadState {
	return e.threadExpectations[threadId]
}

func (e *ExpectedMTState) Validate(t require.TestingT, actualState *multithreaded.State) {
	require.Equal(t, e.PreimageKey, actualState.GetPreimageKey(), fmt.Sprintf("Expect preimageKey = %v", e.PreimageKey))
	require.Equal(t, e.PreimageOffset, actualState.GetPreimageOffset(), fmt.Sprintf("Expect preimageOffset = %v", e.PreimageOffset))
	require.Equal(t, e.Heap, actualState.GetHeap(), fmt.Sprintf("Expect heap = 0x%x", e.Heap))
	require.Equal(t, e.ExitCode, actualState.GetExitCode(), fmt.Sprintf("Expect exitCode = 0x%x", e.ExitCode))
	require.Equal(t, e.Exited, actualState.GetExited(), fmt.Sprintf("Expect exited = %v", e.Exited))
	require.Equal(t, e.Step, actualState.GetStep(), fmt.Sprintf("Expect step = %d", e.Step))
	require.Equal(t, e.LastHint, actualState.GetLastHint(), fmt.Sprintf("Expect lastHint = %v", e.LastHint))
	require.Equal(t, e.MemoryRoot, common.Hash(actualState.GetMemory().MerkleRoot()), fmt.Sprintf("Expect memory root = %v", e.MemoryRoot))
	// Thread-related global fields
	require.Equal(t, e.StepsSinceLastContextSwitch, actualState.StepsSinceLastContextSwitch, fmt.Sprintf("Expect StepsSinceLastContextSwitch = %v", e.StepsSinceLastContextSwitch))
	require.Equal(t, e.Wakeup, actualState.Wakeup, fmt.Sprintf("Expect Wakeup = %v", e.Wakeup))
	require.Equal(t, e.TraverseRight, actualState.TraverseRight, fmt.Sprintf("Expect TraverseRight = %v", e.TraverseRight))
	require.Equal(t, e.NextThreadId, actualState.NextThreadId, fmt.Sprintf("Expect NextThreadId = %v", e.NextThreadId))
	require.Equal(t, e.ThreadCount, actualState.ThreadCount(), fmt.Sprintf("Expect thread count = %v", e.ThreadCount))
	require.Equal(t, e.RightStackSize, len(actualState.RightThreadStack), fmt.Sprintf("Expect right stack size = %v", e.RightStackSize))
	require.Equal(t, e.LeftStackSize, len(actualState.LeftThreadStack), fmt.Sprintf("Expect right stack size = %v", e.LeftStackSize))

	// Check active thread
	activeThread := actualState.GetCurrentThread()
	require.Equal(t, e.ActiveThreadId, activeThread.ThreadId)
	// Check all threads
	expectedThreadCount := 0
	for tid, exp := range e.threadExpectations {
		actualThread := FindThread(actualState, tid)
		isActive := tid == activeThread.ThreadId
		if exp.Dropped {
			require.Nil(t, actualThread, "Thread %v should have been dropped", tid)
		} else {
			require.NotNil(t, actualThread, "Could not find thread matching expected thread with id %v", tid)
			e.validateThread(t, exp, actualThread, isActive)
			expectedThreadCount++
		}
	}
	require.Equal(t, expectedThreadCount, actualState.ThreadCount(), "Thread expectations do not match thread count")
}

func (e *ExpectedMTState) validateThread(t require.TestingT, et *ExpectedThreadState, actual *multithreaded.ThreadState, isActive bool) {
	threadInfo := fmt.Sprintf("tid = %v, active = %v", actual.ThreadId, isActive)
	require.Equal(t, et.ThreadId, actual.ThreadId, fmt.Sprintf("Expect ThreadId = 0x%x (%v)", et.ThreadId, threadInfo))
	require.Equal(t, et.PC, actual.Cpu.PC, fmt.Sprintf("Expect PC = 0x%x (%v)", et.PC, threadInfo))
	require.Equal(t, et.NextPC, actual.Cpu.NextPC, fmt.Sprintf("Expect nextPC = 0x%x (%v)", et.NextPC, threadInfo))
	require.Equal(t, et.HI, actual.Cpu.HI, fmt.Sprintf("Expect HI = 0x%x (%v)", et.HI, threadInfo))
	require.Equal(t, et.LO, actual.Cpu.LO, fmt.Sprintf("Expect LO = 0x%x (%v)", et.LO, threadInfo))
	require.Equal(t, et.Registers, actual.Registers, fmt.Sprintf("Expect registers to match (%v)", threadInfo))
	require.Equal(t, et.ExitCode, actual.ExitCode, fmt.Sprintf("Expect exitCode = %v (%v)", et.ExitCode, threadInfo))
	require.Equal(t, et.Exited, actual.Exited, fmt.Sprintf("Expect exited = %v (%v)", et.Exited, threadInfo))
	require.Equal(t, et.FutexAddr, actual.FutexAddr, fmt.Sprintf("Expect futexAddr = %v (%v)", et.FutexAddr, threadInfo))
	require.Equal(t, et.FutexVal, actual.FutexVal, fmt.Sprintf("Expect futexVal = %v (%v)", et.FutexVal, threadInfo))
	require.Equal(t, et.FutexTimeoutStep, actual.FutexTimeoutStep, fmt.Sprintf("Expect futexTimeoutStep = %v (%v)", et.FutexTimeoutStep, threadInfo))
}
