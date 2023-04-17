package main

const (
	pageAddrSize = 10
	pageSize     = 1 << pageAddrSize
)

type Page [pageSize]byte

type State struct {
	PC        uint32
	Registers [32]uint32
	Hi, Lo    uint32 // special registers
	Heap      uint32 // to handle mmap growth

	Memory map[uint32]*Page

	Exit   uint32
	Exited bool
}

// TODO hooks for unicorn to modify the state

// TODO merkleization

// TODO load from JSON

// TODO store to JSON

// TODO hooks to detect state reads/writes
// -> maintain access-list

// TODO load Unicorn with hooks

// TODO convert access-list to calldata and state-sets for EVM
