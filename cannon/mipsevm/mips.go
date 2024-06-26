package mipsevm

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func (m *InstrumentedState) readPreimage(key [32]byte, offset uint32) (dat [32]byte, datLen uint32) {
	preimage := m.lastPreimage
	if key != m.lastPreimageKey {
		m.lastPreimageKey = key
		data := m.preimageOracle.GetPreimage(key)
		// add the length prefix
		preimage = make([]byte, 0, 8+len(data))
		preimage = binary.BigEndian.AppendUint64(preimage, uint64(len(data)))
		preimage = append(preimage, data...)
		m.lastPreimage = preimage
	}
	m.lastPreimageOffset = offset
	datLen = uint32(copy(dat[:], preimage[offset:]))
	return
}

func (m *InstrumentedState) trackMemAccess(effAddr uint32) {
	if m.memProofEnabled && m.lastMemAccess != effAddr {
		if m.lastMemAccess != ^uint32(0) {
			panic(fmt.Errorf("unexpected different mem access at %08x, already have access at %08x buffered", effAddr, m.lastMemAccess))
		}
		m.lastMemAccess = effAddr
		m.memProof = m.state.Memory.MerkleProof(effAddr)
	}
}

func (m *InstrumentedState) handleSyscall() error {
	syscallNum, a0, a1, a2 := getSyscallArgs(&m.state.Registers)

	v0 := uint32(0)
	v1 := uint32(0)

	//fmt.Printf("syscall: %d\n", syscallNum)
	switch syscallNum {
	case sysMmap:
		var newHeap uint32
		v0, v1, newHeap = handleSysMmap(a0, a1, m.state.Heap)
		m.state.Heap = newHeap
	case sysBrk:
		v0 = 0x40000000
	case sysClone: // clone (not supported)
		v0 = 1
	case sysExitGroup:
		m.state.Exited = true
		m.state.ExitCode = uint8(a0)
		return nil
	case sysRead:
		var newPreimageOffset uint32
		v0, v1, newPreimageOffset = handleSysRead(a0, a1, a2, m.state.PreimageKey, m.state.PreimageOffset, m.readPreimage, m.state.Memory, m.trackMemAccess)
		m.state.PreimageOffset = newPreimageOffset
	case sysWrite:
		var newLastHint hexutil.Bytes
		var newPreimageKey common.Hash
		var newPreimageOffset uint32
		v0, v1, newLastHint, newPreimageKey, newPreimageOffset = handleSysWrite(a0, a1, a2, m.state.LastHint, m.state.PreimageKey, m.state.PreimageOffset, m.preimageOracle, m.state.Memory, m.trackMemAccess, m.stdOut, m.stdErr)
		m.state.LastHint = newLastHint
		m.state.PreimageKey = newPreimageKey
		m.state.PreimageOffset = newPreimageOffset
	case sysFcntl:
		v0, v1 = handleSysFcntl(a0, a1)
	}

	handleSyscallUpdates(&m.state.Cpu, &m.state.Registers, v0, v1)
	return nil
}

func (m *InstrumentedState) pushStack(target uint32) {
	if !m.debugEnabled {
		return
	}
	m.debug.stack = append(m.debug.stack, target)
	m.debug.caller = append(m.debug.caller, m.state.Cpu.PC)
}

func (m *InstrumentedState) popStack() {
	if !m.debugEnabled {
		return
	}
	if len(m.debug.stack) != 0 {
		fn := m.debug.meta.LookupSymbol(m.state.Cpu.PC)
		topFn := m.debug.meta.LookupSymbol(m.debug.stack[len(m.debug.stack)-1])
		if fn != topFn {
			// most likely the function was inlined. Snap back to the last return.
			i := len(m.debug.stack) - 1
			for ; i >= 0; i-- {
				if m.debug.meta.LookupSymbol(m.debug.stack[i]) == fn {
					m.debug.stack = m.debug.stack[:i]
					m.debug.caller = m.debug.caller[:i]
					break
				}
			}
		} else {
			m.debug.stack = m.debug.stack[:len(m.debug.stack)-1]
			m.debug.caller = m.debug.caller[:len(m.debug.caller)-1]
		}
	} else {
		fmt.Printf("ERROR: stack underflow at pc=%x. step=%d\n", m.state.Cpu.PC, m.state.Step)
	}
}

func (m *InstrumentedState) Traceback() {
	fmt.Printf("traceback at pc=%x. step=%d\n", m.state.Cpu.PC, m.state.Step)
	for i := len(m.debug.stack) - 1; i >= 0; i-- {
		s := m.debug.stack[i]
		idx := len(m.debug.stack) - i - 1
		fmt.Printf("\t%d %x in %s caller=%08x\n", idx, s, m.debug.meta.LookupSymbol(s), m.debug.caller[i])
	}
}

func (m *InstrumentedState) mipsStep() error {
	if m.state.Exited {
		return nil
	}
	m.state.Step += 1
	// instruction fetch
	insn := m.state.Memory.GetMemory(m.state.Cpu.PC)
	opcode := insn >> 26 // 6-bits

	// j-type j/jal
	if opcode == 2 || opcode == 3 {
		linkReg := uint32(0)
		if opcode == 3 {
			linkReg = 31
		}
		// Take top 4 bits of the next PC (its 256 MB region), and concatenate with the 26-bit offset
		target := (m.state.Cpu.NextPC & 0xF0000000) | ((insn & 0x03FFFFFF) << 2)
		m.pushStack(target)
		return handleJump(&m.state.Cpu, &m.state.Registers, linkReg, target)
	}

	// register fetch
	rs := uint32(0) // source register 1 value
	rt := uint32(0) // source register 2 / temp value
	rtReg := (insn >> 16) & 0x1F

	// R-type or I-type (stores rt)
	rs = m.state.Registers[(insn>>21)&0x1F]
	rdReg := rtReg
	if opcode == 0 || opcode == 0x1c {
		// R-type (stores rd)
		rt = m.state.Registers[rtReg]
		rdReg = (insn >> 11) & 0x1F
	} else if opcode < 0x20 {
		// rt is SignExtImm
		// don't sign extend for andi, ori, xori
		if opcode == 0xC || opcode == 0xD || opcode == 0xe {
			// ZeroExtImm
			rt = insn & 0xFFFF
		} else {
			// SignExtImm
			rt = signExtend(insn&0xFFFF, 16)
		}
	} else if opcode >= 0x28 || opcode == 0x22 || opcode == 0x26 {
		// store rt value with store
		rt = m.state.Registers[rtReg]

		// store actual rt with lwl and lwr
		rdReg = rtReg
	}

	if (opcode >= 4 && opcode < 8) || opcode == 1 {
		return handleBranch(&m.state.Cpu, &m.state.Registers, opcode, insn, rtReg, rs)
	}

	storeAddr := uint32(0xFF_FF_FF_FF)
	// memory fetch (all I-type)
	// we do the load for stores also
	mem := uint32(0)
	if opcode >= 0x20 {
		// M[R[rs]+SignExtImm]
		rs += signExtend(insn&0xFFFF, 16)
		addr := rs & 0xFFFFFFFC
		m.trackMemAccess(addr)
		mem = m.state.Memory.GetMemory(addr)
		if opcode >= 0x28 && opcode != 0x30 {
			// store
			storeAddr = addr
			// store opcodes don't write back to a register
			rdReg = 0
		}
	}

	// ALU
	val := executeMipsInstruction(insn, rs, rt, mem)

	fun := insn & 0x3f // 6-bits
	if opcode == 0 && fun >= 8 && fun < 0x1c {
		if fun == 8 || fun == 9 { // jr/jalr
			linkReg := uint32(0)
			if fun == 9 {
				linkReg = rdReg
			}
			m.popStack()
			return handleJump(&m.state.Cpu, &m.state.Registers, linkReg, rs)
		}

		if fun == 0xa { // movz
			return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, rs, rt == 0)
		}
		if fun == 0xb { // movn
			return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, rs, rt != 0)
		}

		// syscall (can read and write)
		if fun == 0xC {
			return m.handleSyscall()
		}

		// lo and hi registers
		// can write back
		if fun >= 0x10 && fun < 0x1c {
			return handleHiLo(&m.state.Cpu, &m.state.Registers, fun, rs, rt, rdReg)
		}
	}

	// stupid sc, write a 1 to rt
	if opcode == 0x38 && rtReg != 0 {
		m.state.Registers[rtReg] = 1
	}

	// write memory
	if storeAddr != 0xFF_FF_FF_FF {
		m.trackMemAccess(storeAddr)
		m.state.Memory.SetMemory(storeAddr, val)
	}

	// write back the value to destination register
	return handleRd(&m.state.Cpu, &m.state.Registers, rdReg, val, true)
}
