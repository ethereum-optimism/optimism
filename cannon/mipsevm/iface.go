package mipsevm

import (
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
)

type FPVMState interface {
	serialize.Serializable

	GetMemory() *memory.Memory

	// GetHeap returns the current memory address at the top of the heap
	GetHeap() arch.Word

	// GetPreimageKey returns the most recently accessed preimage key
	GetPreimageKey() common.Hash

	// GetPreimageOffset returns the current offset into the current preimage
	GetPreimageOffset() arch.Word

	// GetPC returns the currently executing program counter
	GetPC() arch.Word

	// GetCpu returns the currently active cpu scalars, including the program counter
	GetCpu() CpuScalars

	// GetRegistersRef returns a pointer to the currently active registers
	GetRegistersRef() *[32]arch.Word

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
	// The first 4 bytes are a Word length prefix.
	// Warning: the hint MAY NOT BE COMPLETE. I.e. this is buffered,
	// and should only be read when len(LastHint) > 4 && Word(LastHint[:4]) <= len(LastHint[4:])
	GetLastHint() hexutil.Bytes

	// EncodeWitness returns the witness for the current state and the state hash
	EncodeWitness() (witness []byte, hash common.Hash)

	// CreateVM creates a FPVM that can operate on this state.
	CreateVM(logger log.Logger, po PreimageOracle, stdOut, stdErr io.Writer, meta Metadata, features FeatureToggles) FPVM
}

type SymbolMatcher func(addr arch.Word) bool

type Metadata interface {
	LookupSymbol(addr arch.Word) string
	CreateSymbolMatcher(name string) SymbolMatcher
}

// FeatureToggles defines the set of features which are enabled only on some of the supported state versions.
// This allows supporting multiple state versions concurrently without needing to create completely separate
// FPVM implementations and duplicate a lot of code.
// Toggles here are temporary and should be removed once the newer state version is deployed widely. The older
// version can then be supported via multicannon pulling in a specific build and support for it dropped in latest code.
type FeatureToggles struct {
	SupportWorkingSysGetRandom bool
}

type FPVM interface {
	// GetState returns the current state of the VM. The FPVMState is updated by successive calls to Step
	GetState() FPVMState

	// Step executes a single instruction and returns the witness for the step
	Step(includeProof bool) (*StepWitness, error)

	// CheckInfiniteLoop returns true if the vm is stuck in an infinite loop
	CheckInfiniteLoop() bool

	// LastPreimage returns the last preimage accessed by the VM
	LastPreimage() (preimageKey [32]byte, preimage []byte, preimageOffset arch.Word)

	// Traceback prints a traceback of the program to the console
	Traceback()

	// GetDebugInfo returns debug information about the VM
	GetDebugInfo() *DebugInfo

	// InitDebug initializes the debug mode of the VM
	InitDebug() error

	// EnableStats if supported by the VM, enables some additional statistics that can be retrieved via GetDebugInfo()
	EnableStats()

	// LookupSymbol returns the symbol located at the specified address.
	// May return an empty string if there's no symbol table available.
	LookupSymbol(addr arch.Word) string
}
