package mipsevm

import "encoding/binary"

type MemTracker func(addr uint32)
type MemGetter func(addr uint32) uint32
type MemSetter func(addr uint32, val uint32)
type PreimageReader func(key [32]byte, offset uint32) (dat [32]byte, datLen uint32)

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

func getSyscallArgs(registers *[32]uint32) (syscallNum, a0, a1, a2 uint32) {
	syscallNum = registers[2] // v0

	a0 = registers[4]
	a1 = registers[5]
	a2 = registers[6]

	return syscallNum, a0, a1, a2
}

func handleMmap(a0, a1, heap uint32) (v0, v1, newHeap uint32) {
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

func handleSysRead(a0, a1, a2 uint32, preimageKey [32]byte, preimageOffset uint32, preimageReader PreimageReader, memGetter MemGetter, memSetter MemSetter, memTracker MemTracker) (v0, v1, newPreimageOffset uint32) {
	v0 = uint32(0)
	v1 = uint32(0)
	newPreimageOffset = preimageOffset

	// args: a0 = fd, a1 = addr, a2 = count
	// returns: v0 = read, v1 = err code
	switch a0 {
	case fdStdin:
		// leave v0 and v1 zero: read nothing, no error
	case fdPreimageRead: // pre-image oracle
		effAddr := a1 & 0xFFffFFfc
		memTracker(effAddr)
		mem := memGetter(effAddr)
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
		memSetter(effAddr, binary.BigEndian.Uint32(outMem[:]))
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
