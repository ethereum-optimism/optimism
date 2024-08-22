package testutil

// 0xbf_c0_00_00 ... BaseAddrEnd is used in tests to write the results to
const BaseAddrEnd = 0xbf_ff_ff_f0

// EndAddr is used as return-address for tests
const EndAddr = 0xa7ef00d0

type MipsVersion int

const (
	MipsSingleThreaded MipsVersion = iota
	MipsMultithreaded
)
