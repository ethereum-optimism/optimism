package multithreaded

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"

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
	Memory *memory.Memory

	PreimageKey    common.Hash
	PreimageOffset uint32 // note that the offset includes the 8-byte length prefix

	Heap uint32 // to handle mmap growth

	ExitCode uint8
	Exited   bool

	Step                        uint64
	StepsSinceLastContextSwitch uint64
	Wakeup                      uint32

	TraverseRight    bool
	LeftThreadStack  []*ThreadState
	RightThreadStack []*ThreadState
	NextThreadId     uint32

	// LastHint is optional metadata, and not part of the VM state itself.
	LastHint hexutil.Bytes
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

func (s *State) CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, meta mipsevm.Metadata) mipsevm.FPVM {
	logger.Info("Using cannon multithreaded VM")
	return NewInstrumentedState(s, po, stdOut, stdErr, logger, meta)
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

// Serialize writes the state in a simple binary format which can be read again using Deserialize
// The format is a simple concatenation of fields, with prefixed item count for repeating items and using big endian
// encoding for numbers.
//
// StateVersion                uint8(1)
// Memory                      As per Memory.Serialize
// PreimageKey                 [32]byte
// PreimageOffset              uint32
// Heap                        uint32
// ExitCode                    uint8
// Exited                      uint8 - 0 for false, 1 for true
// Step                        uint64
// StepsSinceLastContextSwitch uint64
// Wakeup                      uint32
// TraverseRight               uint8 - 0 for false, 1 for true
// NextThreadId                uint32
// len(LeftThreadStack)        uint32
// LeftThreadStack entries     as per ThreadState.Serialize
// len(RightThreadStack)       uint32
// RightThreadStack entries    as per ThreadState.Serialize
// len(LastHint)			   uint32 (0 when LastHint is nil)
// LastHint 				   []byte
func (s *State) Serialize(out io.Writer) error {
	bout := serialize.NewBinaryWriter(out)

	if err := s.Memory.Serialize(out); err != nil {
		return err
	}
	if err := bout.WriteHash(s.PreimageKey); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.PreimageOffset); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.Heap); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.ExitCode); err != nil {
		return err
	}
	if err := bout.WriteBool(s.Exited); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.Step); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.StepsSinceLastContextSwitch); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.Wakeup); err != nil {
		return err
	}
	if err := bout.WriteBool(s.TraverseRight); err != nil {
		return err
	}
	if err := bout.WriteUInt(s.NextThreadId); err != nil {
		return err
	}

	if err := bout.WriteUInt(uint32(len(s.LeftThreadStack))); err != nil {
		return err
	}
	for _, stack := range s.LeftThreadStack {
		if err := stack.Serialize(out); err != nil {
			return err
		}
	}
	if err := bout.WriteUInt(uint32(len(s.RightThreadStack))); err != nil {
		return err
	}
	for _, stack := range s.RightThreadStack {
		if err := stack.Serialize(out); err != nil {
			return err
		}
	}
	if err := bout.WriteBytes(s.LastHint); err != nil {
		return err
	}

	return nil
}

func (s *State) Deserialize(in io.Reader) error {
	bin := serialize.NewBinaryReader(in)
	s.Memory = memory.NewMemory()
	if err := s.Memory.Deserialize(in); err != nil {
		return err
	}
	if err := bin.ReadHash(&s.PreimageKey); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.PreimageOffset); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.Heap); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.ExitCode); err != nil {
		return err
	}
	if err := bin.ReadBool(&s.Exited); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.Step); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.StepsSinceLastContextSwitch); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.Wakeup); err != nil {
		return err
	}
	if err := bin.ReadBool(&s.TraverseRight); err != nil {
		return err
	}

	if err := bin.ReadUInt(&s.NextThreadId); err != nil {
		return err
	}

	var leftThreadStackSize uint32
	if err := bin.ReadUInt(&leftThreadStackSize); err != nil {
		return err
	}
	s.LeftThreadStack = make([]*ThreadState, leftThreadStackSize)
	for i := range s.LeftThreadStack {
		s.LeftThreadStack[i] = &ThreadState{}
		if err := s.LeftThreadStack[i].Deserialize(in); err != nil {
			return err
		}
	}

	var rightThreadStackSize uint32
	if err := bin.ReadUInt(&rightThreadStackSize); err != nil {
		return err
	}
	s.RightThreadStack = make([]*ThreadState, rightThreadStackSize)
	for i := range s.RightThreadStack {
		s.RightThreadStack[i] = &ThreadState{}
		if err := s.RightThreadStack[i].Deserialize(in); err != nil {
			return err
		}
	}

	if err := bin.ReadBytes((*[]byte)(&s.LastHint)); err != nil {
		return err
	}
	return nil
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
