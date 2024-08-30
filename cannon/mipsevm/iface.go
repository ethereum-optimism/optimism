package mipsevm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

type FPVMState interface {
	GetMemory() *memory.Memory

	// GetHeap returns the current memory address at the top of the heap
	GetHeap() uint32

	// GetPreimageKey returns the most recently accessed preimage key
	GetPreimageKey() common.Hash

	// GetPreimageOffset returns the current offset into the current preimage
	GetPreimageOffset() uint32

	// GetPC returns the currently executing program counter
	GetPC() uint32

	// GetCpu returns the currently active cpu scalars, including the program counter
	GetCpu() CpuScalars

	// GetRegistersRef returns a pointer to the currently active registers
	GetRegistersRef() *[32]uint32

	// GetStep returns the current VM step
	GetStep() uint64

	// GetExited returns whether the state exited bit is set
	GetExited() bool

	// GetExitCode returns the exit code
	GetExitCode() uint8

	// GetLastHint returns optional metadata which is not part of the VM state itself.
	// It is used to remember the last pre-image hint,
	// so a VM can start from any state without fetching prior pre-images,
	// and instead just repeat the last hint on setup,
	// to make sure pre-image requests can be served.
	// The first 4 bytes are a uint32 length prefix.
	// Warning: the hint MAY NOT BE COMPLETE. I.e. this is buffered,
	// and should only be read when len(LastHint) > 4 && uint32(LastHint[:4]) <= len(LastHint[4:])
	GetLastHint() hexutil.Bytes

	// EncodeWitness returns the witness for the current state and the state hash
	EncodeWitness() (witness []byte, hash common.Hash)
}

type FPVM interface {
	// GetState returns the current state of the VM. The FPVMState is updated by successive calls to Step
	GetState() FPVMState

	// Step executes a single instruction and returns the witness for the step
	Step(includeProof bool) (*StepWitness, error)

	// CheckInfiniteLoop returns true if the vm is stuck in an infinite loop
	CheckInfiniteLoop() bool

	// LastPreimage returns the last preimage accessed by the VM
	LastPreimage() (preimageKey [32]byte, preimage []byte, preimageOffset uint32)

	// Traceback prints a traceback of the program to the console
	Traceback()

	// GetDebugInfo returns debug information about the VM
	GetDebugInfo() *DebugInfo

	// LookupSymbol returns the symbol located at the specified address.
	// May return an empty string if there's no symbol table available.
	LookupSymbol(addr uint32) string
}
