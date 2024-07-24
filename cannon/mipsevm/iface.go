package mipsevm

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

type FPVMState interface {
	GetMemory() *memory.Memory

	// GetPC returns the currently executing program counter
	GetPC() uint32

	// GetRegisters returns the currently active registers
	GetRegisters() *[32]uint32

	// GetStep returns the current VM step
	GetStep() uint64

	// GetExited returns whether the state exited bit is set
	GetExited() bool

	// GetExitCode returns the exit code
	GetExitCode() uint8

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
}
