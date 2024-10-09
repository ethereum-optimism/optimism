package exec

// FYI: https://en.wikibooks.org/wiki/MIPS_Assembly/Register_File
const (
	Reg0  = 0
	RegAt = 1
	// syscall number; 1st return value
	RegV0 = 2
	// 2nd return value
	RegV1 = 3
	// syscall arguments; returned unmodified
	RegA0 = 4
	RegA1 = 5
	RegA2 = 6
	// 4th syscall argument; set to 0/1 for success/error
	RegA3 = 7
	// caller saved
	RegT0 = 8
	RegT1 = 9
	RegT2 = 10
	RegT3 = 11
	RegT4 = 12
	RegT5 = 13
	RegT6 = 14
	RegT7 = 15
	// callee saved
	RegS0 = 16
	RegS1 = 17
	RegS2 = 18
	RegS3 = 19
	RegS4 = 20
	RegS5 = 21
	RegS6 = 22
	RegS7 = 23

	RegT8 = 24
	RegT9 = 25
	RegK0 = 26
	RegK1 = 27
	RegGP = 28
	RegSP = 29
	RegFP = 30
	RegRA = 31
)

// FYI: https://web.archive.org/web/20231223163047/https://www.linux-mips.org/wiki/Syscall
const (
	RegSyscallNum    = RegV0
	RegSyscallRet1   = RegV0
	RegSyscallRet2   = RegV1
	RegSyscallResult = RegA3
	RegSyscallParam1 = RegA0
	RegSyscallParam2 = RegA1
	RegSyscallParam3 = RegA2
	RegSyscallParam4 = RegA3
)
