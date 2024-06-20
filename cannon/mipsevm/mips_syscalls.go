package mipsevm

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
