package multithreaded

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
)

func (m *InstrumentedState) handleSyscall() error {
	thread := m.state.getCurrentThread()

	syscallNum, a0, a1, a2, a3 := exec.GetSyscallArgs(m.state.GetRegisters())
	v0 := uint32(0)
	v1 := uint32(0)

	//fmt.Printf("syscall: %d\n", syscallNum)
	switch syscallNum {
	case exec.SysMmap:
		var newHeap uint32
		v0, v1, newHeap = exec.HandleSysMmap(a0, a1, m.state.Heap)
		m.state.Heap = newHeap
	case exec.SysBrk:
		v0 = exec.BrkStart
	case exec.SysClone: // clone
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
		}
		for i := 0; i < 32; i++ {
			newThread.Registers[i] = thread.Registers[i]
		}
		newThread.Registers[29] = a1
		// the child will perceive a 0 value as returned value instead, and no error
		newThread.Registers[2] = 0
		newThread.Registers[7] = 0
		m.state.NextThreadId++

		// Preempt this thread for the new one. But not before updating PCs
		exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
		m.pushThread(newThread)
		return nil
	case exec.SysExitGroup:
		m.state.Exited = true
		m.state.ExitCode = uint8(a0)
		return nil
	case exec.SysRead:
		var newPreimageOffset uint32
		v0, v1, newPreimageOffset = exec.HandleSysRead(a0, a1, a2, m.state.PreimageKey, m.state.PreimageOffset, m.preimageOracle, m.state.Memory, m.memoryTracker)
		m.state.PreimageOffset = newPreimageOffset
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
		return nil
	case exec.SysFutex:
		// args: a0 = addr, a1 = op, a2 = val, a3 = timeout
		switch a1 {
		case exec.FutexWaitPrivate:
			thread.FutexAddr = a0
			m.memoryTracker.TrackMemAccess(a0)
			mem := m.state.Memory.GetMemory(a0)
			if mem != a2 {
				v0 = exec.SysErrorSignal
				v1 = exec.MipsEAGAIN
			} else {
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
			m.state.Wakeup = a0
			// Don't indicate to the program that we've woken up a waiting thread, as there are no guarantees.
			// The woken up thread should indicate this in userspace.
			v0 = 0
			v1 = 0
			exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
			m.preemptThread(thread, true)
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
		m.preemptThread(thread, true)
		return nil
	case exec.SysOpen:
		v0 = exec.SysErrorSignal
		v1 = exec.MipsEBADF

	}

	exec.HandleSyscallUpdates(&thread.Cpu, &thread.Registers, v0, v1)
	return nil
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1

	// If we've completed traversing both stacks for the wakeup check
	if m.state.Wakeup != exec.FutexEmptyAddr && m.state.TraverseRight && len(m.state.RightThreadStack) == 0 {
		m.state.TraverseRight = false
		m.state.Wakeup = exec.FutexEmptyAddr
		return nil
	}

	thread := m.state.getCurrentThread()
	if thread.Exited {
		m.popThread()
		return nil
	}

	// Search for the first thread blocked by the wakeup call, if wakeup is set
	// Don't allow regular execution until we resolved if we have woken up any thread.
	if m.state.Wakeup != exec.FutexEmptyAddr && m.state.Wakeup != thread.FutexAddr {
		m.preemptThread(thread, !m.state.TraverseRight)
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
			m.memoryTracker.TrackMemAccess(thread.FutexAddr)
			mem := m.state.Memory.GetMemory(thread.FutexAddr)
			if thread.FutexVal == mem {
				// still got expected value, continue sleeping, try next thread.
				m.preemptThread(thread, true)
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
		msg := fmt.Sprintf("Thread has reached maximum execution steps (%v) - preempting.", exec.SchedQuantum)
		m.log.Info(msg, "threadId", thread.ThreadId, "threadCount", m.state.threadCount())
		m.preemptThread(thread, true)
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

	// Exec the rest of the step logic
	return exec.ExecMipsCoreStepLogic(m.state.getCpu(), m.state.GetRegisters(), m.state.Memory, insn, opcode, fun, m.memoryTracker, m.stackTracker)
}

func (m *InstrumentedState) onWaitComplete(thread *ThreadState, isTimedOut bool) {
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

	// Clear wakeup signal
	m.state.Wakeup = exec.FutexEmptyAddr
}

func (m *InstrumentedState) preemptThread(thread *ThreadState, doUpdateDirection bool) {
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

	current := m.state.getActiveThreadStack()
	if doUpdateDirection && len(current) == 0 {
		m.state.TraverseRight = !m.state.TraverseRight
	}
	m.state.StepsSinceLastContextSwitch = 0
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
