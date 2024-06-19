package mipsevm

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// StateWitnessSize is the size of the state witness encoding in bytes.
var StateWitnessSize = 226

type CpuScalars struct {
	PC     uint32 `json:"pc"`
	NextPC uint32 `json:"nextPC"`
	LO     uint32 `json:"lo"`
	HI     uint32 `json:"hi"`
}

type State struct {
	Memory *Memory `json:"memory"`

	PreimageKey    common.Hash `json:"preimageKey"`
	PreimageOffset uint32      `json:"preimageOffset"` // note that the offset includes the 8-byte length prefix

	Cpu CpuScalars `json:"cpu"`

	Heap uint32 `json:"heap"` // to handle mmap growth

	ExitCode uint8 `json:"exit"`
	Exited   bool  `json:"exited"`

	Step uint64 `json:"step"`

	Registers [32]uint32 `json:"registers"`

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

type stateMarshaling struct {
	Memory         *Memory       `json:"memory"`
	PreimageKey    common.Hash   `json:"preimageKey"`
	PreimageOffset uint32        `json:"preimageOffset"`
	PC             uint32        `json:"pc"`
	NextPC         uint32        `json:"nextPC"`
	LO             uint32        `json:"lo"`
	HI             uint32        `json:"hi"`
	Heap           uint32        `json:"heap"`
	ExitCode       uint8         `json:"exit"`
	Exited         bool          `json:"exited"`
	Step           uint64        `json:"step"`
	Registers      [32]uint32    `json:"registers"`
	LastHint       hexutil.Bytes `json:"lastHint,omitempty"`
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

func (s *State) GetStep() uint64 { return s.Step }

func (s *State) VMStatus() uint8 {
	return vmStatus(s.Exited, s.ExitCode)
}

func (s *State) EncodeWitness() StateWitness {
	out := make([]byte, 0)
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
	if s.Exited {
		out = append(out, 1)
	} else {
		out = append(out, 0)
	}
	out = binary.BigEndian.AppendUint64(out, s.Step)
	for _, r := range s.Registers {
		out = binary.BigEndian.AppendUint32(out, r)
	}
	return out
}

type StateWitness []byte

const (
	VMStatusValid      = 0
	VMStatusInvalid    = 1
	VMStatusPanic      = 2
	VMStatusUnfinished = 3
)

func (sw StateWitness) StateHash() (common.Hash, error) {
	if len(sw) != 226 {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected 226", len(sw))
	}

	hash := crypto.Keccak256Hash(sw)
	offset := 32*2 + 4*6
	exitCode := sw[offset]
	exited := sw[offset+1]
	status := vmStatus(exited == 1, exitCode)
	hash[0] = status
	return hash, nil
}

func vmStatus(exited bool, exitCode uint8) uint8 {
	if !exited {
		return VMStatusUnfinished
	}

	switch exitCode {
	case 0:
		return VMStatusValid
	case 1:
		return VMStatusInvalid
	default:
		return VMStatusPanic
	}
}
