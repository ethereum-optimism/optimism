package multithreaded

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

func (m *InstrumentedState) handleSyscall() error {
	thread := m.state.GetCurrentThread()

	syscallNum, a0, a1, a2, a3 := exec.GetSyscallArgs(m.state.GetRegistersRef())
	v0 := uint32(0)
	v1 := uint32(0)

	//fmt.Printf("syscall: %d\n", syscallNum)
	switch syscallNum {
	case exec.SysMmap:
		var newHeap uint32
		v0, v1, newHeap = exec.HandleSysMmap(a0, a1, m.state.Heap)
		m.state.Heap = newHeap
	case exec.SysBrk:
		v0 = program.PROGRAM_BREAK
	case exec.SysClone: // clone
		// a0 = flag bitmask, a1 = stack pointer
		if exec.ValidCloneFlags != a0 {
			m.state.Exited = true
			m.state.ExitCode = mipsevm.VMStatusPanic
			return nil
		}

		v0 = m.state.NextThreadId
		v1 = 0
		newThread := &ThreadState{
			ThreadId:         m.state.NextThreadId,
			ExitCode:         0,
			Exited:           false,
			FutexAddr:        exec.FutexEmptyAddr,
			FutexVal:         0,
			FutexTimeoutStep: 0,
			Cpu: mipsevm.CpuScalars{
				PC:     thread.Cpu.NextPC,
				NextPC: thread.Cpu.NextPC + 4,
				HI:     thread.Cpu.HI,
				LO:     thread.Cpu.LO,
			},
			Registers: thread.Registers,
		}

		newThread.Registers[29] = a1
		// the child will perceive a 0 value as returned value instead, and no error
		newThread.Registers[2] = 0
		newThread.Registers[7] = 0
		m.state.NextThreadId++

		// Preempt this thread for the new one. But not before updating PCs
		stackCaller := thread.Cpu.PC
		stackTarget := thread.Cpu.NextPC
		exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
		m.pushThread(newThread)
		// Note: We need to call stackTracker after pushThread
		// to ensure we are tracking in the context of the new thread
		m.stackTracker.PushStack(stackCaller, stackTarget)
		return nil
	case exec.SysExitGroup:
		m.state.Exited = true
		m.state.ExitCode = uint8(a0)
		return nil
	case exec.SysRead:
		var newPreimageOffset uint32
		var memUpdated bool
		var memAddr uint32
		v0, v1, newPreimageOffset, memUpdated, memAddr = exec.HandleSysRead(a0, a1, a2, m.state.PreimageKey, m.state.PreimageOffset, m.preimageOracle, m.state.Memory, m.memoryTracker)
		m.state.PreimageOffset = newPreimageOffset
		if memUpdated {
			m.handleMemoryUpdate(memAddr)
		}
	case exec.SysWrite:
		var newLastHint hexutil.Bytes
		var newPreimageKey common.Hash
		var newPreimageOffset uint32
		v0, v1, newLastHint, newPreimageKey, newPreimageOffset = exec.HandleSysWrite(a0, a1, a2, m.state.LastHint, m.state.PreimageKey, m.state.PreimageOffset, m.preimageOracle, m.state.Memory, m.memoryTracker, m.stdOut, m.stdErr)
		m.state.LastHint = newLastHint
		m.state.PreimageKey = newPreimageKey
		m.state.PreimageOffset = newPreimageOffset
	case exec.SysFcntl:
		v0, v1 = exec.HandleSysFcntl(a0, a1)
	case exec.SysGetTID:
		v0 = thread.ThreadId
		v1 = 0
	case exec.SysExit:
		thread.Exited = true
		thread.ExitCode = uint8(a0)
		if m.lastThreadRemaining() {
			m.state.Exited = true
			m.state.ExitCode = uint8(a0)
		}
		return nil
	case exec.SysFutex:
		// args: a0 = addr, a1 = op, a2 = val, a3 = timeout
		effAddr := a0 & 0xFFffFFfc
		switch a1 {
		case exec.FutexWaitPrivate:
			m.memoryTracker.TrackMemAccess(effAddr)
			mem := m.state.Memory.GetMemory(effAddr)
			if mem != a2 {
				v0 = exec.SysErrorSignal
				v1 = exec.MipsEAGAIN
			} else {
				thread.FutexAddr = effAddr
				thread.FutexVal = a2
				if a3 == 0 {
					thread.FutexTimeoutStep = exec.FutexNoTimeout
				} else {
					thread.FutexTimeoutStep = m.state.Step + exec.FutexTimeoutSteps
				}
				// Leave cpu scalars as-is. This instruction will be completed by `onWaitComplete`
				return nil
			}
		case exec.FutexWakePrivate:
			// Trigger thread traversal starting from the left stack until we find one waiting on the wakeup
			// address
			m.state.Wakeup = effAddr
			// Don't indicate to the program that we've woken up a waiting thread, as there are no guarantees.
			// The woken up thread should indicate this in userspace.
			v0 = 0
			v1 = 0
			exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
			m.preemptThread(thread)
			m.state.TraverseRight = len(m.state.LeftThreadStack) == 0
			return nil
		default:
			v0 = exec.SysErrorSignal
			v1 = exec.MipsEINVAL
		}
	case exec.SysSchedYield, exec.SysNanosleep:
		v0 = 0
		v1 = 0
		exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
		m.preemptThread(thread)
		return nil
	case exec.SysOpen:
		v0 = exec.SysErrorSignal
		v1 = exec.MipsEBADF
	case exec.SysClockGetTime:
		switch a0 {
		case exec.ClockGettimeRealtimeFlag, exec.ClockGettimeMonotonicFlag:
			v0, v1 = 0, 0
			var secs, nsecs uint32
			if a0 == exec.ClockGettimeMonotonicFlag {
				// monotonic clock_gettime is used by Go guest programs for goroutine scheduling and to implement
				// `time.Sleep` (and other sleep related operations).
				secs = uint32(m.state.Step / exec.HZ)
				nsecs = uint32((m.state.Step % exec.HZ) * (1_000_000_000 / exec.HZ))
			} // else realtime set to Unix Epoch

			effAddr := a1 & 0xFFffFFfc
			m.memoryTracker.TrackMemAccess(effAddr)
			m.state.Memory.SetMemory(effAddr, secs)
			m.handleMemoryUpdate(effAddr)
			m.memoryTracker.TrackMemAccess2(effAddr + 4)
			m.state.Memory.SetMemory(effAddr+4, nsecs)
			m.handleMemoryUpdate(effAddr + 4)
		default:
			v0 = exec.SysErrorSignal
			v1 = exec.MipsEINVAL
		}
	case exec.SysGetpid:
		v0 = 0
		v1 = 0
	case exec.SysMunmap:
	case exec.SysGetAffinity:
	case exec.SysMadvise:
	case exec.SysRtSigprocmask:
	case exec.SysSigaltstack:
	case exec.SysRtSigaction:
	case exec.SysPrlimit64:
	case exec.SysClose:
	case exec.SysPread64:
	case exec.SysFstat64:
	case exec.SysOpenAt:
	case exec.SysReadlink:
	case exec.SysReadlinkAt:
	case exec.SysIoctl:
	case exec.SysEpollCreate1:
	case exec.SysPipe2:
	case exec.SysEpollCtl:
	case exec.SysEpollPwait:
	case exec.SysGetRandom:
	case exec.SysUname:
	case exec.SysStat64:
	case exec.SysGetuid:
	case exec.SysGetgid:
	case exec.SysLlseek:
	case exec.SysMinCore:
	case exec.SysTgkill:
	case exec.SysSetITimer:
	case exec.SysTimerCreate:
	case exec.SysTimerSetTime:
	case exec.SysTimerDelete:
	default:
		m.Traceback()
		panic(fmt.Sprintf("unrecognized syscall: %d", syscallNum))
	}

	exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
	return nil
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1
	thread := m.state.GetCurrentThread()

	// During wakeup traversal, search for the first thread blocked on the wakeup address.
	// Don't allow regular execution until we have found such a thread or else we have visited all threads.
	if m.state.Wakeup != exec.FutexEmptyAddr {
		// We are currently performing a wakeup traversal
		if m.state.Wakeup == thread.FutexAddr {
			// We found a target thread, resume normal execution and process this thread
			m.state.Wakeup = exec.FutexEmptyAddr
		} else {
			// This is not the thread we're looking for, move on
			traversingRight := m.state.TraverseRight
			changedDirections := m.preemptThread(thread)
			if traversingRight && changedDirections {
				// We started the wakeup traversal walking left and we've now walked all the way right
				// We have therefore visited all threads and can resume normal thread execution
				m.state.Wakeup = exec.FutexEmptyAddr
			}
		}
		return nil
	}

	if thread.Exited {
		m.popThread()
		m.stackTracker.DropThread(thread.ThreadId)
		return nil
	}

	// check if thread is blocked on a futex
	if thread.FutexAddr != exec.FutexEmptyAddr {
		// if set, then check futex
		// check timeout first
		if m.state.Step > thread.FutexTimeoutStep {
			// timeout! Allow execution
			m.onWaitComplete(thread, true)
			return nil
		} else {
			effAddr := thread.FutexAddr & 0xFFffFFfc
			m.memoryTracker.TrackMemAccess(effAddr)
			mem := m.state.Memory.GetMemory(effAddr)
			if thread.FutexVal == mem {
				// still got expected value, continue sleeping, try next thread.
				m.preemptThread(thread)
				return nil
			} else {
				// wake thread up, the value at its address changed!
				// Userspace can turn thread back to sleep if it was too sporadic.
				m.onWaitComplete(thread, false)
				return nil
			}
		}
	}

	if m.state.StepsSinceLastContextSwitch >= exec.SchedQuantum {
		// Force a context switch as this thread has been active too long
		if m.state.ThreadCount() > 1 {
			// Log if we're hitting our context switch limit - only matters if we have > 1 thread
			if m.log.Enabled(context.Background(), log.LevelTrace) {
				msg := fmt.Sprintf("Thread has reached maximum execution steps (%v) - preempting.", exec.SchedQuantum)
				m.log.Trace(msg, "threadId", thread.ThreadId, "threadCount", m.state.ThreadCount(), "pc", thread.Cpu.PC)
			}
		}
		m.preemptThread(thread)
		return nil
	}
	m.state.StepsSinceLastContextSwitch += 1

	//instruction fetch
	insn, opcode, fun := exec.GetInstructionDetails(m.state.GetPC(), m.state.Memory)

	// Handle syscall separately
	// syscall (can read and write)
	if opcode == 0 && fun == 0xC {
		return m.handleSyscall()
	}

	// Handle RMW (read-modify-write) ops
	if opcode == exec.OpLoadLinked || opcode == exec.OpStoreConditional {
		return m.handleRMWOps(insn, opcode)
	}

	// Exec the rest of the step logic
	memUpdated, memAddr, err := exec.ExecMipsCoreStepLogic(m.state.getCpuRef(), m.state.GetRegistersRef(), m.state.Memory, insn, opcode, fun, m.memoryTracker, m.stackTracker)
	if err != nil {
		return err
	}
	if memUpdated {
		m.handleMemoryUpdate(memAddr)
	}

	return nil
}

func (m *InstrumentedState) handleMemoryUpdate(memAddr uint32) {
	if memAddr == m.state.LLAddress {
		// Reserved address was modified, clear the reservation
		m.clearLLMemoryReservation()
	}
}

func (m *InstrumentedState) clearLLMemoryReservation() {
	m.state.LLReservationActive = false
	m.state.LLAddress = 0
	m.state.LLOwnerThread = 0
}

// handleRMWOps handles LL and SC operations which provide the primitives to implement read-modify-write operations
func (m *InstrumentedState) handleRMWOps(insn, opcode uint32) error {
	baseReg := (insn >> 21) & 0x1F
	base := m.state.GetRegistersRef()[baseReg]
	rtReg := (insn >> 16) & 0x1F
	offset := exec.SignExtendImmediate(insn)

	effAddr := (base + offset) & 0xFFFFFFFC
	m.memoryTracker.TrackMemAccess(effAddr)
	mem := m.state.Memory.GetMemory(effAddr)

	var retVal uint32
	threadId := m.state.GetCurrentThread().ThreadId
	if opcode == exec.OpLoadLinked {
		retVal = mem
		m.state.LLReservationActive = true
		m.state.LLAddress = effAddr
		m.state.LLOwnerThread = threadId
	} else if opcode == exec.OpStoreConditional {
		// Check if our memory reservation is still intact
		if m.state.LLReservationActive && m.state.LLOwnerThread == threadId && m.state.LLAddress == effAddr {
			// Complete atomic update: set memory and return 1 for success
			m.clearLLMemoryReservation()
			rt := m.state.GetRegistersRef()[rtReg]
			m.state.Memory.SetMemory(effAddr, rt)
			retVal = 1
		} else {
			// Atomic update failed, return 0 for failure
			retVal = 0
		}
	} else {
		panic(fmt.Sprintf("Invalid instruction passed to handleRMWOps (opcode %08x)", opcode))
	}

	return exec.HandleRd(m.state.getCpuRef(), m.state.GetRegistersRef(), rtReg, retVal, true)
}

func (m *InstrumentedState) onWaitComplete(thread *ThreadState, isTimedOut bool) {
	// Note: no need to reset m.state.Wakeup.  If we're here, the Wakeup field has already been reset
	// Clear the futex state
	thread.FutexAddr = exec.FutexEmptyAddr
	thread.FutexVal = 0
	thread.FutexTimeoutStep = 0

	// Complete the FUTEX_WAIT syscall
	v0 := uint32(0)
	v1 := uint32(0)
	if isTimedOut {
		v0 = exec.SysErrorSignal
		v1 = exec.MipsETIMEDOUT
	}
	exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
}

func (m *InstrumentedState) preemptThread(thread *ThreadState) bool {
	// Pop thread from the current stack and push to the other stack
	if m.state.TraverseRight {
		rtThreadCnt := len(m.state.RightThreadStack)
		if rtThreadCnt == 0 {
			panic("empty right thread stack")
		}
		m.state.RightThreadStack = m.state.RightThreadStack[:rtThreadCnt-1]
		m.state.LeftThreadStack = append(m.state.LeftThreadStack, thread)
	} else {
		lftThreadCnt := len(m.state.LeftThreadStack)
		if lftThreadCnt == 0 {
			panic("empty left thread stack")
		}
		m.state.LeftThreadStack = m.state.LeftThreadStack[:lftThreadCnt-1]
		m.state.RightThreadStack = append(m.state.RightThreadStack, thread)
	}

	changeDirections := false
	current := m.state.getActiveThreadStack()
	if len(current) == 0 {
		m.state.TraverseRight = !m.state.TraverseRight
		changeDirections = true
	}

	m.state.StepsSinceLastContextSwitch = 0
	return changeDirections
}

func (m *InstrumentedState) pushThread(thread *ThreadState) {
	if m.state.TraverseRight {
		m.state.RightThreadStack = append(m.state.RightThreadStack, thread)
	} else {
		m.state.LeftThreadStack = append(m.state.LeftThreadStack, thread)
	}
	m.state.StepsSinceLastContextSwitch = 0
}

func (m *InstrumentedState) popThread() {
	if m.state.TraverseRight {
		m.state.RightThreadStack = m.state.RightThreadStack[:len(m.state.RightThreadStack)-1]
	} else {
		m.state.LeftThreadStack = m.state.LeftThreadStack[:len(m.state.LeftThreadStack)-1]
	}

	current := m.state.getActiveThreadStack()
	if len(current) == 0 {
		m.state.TraverseRight = !m.state.TraverseRight
	}
	m.state.StepsSinceLastContextSwitch = 0
}

func (m *InstrumentedState) lastThreadRemaining() bool {
	return m.state.ThreadCount() == 1
}
