package singlethreaded

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
)

func (m *InstrumentedState) handleSyscall() error {
	syscallNum, a0, a1, a2, _ := exec.GetSyscallArgs(&m.state.Registers)

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
	case exec.SysClone: // clone (not supported)
		v0 = 1
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
	}

	exec.HandleSyscallUpdates(&m.state.Cpu, &m.state.Registers, v0, v1)
	return nil
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1
	// instruction fetch
	insn, opcode, fun := exec.GetInstructionDetails(m.state.Cpu.PC, m.state.Memory)

	// Handle syscall separately
	// syscall (can read and write)
	if opcode == 0 && fun == 0xC {
		return m.handleSyscall()
	}

	// Exec the rest of the step logic
	return exec.ExecMipsCoreStepLogic(&m.state.Cpu, &m.state.Registers, m.state.Memory, insn, opcode, fun, m.memoryTracker, m.stackTracker)
}
