package singlethreaded

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
)

// STATE_WITNESS_SIZE is the size of the state witness encoding in bytes.
const STATE_WITNESS_SIZE = 226

type State struct {
	Memory *memory.Memory `json:"memory"`

	PreimageKey    common.Hash `json:"preimageKey"`
	PreimageOffset uint32      `json:"preimageOffset"` // note that the offset includes the 8-byte length prefix

	Cpu mipsevm.CpuScalars `json:"cpu"`

	Heap uint32 `json:"heap"` // to handle mmap growth

	ExitCode uint8 `json:"exit"`
	Exited   bool  `json:"exited"`

	Step uint64 `json:"step"`

	Registers [32]uint32 `json:"registers"`

	// LastHint is optional metadata, and not part of the VM state itself.
	LastHint hexutil.Bytes `json:"lastHint,omitempty"`
}

var _ mipsevm.FPVMState = (*State)(nil)

func CreateEmptyState() *State {
	return &State{
		Cpu: mipsevm.CpuScalars{
			PC:     0,
			NextPC: 4,
			LO:     0,
			HI:     0,
		},
		Heap:      0,
		Registers: [32]uint32{},
		Memory:    memory.NewMemory(),
		ExitCode:  0,
		Exited:    false,
		Step:      0,
	}
}

func CreateInitialState(pc, heapStart uint32) *State {
	state := CreateEmptyState()
	state.Cpu.PC = pc
	state.Cpu.NextPC = pc + 4
	state.Heap = heapStart

	return state
}

type stateMarshaling struct {
	Memory         *memory.Memory `json:"memory"`
	PreimageKey    common.Hash    `json:"preimageKey"`
	PreimageOffset uint32         `json:"preimageOffset"`
	PC             uint32         `json:"pc"`
	NextPC         uint32         `json:"nextPC"`
	LO             uint32         `json:"lo"`
	HI             uint32         `json:"hi"`
	Heap           uint32         `json:"heap"`
	ExitCode       uint8          `json:"exit"`
	Exited         bool           `json:"exited"`
	Step           uint64         `json:"step"`
	Registers      [32]uint32     `json:"registers"`
	LastHint       hexutil.Bytes  `json:"lastHint,omitempty"`
}

func (s *State) MarshalJSON() ([]byte, error) { // nosemgrep
	sm := &stateMarshaling{
		Memory:         s.Memory,
		PreimageKey:    s.PreimageKey,
		PreimageOffset: s.PreimageOffset,
		PC:             s.Cpu.PC,
		NextPC:         s.Cpu.NextPC,
		LO:             s.Cpu.LO,
		HI:             s.Cpu.HI,
		Heap:           s.Heap,
		ExitCode:       s.ExitCode,
		Exited:         s.Exited,
		Step:           s.Step,
		Registers:      s.Registers,
		LastHint:       s.LastHint,
	}
	return json.Marshal(sm)
}

func (s *State) UnmarshalJSON(data []byte) error {
	sm := new(stateMarshaling)
	if err := json.Unmarshal(data, sm); err != nil {
		return err
	}
	s.Memory = sm.Memory
	s.PreimageKey = sm.PreimageKey
	s.PreimageOffset = sm.PreimageOffset
	s.Cpu.PC = sm.PC
	s.Cpu.NextPC = sm.NextPC
	s.Cpu.LO = sm.LO
	s.Cpu.HI = sm.HI
	s.Heap = sm.Heap
	s.ExitCode = sm.ExitCode
	s.Exited = sm.Exited
	s.Step = sm.Step
	s.Registers = sm.Registers
	s.LastHint = sm.LastHint
	return nil
}

func (s *State) GetPC() uint32 { return s.Cpu.PC }

func (s *State) GetCpu() mipsevm.CpuScalars { return s.Cpu }

func (s *State) GetRegistersRef() *[32]uint32 { return &s.Registers }

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
	out = binary.BigEndian.AppendUint32(out, s.Cpu.PC)
	out = binary.BigEndian.AppendUint32(out, s.Cpu.NextPC)
	out = binary.BigEndian.AppendUint32(out, s.Cpu.LO)
	out = binary.BigEndian.AppendUint32(out, s.Cpu.HI)
	out = binary.BigEndian.AppendUint32(out, s.Heap)
	out = append(out, s.ExitCode)
	out = mipsevm.AppendBoolToWitness(out, s.Exited)
	out = binary.BigEndian.AppendUint64(out, s.Step)
	for _, r := range s.Registers {
		out = binary.BigEndian.AppendUint32(out, r)
	}
	return out, stateHashFromWitness(out)
}

func (s *State) Serialize(out io.Writer) error {
	// Write the version byte to the buffer.
	if err := binary.Write(out, binary.BigEndian, uint8(0)); err != nil {
		return err
	}

	// Write memory
	if err := s.Memory.Serialize(out); err != nil {
		return err
	}
	// Write the preimage key as a 32-byte hash
	if _, err := out.Write(s.PreimageKey[:]); err != nil {
		return err
	}
	// Write the preimage offset as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.PreimageOffset); err != nil {
		return err
	}
	// Write the PC as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.Cpu.PC); err != nil {
		return err
	}
	// Write the NextPC as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.Cpu.NextPC); err != nil {
		return err
	}
	// Write the LO register as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.Cpu.LO); err != nil {
		return err
	}
	// Write the HI register as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.Cpu.HI); err != nil {
		return err
	}
	// Write the Heap pointer as a big endian uint32
	if err := binary.Write(out, binary.BigEndian, s.Heap); err != nil {
		return err
	}
	// Write the exit code as a single byte
	if err := binary.Write(out, binary.BigEndian, s.ExitCode); err != nil {
		return err
	}
	// Write the exited flag as a single byte
	var exited uint8
	if s.Exited {
		exited = 1
	}
	if err := binary.Write(out, binary.BigEndian, exited); err != nil {
		return err
	}
	// Write the step counter as a big endian uint64
	if err := binary.Write(out, binary.BigEndian, s.Step); err != nil {
		return err
	}
	// Write the registers as big endian uint32s
	for _, r := range s.Registers {
		if err := binary.Write(out, binary.BigEndian, r); err != nil {
			return err
		}
	}

	// Write the length of the last hint as a big endian uint32.
	// Note that the length is set to 0 even if the hint is nil.
	if s.LastHint == nil {
		if err := binary.Write(out, binary.BigEndian, uint32(0)); err != nil {
			return err
		}
	} else {
		if err := binary.Write(out, binary.BigEndian, uint32(len(s.LastHint))); err != nil {
			return err
		}

		n, err := out.Write(s.LastHint)
		if err != nil {
			return err
		}
		if n != len(s.LastHint) {
			panic("failed to write full last hint")
		}
	}

	return nil
}

func (s *State) Deserialize(in io.Reader) error {
	// Read the version byte from the buffer.
	var version uint8
	if err := binary.Read(in, binary.BigEndian, &version); err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("invalid state encoding version %d", version)
	}
	s.Memory = memory.NewMemory()
	if err := s.Memory.Deserialize(in); err != nil {
		return err
	}
	// Read the preimage key as a 32-byte hash
	if _, err := io.ReadFull(in, s.PreimageKey[:]); err != nil {
		return err
	}
	// Read the preimage offset as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.PreimageOffset); err != nil {
		return err
	}
	// Read the PC as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.Cpu.PC); err != nil {
		return err
	}
	// Read the NextPC as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.Cpu.NextPC); err != nil {
		return err
	}
	// Read the LO register as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.Cpu.LO); err != nil {
		return err
	}
	// Read the HI register as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.Cpu.HI); err != nil {
		return err
	}
	// Read the Heap pointer as a big endian uint32
	if err := binary.Read(in, binary.BigEndian, &s.Heap); err != nil {
		return err
	}
	// Read the exit code as a single byte
	var exitCode uint8
	if err := binary.Read(in, binary.BigEndian, &exitCode); err != nil {
		return err
	}
	s.ExitCode = exitCode
	// Read the exited flag as a single byte
	var exited uint8
	if err := binary.Read(in, binary.BigEndian, &exited); err != nil {
		return err
	}
	if exited == 1 {
		s.Exited = true
	} else {
		s.Exited = false
	}
	// Read the step counter as a big endian uint64
	if err := binary.Read(in, binary.BigEndian, &s.Step); err != nil {
		return err
	}
	// Read the registers as big endian uint32s
	for i := range s.Registers {
		if err := binary.Read(in, binary.BigEndian, &s.Registers[i]); err != nil {
			return err
		}
	}

	// Read the length of the last hint as a big endian uint32.
	// Note that the length is set to 0 even if the hint is nil.
	var lastHintLen uint32
	if err := binary.Read(in, binary.BigEndian, &lastHintLen); err != nil {
		return err
	}
	if lastHintLen > 0 {
		lastHint := make([]byte, lastHintLen)
		n, err := in.Read(lastHint)
		if err != nil {
			return err
		}
		if n != int(lastHintLen) {
			panic("failed to read full last hint")
		}
		s.LastHint = lastHint
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
	offset := 32*2 + 4*6
	exitCode := sw[offset]
	exited := sw[offset+1]
	status := mipsevm.VmStatus(exited == 1, exitCode)
	hash[0] = status
	return hash
}
