package multithreaded

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

// STATE_WITNESS_SIZE is the size of the state witness encoding in bytes.
const STATE_WITNESS_SIZE = 163
const (
	MEMROOT_WITNESS_OFFSET                    = 0
	PREIMAGE_KEY_WITNESS_OFFSET               = MEMROOT_WITNESS_OFFSET + 32
	PREIMAGE_OFFSET_WITNESS_OFFSET            = PREIMAGE_KEY_WITNESS_OFFSET + 32
	HEAP_WITNESS_OFFSET                       = PREIMAGE_OFFSET_WITNESS_OFFSET + 4
	EXITCODE_WITNESS_OFFSET                   = HEAP_WITNESS_OFFSET + 4
	EXITED_WITNESS_OFFSET                     = EXITCODE_WITNESS_OFFSET + 1
	STEP_WITNESS_OFFSET                       = EXITED_WITNESS_OFFSET + 1
	STEPS_SINCE_CONTEXT_SWITCH_WITNESS_OFFSET = STEP_WITNESS_OFFSET + 8
	WAKEUP_WITNESS_OFFSET                     = STEPS_SINCE_CONTEXT_SWITCH_WITNESS_OFFSET + 8
	TRAVERSE_RIGHT_WITNESS_OFFSET             = WAKEUP_WITNESS_OFFSET + 4
	LEFT_THREADS_ROOT_WITNESS_OFFSET          = TRAVERSE_RIGHT_WITNESS_OFFSET + 1
	RIGHT_THREADS_ROOT_WITNESS_OFFSET         = LEFT_THREADS_ROOT_WITNESS_OFFSET + 32
	THREAD_ID_WITNESS_OFFSET                  = RIGHT_THREADS_ROOT_WITNESS_OFFSET + 32
)

type State struct {
	Memory *memory.Memory `json:"memory"`

	PreimageKey    common.Hash `json:"preimageKey"`
	PreimageOffset uint32      `json:"preimageOffset"` // note that the offset includes the 8-byte length prefix

	Heap uint32 `json:"heap"` // to handle mmap growth

	ExitCode uint8 `json:"exit"`
	Exited   bool  `json:"exited"`

	Step                        uint64 `json:"step"`
	StepsSinceLastContextSwitch uint64 `json:"stepsSinceLastContextSwitch"`
	Wakeup                      uint32 `json:"wakeup"`

	TraverseRight    bool           `json:"traverseRight"`
	LeftThreadStack  []*ThreadState `json:"leftThreadStack"`
	RightThreadStack []*ThreadState `json:"rightThreadStack"`
	NextThreadId     uint32         `json:"nextThreadId"`

	// LastHint is optional metadata, and not part of the VM state itself.
	LastHint hexutil.Bytes `json:"lastHint,omitempty"`
}

var _ mipsevm.FPVMState = (*State)(nil)

func CreateEmptyState() *State {
	initThread := CreateEmptyThread()

	return &State{
		Memory:           memory.NewMemory(),
		Heap:             0,
		ExitCode:         0,
		Exited:           false,
		Step:             0,
		Wakeup:           exec.FutexEmptyAddr,
		TraverseRight:    false,
		LeftThreadStack:  []*ThreadState{initThread},
		RightThreadStack: []*ThreadState{},
		NextThreadId:     initThread.ThreadId + 1,
	}
}

func CreateInitialState(pc, heapStart uint32) *State {
	state := CreateEmptyState()
	currentThread := state.GetCurrentThread()
	currentThread.Cpu.PC = pc
	currentThread.Cpu.NextPC = pc + 4
	state.Heap = heapStart

	return state
}

func (s *State) GetCurrentThread() *ThreadState {
	activeStack := s.getActiveThreadStack()

	activeStackSize := len(activeStack)
	if activeStackSize == 0 {
		panic("Active thread stack is empty")
	}

	return activeStack[activeStackSize-1]
}

func (s *State) getActiveThreadStack() []*ThreadState {
	var activeStack []*ThreadState
	if s.TraverseRight {
		activeStack = s.RightThreadStack
	} else {
		activeStack = s.LeftThreadStack
	}

	return activeStack
}

func (s *State) getRightThreadStackRoot() common.Hash {
	return s.calculateThreadStackRoot(s.RightThreadStack)
}

func (s *State) getLeftThreadStackRoot() common.Hash {
	return s.calculateThreadStackRoot(s.LeftThreadStack)
}

func (s *State) calculateThreadStackRoot(stack []*ThreadState) common.Hash {
	curRoot := EmptyThreadsRoot
	for _, thread := range stack {
		curRoot = computeThreadRoot(curRoot, thread)
	}

	return curRoot
}

func (s *State) GetPC() uint32 {
	activeThread := s.GetCurrentThread()
	return activeThread.Cpu.PC
}

func (s *State) GetCpu() mipsevm.CpuScalars {
	activeThread := s.GetCurrentThread()
	return activeThread.Cpu
}

func (s *State) getCpuRef() *mipsevm.CpuScalars {
	return &s.GetCurrentThread().Cpu
}

func (s *State) GetRegistersRef() *[32]uint32 {
	activeThread := s.GetCurrentThread()
	return &activeThread.Registers
}

func (s *State) GetExitCode() uint8 { return s.ExitCode }

func (s *State) GetExited() bool { return s.Exited }

func (s *State) GetStep() uint64 { return s.Step }

func (s *State) GetLastHint() hexutil.Bytes {
	return s.LastHint
}

func (s *State) VMStatus() uint8 {
	return mipsevm.VmStatus(s.Exited, s.ExitCode)
}

func (s *State) GetMemory() *memory.Memory {
	return s.Memory
}

func (s *State) GetHeap() uint32 {
	return s.Heap
}

func (s *State) GetPreimageKey() common.Hash {
	return s.PreimageKey
}

func (s *State) GetPreimageOffset() uint32 {
	return s.PreimageOffset
}

func (s *State) EncodeWitness() ([]byte, common.Hash) {
	out := make([]byte, 0, STATE_WITNESS_SIZE)
	memRoot := s.Memory.MerkleRoot()
	out = append(out, memRoot[:]...)
	out = append(out, s.PreimageKey[:]...)
	out = binary.BigEndian.AppendUint32(out, s.PreimageOffset)
	out = binary.BigEndian.AppendUint32(out, s.Heap)
	out = append(out, s.ExitCode)
	out = mipsevm.AppendBoolToWitness(out, s.Exited)

	out = binary.BigEndian.AppendUint64(out, s.Step)
	out = binary.BigEndian.AppendUint64(out, s.StepsSinceLastContextSwitch)
	out = binary.BigEndian.AppendUint32(out, s.Wakeup)

	leftStackRoot := s.getLeftThreadStackRoot()
	rightStackRoot := s.getRightThreadStackRoot()
	out = mipsevm.AppendBoolToWitness(out, s.TraverseRight)
	out = append(out, (leftStackRoot)[:]...)
	out = append(out, (rightStackRoot)[:]...)
	out = binary.BigEndian.AppendUint32(out, s.NextThreadId)

	return out, stateHashFromWitness(out)
}

func (s *State) EncodeThreadProof() []byte {
	activeStack := s.getActiveThreadStack()
	threadCount := len(activeStack)
	if threadCount == 0 {
		panic("Invalid empty thread stack")
	}

	activeThread := activeStack[threadCount-1]
	otherThreads := activeStack[:threadCount-1]
	threadBytes := activeThread.serializeThread()
	otherThreadsWitness := s.calculateThreadStackRoot(otherThreads)

	out := make([]byte, 0, THREAD_WITNESS_SIZE)
	out = append(out, threadBytes[:]...)
	out = append(out, otherThreadsWitness[:]...)

	return out
}

func (s *State) ThreadCount() int {
	return len(s.LeftThreadStack) + len(s.RightThreadStack)
}

type StateWitness []byte

func (sw StateWitness) StateHash() (common.Hash, error) {
	if len(sw) != STATE_WITNESS_SIZE {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected %d", len(sw), STATE_WITNESS_SIZE)
	}
	return stateHashFromWitness(sw), nil
}

func GetStateHashFn() mipsevm.HashFn {
	return func(sw []byte) (common.Hash, error) {
		return StateWitness(sw).StateHash()
	}
}

func stateHashFromWitness(sw []byte) common.Hash {
	if len(sw) != STATE_WITNESS_SIZE {
		panic("Invalid witness length")
	}
	hash := crypto.Keccak256Hash(sw)
	exitCode := sw[EXITCODE_WITNESS_OFFSET]
	exited := sw[EXITED_WITNESS_OFFSET]
	status := mipsevm.VmStatus(exited == 1, exitCode)
	hash[0] = status
	return hash
}
