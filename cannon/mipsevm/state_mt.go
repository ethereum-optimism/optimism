package mipsevm

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// SERIALIZED_THREAD_SIZE is the size of a serialized ThreadContext object
const SERIALIZED_THREAD_SIZE = 166

// THREAD_WITNESS_SIZE is the size of a thread witness encoded in bytes.
//
//	It consists of the active thread serialized and concatenated with the
//	32 byte hash onion of the active thread stack without the active thread
const THREAD_WITNESS_SIZE = SERIALIZED_THREAD_SIZE + 32

// The empty thread root - keccak256(bytes32(0) ++ bytes32(0))
var EmptyThreadsRoot common.Hash = common.HexToHash("0xad3228b676f7d3cd4284a5443f17f1962b36e491b30a40b2405849e597ba5fb5")

type ThreadContext struct {
	// Metadata
	ThreadId uint32 `json:"threadId"`
	ExitCode uint8  `json:"exit"`
	Exited   bool   `json:"exited"`
	// State
	FutexAddr        uint32     `json:"futexAddr"`
	FutexVal         uint32     `json:"futexVal"`
	FutexTimeoutStep uint64     `json:"futexTimeoutStep"`
	Cpu              CpuScalars `json:"cpu"`
	Registers        [32]uint32 `json:"registers"`
}

func (t *ThreadContext) serializeThread() []byte {
	// TODO
	out := make([]byte, 0, SERIALIZED_THREAD_SIZE)

	//out = binary.BigEndian.AppendUint32(out, t.Cpu.PC)
	//out = binary.BigEndian.AppendUint32(out, t.Cpu.NextPC)
	//out = binary.BigEndian.AppendUint32(out, t.Cpu.LO)
	//out = binary.BigEndian.AppendUint32(out, t.Cpu.HI)
	//
	//for _, r := range t.Registers {
	//	out = binary.BigEndian.AppendUint32(out, r)
	//}

	return out
}

func computeThreadRoot(prevStackRoot common.Hash, threadToPush *ThreadContext) common.Hash {
	hashedThread := crypto.Keccak256Hash(threadToPush.serializeThread())

	var hashData []byte
	hashData = append(hashData, prevStackRoot[:]...)
	hashData = append(hashData, hashedThread[:]...)

	return crypto.Keccak256Hash(hashData)
}

// MT_STATE_WITNESS_SIZE is the size of the state witness encoding in bytes.
const MT_STATE_WITNESS_SIZE = 163
const (
	MEMROOT_MT_WITNESS_OFFSET                    = 0
	PREIMAGE_KEY_MT_WITNESS_OFFSET               = MEMROOT_MT_WITNESS_OFFSET + 32
	PREIMAGE_OFFSET_MT_WITNESS_OFFSET            = PREIMAGE_KEY_MT_WITNESS_OFFSET + 32
	HEAP_MT_WITNESS_OFFSET                       = PREIMAGE_OFFSET_MT_WITNESS_OFFSET + 4
	EXITCODE_MT_WITNESS_OFFSET                   = HEAP_MT_WITNESS_OFFSET + 4
	EXITED_MT_WITNESS_OFFSET                     = EXITCODE_MT_WITNESS_OFFSET + 1
	STEP_MT_WITNESS_OFFSET                       = EXITED_MT_WITNESS_OFFSET + 1
	STEPS_SINCE_CONTEXT_SWITCH_MT_WITNESS_OFFSET = STEP_MT_WITNESS_OFFSET + 8
	WAKEUP_MT_WITNESS_OFFSET                     = STEPS_SINCE_CONTEXT_SWITCH_MT_WITNESS_OFFSET + 8
	TRAVERSE_RIGHT_MT_WITNESS_OFFSET             = WAKEUP_MT_WITNESS_OFFSET + 4
	LEFT_THREADS_ROOT_MT_WITNESS_OFFSET          = TRAVERSE_RIGHT_MT_WITNESS_OFFSET + 1
	RIGHT_THREADS_ROOT_MT_WITNESS_OFFSET         = LEFT_THREADS_ROOT_MT_WITNESS_OFFSET + 32
	THREAD_ID_MT_WITNESS_OFFSET                  = RIGHT_THREADS_ROOT_MT_WITNESS_OFFSET + 32
)

type MTState struct {
	Memory *Memory `json:"memory"`

	PreimageKey    common.Hash `json:"preimageKey"`
	PreimageOffset uint32      `json:"preimageOffset"` // note that the offset includes the 8-byte length prefix

	Heap uint32 `json:"heap"` // to handle mmap growth

	ExitCode uint8 `json:"exit"`
	Exited   bool  `json:"exited"`

	Step                        uint64 `json:"step"`
	StepsSinceLastContextSwitch uint64 `json:"stepsSinceLastContextSwitch"`
	Wakeup                      uint32 `json:"wakeup"`

	TraverseRight         bool            `json:"traverseRight"`
	LeftThreadStack       []ThreadContext `json:"leftThreadStack"`
	RightThreadStack      []ThreadContext `json:"rightThreadStack"`
	LeftThreadStackRoots  []common.Hash   `json:"leftThreadStackRoots"`
	RightThreadStackRoots []common.Hash   `json:"rightThreadStackRoots"`
	NextThreadId          uint32          `json:"nextThreadId"`

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

func CreateEmptyMTState() *MTState {
	initThreadId := uint32(0)
	initThread := ThreadContext{
		ThreadId: initThreadId,
		ExitCode: 0,
		Exited:   false,
		Cpu: CpuScalars{
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
	initThreadRoot := computeThreadRoot(EmptyThreadsRoot, &initThread)

	return &MTState{
		Memory:                NewMemory(),
		Heap:                  0,
		ExitCode:              0,
		Exited:                false,
		Step:                  0,
		Wakeup:                ^uint32(0),
		TraverseRight:         true,
		LeftThreadStack:       []ThreadContext{},
		RightThreadStack:      []ThreadContext{initThread},
		LeftThreadStackRoots:  []common.Hash{},
		RightThreadStackRoots: []common.Hash{initThreadRoot},
		NextThreadId:          initThreadId + 1,
	}
}

func CreateInitialMTState(pc, heapStart uint32) *MTState {
	state := CreateEmptyMTState()
	state.UpdateCurrentThread(func(t *ThreadContext) {
		t.Cpu.PC = pc
		t.Cpu.NextPC = pc + 4
	})
	state.Heap = heapStart

	return state
}

func (s *MTState) getCurrentThread() *ThreadContext {
	activeStack, _ := s.getActiveThreadStack()

	activeStackSize := len(activeStack)
	if activeStackSize == 0 {
		panic("Active thread stack is empty")
	}

	return &activeStack[activeStackSize-1]
}

func (s *MTState) updateCurrentThreadRoot() {
	activeStack, activeRoots := s.getActiveThreadStack()
	activeStackSize := len(activeStack)

	prevStackRoot := EmptyThreadsRoot
	if activeStackSize > 1 {
		prevStackRoot = activeRoots[activeStackSize-2]
	}

	newRoot := computeThreadRoot(prevStackRoot, s.getCurrentThread())
	activeRoots[activeStackSize-1] = newRoot
}

type ThreadMutator func(thread *ThreadContext)

func (s *MTState) UpdateCurrentThread(mutator ThreadMutator) {
	activeThread := s.getCurrentThread()
	mutator(activeThread)
	s.updateCurrentThreadRoot()
}

func (s *MTState) getActiveThreadStack() ([]ThreadContext, []common.Hash) {
	var activeStack []ThreadContext
	var activeStackRoots []common.Hash
	if s.TraverseRight {
		activeStack = s.RightThreadStack
		activeStackRoots = s.RightThreadStackRoots
	} else {
		activeStack = s.LeftThreadStack
		activeStackRoots = s.LeftThreadStackRoots
	}

	return activeStack, activeStackRoots
}

func (s *MTState) getRightThreadStackRoot() common.Hash {
	return s.getThreadStackRoot(s.RightThreadStackRoots)
}

func (s *MTState) getLeftThreadStackRoot() common.Hash {
	return s.getThreadStackRoot(s.LeftThreadStackRoots)
}

func (s *MTState) getThreadStackRoot(stackRoots []common.Hash) common.Hash {
	stackLen := len(stackRoots)
	if stackLen == 0 {
		return EmptyThreadsRoot
	}

	return stackRoots[stackLen-1]
}

func (s *MTState) PreemptThread() {
	// TODO
}

func (s *MTState) PushThread(thread *ThreadContext) {
	// TODO
}

func (s *MTState) GetPC() uint32 {
	activeThread := s.getCurrentThread()
	return activeThread.Cpu.PC
}

func (s *MTState) GetExitCode() uint8 { return s.ExitCode }

func (s *MTState) GetExited() bool { return s.Exited }

func (s *MTState) GetStep() uint64 { return s.Step }

func (s *MTState) VMStatus() uint8 {
	return vmStatus(s.Exited, s.ExitCode)
}

func (s *MTState) GetMemory() *Memory {
	return s.Memory
}

func (s *MTState) EncodeWitness() ([]byte, common.Hash) {
	out := make([]byte, 0, MT_STATE_WITNESS_SIZE)
	memRoot := s.Memory.MerkleRoot()
	out = append(out, memRoot[:]...)
	out = append(out, s.PreimageKey[:]...)
	out = binary.BigEndian.AppendUint32(out, s.PreimageOffset)
	out = binary.BigEndian.AppendUint32(out, s.Heap)
	out = append(out, s.ExitCode)
	out = AppendBoolToWitness(out, s.Exited)

	out = binary.BigEndian.AppendUint64(out, s.Step)
	out = binary.BigEndian.AppendUint64(out, s.StepsSinceLastContextSwitch)
	out = binary.BigEndian.AppendUint32(out, s.Wakeup)

	leftStackRoot := s.getLeftThreadStackRoot()
	rightStackRoot := s.getRightThreadStackRoot()
	out = AppendBoolToWitness(out, s.TraverseRight)
	out = append(out, (leftStackRoot)[:]...)
	out = append(out, (rightStackRoot)[:]...)
	out = binary.BigEndian.AppendUint32(out, s.NextThreadId)

	return out, mtStateHashFromWitness(out)
}

type MTStateWitness []byte

func (sw MTStateWitness) StateHash() (common.Hash, error) {
	if len(sw) != MT_STATE_WITNESS_SIZE {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected %d", len(sw), MT_STATE_WITNESS_SIZE)
	}
	return mtStateHashFromWitness(sw), nil
}

func mtStateHashFromWitness(sw []byte) common.Hash {
	if len(sw) != MT_STATE_WITNESS_SIZE {
		panic("Invalid witness length")
	}
	hash := crypto.Keccak256Hash(sw)
	exitCode := sw[EXITCODE_MT_WITNESS_OFFSET]
	exited := sw[EXITED_MT_WITNESS_OFFSET]
	status := vmStatus(exited == 1, exitCode)
	hash[0] = status
	return hash
}
