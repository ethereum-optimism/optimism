package multithreaded

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

// SERIALIZED_THREAD_SIZE is the size of a serialized ThreadState object
const SERIALIZED_THREAD_SIZE = 166

// THREAD_WITNESS_SIZE is the size of a thread witness encoded in bytes.
//
//	It consists of the active thread serialized and concatenated with the
//	32 byte hash onion of the active thread stack without the active thread
const THREAD_WITNESS_SIZE = SERIALIZED_THREAD_SIZE + 32

// The empty thread root - keccak256(bytes32(0) ++ bytes32(0))
var EmptyThreadsRoot common.Hash = common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")

type ThreadState struct {
	ThreadId         uint32             `json:"threadId"`
	ExitCode         uint8              `json:"exit"`
	Exited           bool               `json:"exited"`
	FutexAddr        uint32             `json:"futexAddr"`
	FutexVal         uint32             `json:"futexVal"`
	FutexTimeoutStep uint64             `json:"futexTimeoutStep"`
	Cpu              mipsevm.CpuScalars `json:"cpu"`
	Registers        [32]uint32         `json:"registers"`
}

func (t *ThreadState) serializeThread() []byte {
	out := make([]byte, 0, SERIALIZED_THREAD_SIZE)

	out = binary.BigEndian.AppendUint32(out, t.ThreadId)
	out = append(out, t.ExitCode)
	out = mipsevm.AppendBoolToWitness(out, t.Exited)
	out = binary.BigEndian.AppendUint32(out, t.FutexAddr)
	out = binary.BigEndian.AppendUint32(out, t.FutexVal)
	out = binary.BigEndian.AppendUint64(out, t.FutexTimeoutStep)

	out = binary.BigEndian.AppendUint32(out, t.Cpu.PC)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.NextPC)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.LO)
	out = binary.BigEndian.AppendUint32(out, t.Cpu.HI)

	for _, r := range t.Registers {
		out = binary.BigEndian.AppendUint32(out, r)
	}

	return out
}

func computeThreadRoot(prevStackRoot common.Hash, threadToPush *ThreadState) common.Hash {
	hashedThread := crypto.Keccak256Hash(threadToPush.serializeThread())

	var hashData []byte
	hashData = append(hashData, prevStackRoot[:]...)
	hashData = append(hashData, hashedThread[:]...)

	return crypto.Keccak256Hash(hashData)
}

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

	TraverseRight    bool          `json:"traverseRight"`
	LeftThreadStack  []ThreadState `json:"leftThreadStack"`
	RightThreadStack []ThreadState `json:"rightThreadStack"`
	NextThreadId     uint32        `json:"nextThreadId"`

	// LastHint is optional metadata, and not part of the VM state itself.
	// It is used to remember the last pre-image hint,
	// so a VM can start from any state without fetching prior pre-images,
	// and instead just repeat the last hint on setup,
	// to make sure pre-image requests can be served.
	// The first 4 bytes are a uin32 length prefix.
	// Warning: the hint MAY NOT BE COMPLETE. I.e. this is buffered,
	// and should only be read when len(LastHint) > 4 && uint32(LastHint[:4]) <= len(LastHint[4:])
	LastHint hexutil.Bytes `json:"lastHint,omitempty"`
}

func CreateEmptyState() *State {
	initThreadId := uint32(0)
	initThread := ThreadState{
		ThreadId: initThreadId,
		ExitCode: 0,
		Exited:   false,
		Cpu: mipsevm.CpuScalars{
			PC:     0,
			NextPC: 0,
			LO:     0,
			HI:     0,
		},
		FutexAddr:        ^uint32(0),
		FutexVal:         0,
		FutexTimeoutStep: 0,
		Registers:        [32]uint32{},
	}

	return &State{
		Memory:           memory.NewMemory(),
		Heap:             0,
		ExitCode:         0,
		Exited:           false,
		Step:             0,
		Wakeup:           ^uint32(0),
		TraverseRight:    false,
		LeftThreadStack:  []ThreadState{initThread},
		RightThreadStack: []ThreadState{},
		NextThreadId:     initThreadId + 1,
	}
}

func CreateInitialState(pc, heapStart uint32) *State {
	state := CreateEmptyState()
	currentThread := state.getCurrentThread()
	currentThread.Cpu.PC = pc
	currentThread.Cpu.NextPC = pc + 4
	state.Heap = heapStart

	return state
}

func (s *State) getCurrentThread() *ThreadState {
	activeStack := s.getActiveThreadStack()

	activeStackSize := len(activeStack)
	if activeStackSize == 0 {
		panic("Active thread stack is empty")
	}

	return &activeStack[activeStackSize-1]
}

func (s *State) getActiveThreadStack() []ThreadState {
	var activeStack []ThreadState
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

func (s *State) calculateThreadStackRoot(stack []ThreadState) common.Hash {
	curRoot := EmptyThreadsRoot
	for _, thread := range stack {
		curRoot = computeThreadRoot(curRoot, &thread)
	}

	return curRoot
}

func (s *State) PreemptThread() {
	// TODO(CP-903)
	panic("Not Implemented")
}

func (s *State) PushThread(thread *ThreadState) {
	// TODO(CP-903)
	panic("Not Implemented")
}

func (s *State) GetPC() uint32 {
	activeThread := s.getCurrentThread()
	return activeThread.Cpu.PC
}

func (s *State) GetRegisters() *[32]uint32 {
	activeThread := s.getCurrentThread()
	return &activeThread.Registers
}

func (s *State) GetExitCode() uint8 { return s.ExitCode }

func (s *State) GetExited() bool { return s.Exited }

func (s *State) GetStep() uint64 { return s.Step }

func (s *State) VMStatus() uint8 {
	return mipsevm.VmStatus(s.Exited, s.ExitCode)
}

func (s *State) GetMemory() *memory.Memory {
	return s.Memory
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

type StateWitness []byte

func (sw StateWitness) StateHash() (common.Hash, error) {
	if len(sw) != STATE_WITNESS_SIZE {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected %d", len(sw), STATE_WITNESS_SIZE)
	}
	return stateHashFromWitness(sw), nil
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
