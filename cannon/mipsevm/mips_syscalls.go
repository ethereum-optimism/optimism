package mipsevm

func getSyscallArgs(registers *[32]uint32) (syscallNum, a0, a1, a2 uint32) {
	syscallNum = registers[2] // v0

	a0 = registers[4]
	a1 = registers[5]
	a2 = registers[6]

	return syscallNum, a0, a1, a2
}
