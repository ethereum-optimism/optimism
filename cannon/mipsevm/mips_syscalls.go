package mipsevm

import (
	"encoding/binary"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	sysMmap      = 4090
	sysBrk       = 4045
	sysClone     = 4120
	sysExitGroup = 4246
	sysRead      = 4003
	sysWrite     = 4004
	sysFcntl     = 4055
)

const (
	fdStdin         = 0
	fdStdout        = 1
	fdStderr        = 2
	fdHintRead      = 3
	fdHintWrite     = 4
	fdPreimageRead  = 5
	fdPreimageWrite = 6
)

const (
	MipsEBADF  = 0x9
	MipsEINVAL = 0x16
)

type PreimageReader func(key [32]byte, offset uint32) (dat [32]byte, datLen uint32)
type MemTracker func(addr uint32)

func getSyscallArgs(registers *[32]uint32) (syscallNum, a0, a1, a2 uint32) {
	syscallNum = registers[2] // v0

	a0 = registers[4]
	a1 = registers[5]
	a2 = registers[6]

	return syscallNum, a0, a1, a2
}

func handleSysMmap(a0, a1, heap uint32) (v0, v1, newHeap uint32) {
	v1 = uint32(0)
	newHeap = heap

	sz := a1
	if sz&PageAddrMask != 0 { // adjust size to align with page size
		sz += PageSize - (sz & PageAddrMask)
	}
	if a0 == 0 {
		v0 = heap
		//fmt.Printf("mmap heap 0x%x size 0x%x\n", v0, sz)
		newHeap += sz
	} else {
		v0 = a0
		//fmt.Printf("mmap hint 0x%x size 0x%x\n", v0, sz)
	}

	return v0, v1, newHeap
}

func handleSysRead(a0, a1, a2 uint32, preimageKey [32]byte, preimageOffset uint32, preimageReader PreimageReader, memory *Memory, memTracker MemTracker) (v0, v1, newPreimageOffset uint32) {
	// args: a0 = fd, a1 = addr, a2 = count
	// returns: v0 = read, v1 = err code
	v0 = uint32(0)
	v1 = uint32(0)
	newPreimageOffset = preimageOffset

	switch a0 {
	case fdStdin:
		// leave v0 and v1 zero: read nothing, no error
	case fdPreimageRead: // pre-image oracle
		effAddr := a1 & 0xFFffFFfc
		memTracker(effAddr)
		mem := memory.GetMemory(effAddr)
		dat, datLen := preimageReader(preimageKey, preimageOffset)
		//fmt.Printf("reading pre-image data: addr: %08x, offset: %d, datLen: %d, data: %x, key: %s  count: %d\n", a1, m.state.PreimageOffset, datLen, dat[:datLen], m.state.PreimageKey, a2)
		alignment := a1 & 3
		space := 4 - alignment
		if space < datLen {
			datLen = space
		}
		if a2 < datLen {
			datLen = a2
		}
		var outMem [4]byte
		binary.BigEndian.PutUint32(outMem[:], mem)
		copy(outMem[alignment:], dat[:datLen])
		memory.SetMemory(effAddr, binary.BigEndian.Uint32(outMem[:]))
		newPreimageOffset += datLen
		v0 = datLen
		//fmt.Printf("read %d pre-image bytes, new offset: %d, eff addr: %08x mem: %08x\n", datLen, m.state.PreimageOffset, effAddr, outMem)
	case fdHintRead: // hint response
		// don't actually read into memory, just say we read it all, we ignore the result anyway
		v0 = a2
	default:
		v0 = 0xFFffFFff
		v1 = MipsEBADF
	}

	return v0, v1, newPreimageOffset
}

func handleSysWrite(a0, a1, a2 uint32, lastHint hexutil.Bytes, preimageKey [32]byte, preimageOffset uint32, oracle PreimageOracle, memory *Memory, memTracker MemTracker, stdOut, stdErr io.Writer) (v0, v1 uint32, newLastHint hexutil.Bytes, newPreimageKey common.Hash, newPreimageOffset uint32) {
	// args: a0 = fd, a1 = addr, a2 = count
	// returns: v0 = written, v1 = err code
	v1 = uint32(0)
	newLastHint = lastHint
	newPreimageKey = preimageKey
	newPreimageOffset = preimageOffset

	switch a0 {
	case fdStdout:
		_, _ = io.Copy(stdOut, memory.ReadMemoryRange(a1, a2))
		v0 = a2
	case fdStderr:
		_, _ = io.Copy(stdErr, memory.ReadMemoryRange(a1, a2))
		v0 = a2
	case fdHintWrite:
		hintData, _ := io.ReadAll(memory.ReadMemoryRange(a1, a2))
		lastHint = append(lastHint, hintData...)
		for len(lastHint) >= 4 { // process while there is enough data to check if there are any hints
			hintLen := binary.BigEndian.Uint32(lastHint[:4])
			if hintLen <= uint32(len(lastHint[4:])) {
				hint := lastHint[4 : 4+hintLen] // without the length prefix
				lastHint = lastHint[4+hintLen:]
				oracle.Hint(hint)
			} else {
				break // stop processing hints if there is incomplete data buffered
			}
		}
		newLastHint = lastHint
		v0 = a2
	case fdPreimageWrite:
		effAddr := a1 & 0xFFffFFfc
		memTracker(effAddr)
		mem := memory.GetMemory(effAddr)
		key := preimageKey
		alignment := a1 & 3
		space := 4 - alignment
		if space < a2 {
			a2 = space
		}
		copy(key[:], key[a2:])
		var tmp [4]byte
		binary.BigEndian.PutUint32(tmp[:], mem)
		copy(key[32-a2:], tmp[alignment:])
		newPreimageKey = key
		newPreimageOffset = 0
		//fmt.Printf("updating pre-image key: %s\n", m.state.PreimageKey)
		v0 = a2
	default:
		v0 = 0xFFffFFff
		v1 = MipsEBADF
	}

	return v0, v1, newLastHint, newPreimageKey, newPreimageOffset
}

func handleSysFcntl(a0, a1 uint32) (v0, v1 uint32) {
	// args: a0 = fd, a1 = cmd
	v1 = uint32(0)

	if a1 == 3 { // F_GETFL: get file descriptor flags
		switch a0 {
		case fdStdin, fdPreimageRead, fdHintRead:
			v0 = 0 // O_RDONLY
		case fdStdout, fdStderr, fdPreimageWrite, fdHintWrite:
			v0 = 1 // O_WRONLY
		default:
			v0 = 0xFFffFFff
			v1 = MipsEBADF
		}
	} else {
		v0 = 0xFFffFFff
		v1 = MipsEINVAL // cmd not recognized by this kernel
	}

	return v0, v1
}

func handleSyscallUpdates(cpu *CpuScalars, registers *[32]uint32, v0, v1 uint32) {
	registers[2] = v0
	registers[7] = v1

	cpu.PC = cpu.NextPC
	cpu.NextPC = cpu.NextPC + 4
}
