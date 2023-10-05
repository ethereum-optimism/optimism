package mipsevm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// StateWitnessSize is the size of the state witness encoding in bytes.
var StateWitnessSize = 226

type State struct {
	Memory *Memory

	PreimageKey    common.Hash
	PreimageOffset uint32 // note that the offset includes the 8-byte length prefix

	PC     uint32
	NextPC uint32
	LO     uint32
	HI     uint32
	Heap   uint32 // to handle mmap growth

	ExitCode uint8
	Exited   bool

	Step uint64

	Registers [32]uint32

	// LastHint is optional metadata, and not part of the VM state itself.
	// It is used to remember the last pre-image hint,
	// so a VM can start from any state without fetching prior pre-images,
	// and instead just repeat the last hint on setup,
	// to make sure pre-image requests can be served.
	// The first 4 bytes are a uin32 length prefix.
	// Warning: the hint MAY NOT BE COMPLETE. I.e. this is buffered,
	// and should only be read when len(LastHint) > 4 && uint32(LastHint[:4]) >= len(LastHint[4:])
	LastHint hexutil.Bytes
}

func (s *State) VMStatus() uint8 {
	return vmStatus(s.Exited, s.ExitCode)
}

func (s *State) EncodeWitness() StateWitness {
	out := make([]byte, 0)
	memRoot := s.Memory.MerkleRoot()
	out = append(out, memRoot[:]...)
	out = append(out, s.PreimageKey[:]...)
	out = binary.BigEndian.AppendUint32(out, s.PreimageOffset)
	out = binary.BigEndian.AppendUint32(out, s.PC)
	out = binary.BigEndian.AppendUint32(out, s.NextPC)
	out = binary.BigEndian.AppendUint32(out, s.LO)
	out = binary.BigEndian.AppendUint32(out, s.HI)
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

func (s *State) Serialize(out io.Writer) error {
	// Write the version byte to the buffer.
	if err := binary.Write(out, binary.BigEndian, uint8(0)); err != nil {
		return err
	}

	// Write the memory to the buffer.
	serMemBuf := new(bytes.Buffer)
	err := s.Memory.Serialize(serMemBuf)
	if err != nil {
		return err
	}
	serMemBytes := serMemBuf.Bytes()
	serMemLen := uint32(len(serMemBytes))
	// Write the length of the memory as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, serMemLen); err != nil {
		return err
	}
	// Write the memory itself
	if _, err = out.Write(serMemBytes); err != nil {
		return err
	}
	// Write the preimage key as a 32-byte hash
	if _, err = out.Write(s.PreimageKey[:]); err != nil {
		return err
	}
	// Write the preimage offset as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.PreimageOffset); err != nil {
		return err
	}
	// Write the PC as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.PC); err != nil {
		return err
	}
	// Write the NextPC as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.NextPC); err != nil {
		return err
	}
	// Write the LO register as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.LO); err != nil {
		return err
	}
	// Write the HI register as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.HI); err != nil {
		return err
	}
	// Write the Heap pointer as a big endian uint32
	if err = binary.Write(out, binary.BigEndian, s.Heap); err != nil {
		return err
	}
	// Write the exit code as a single byte
	if err = binary.Write(out, binary.BigEndian, s.ExitCode); err != nil {
		return err
	}
	// Write the exited flag as a single byte
	if s.Exited {
		err = binary.Write(out, binary.BigEndian, uint8(1))
	} else {
		err = binary.Write(out, binary.BigEndian, uint8(0))
	}
	if err != nil {
		return err
	}
	// Write the step counter as a big endian uint64
	if err = binary.Write(out, binary.BigEndian, s.Step); err != nil {
		return err
	}
	// Write the registers as big endian uint32s
	for _, r := range s.Registers {
		if err = binary.Write(out, binary.BigEndian, r); err != nil {
			return err
		}
	}

	// Write the length of the last hint as a big endian uint32.
	// Note that the length is set to 0 even if the hint is nil.
	if s.LastHint == nil {
		if err = binary.Write(out, binary.BigEndian, uint32(0)); err != nil {
			return err
		}
	} else {
		if err = binary.Write(out, binary.BigEndian, uint32(len(s.LastHint))); err != nil {
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

	// Read the length of the memory as a big endian uint32
	var serMemLen uint32
	if err := binary.Read(in, binary.BigEndian, &serMemLen); err != nil {
		return err
	}
	// Read the memory itself
	serMem := make([]byte, serMemLen)
	n, err := io.ReadFull(in, serMem)
	if err != nil {
		return err
	}
	if n != int(serMemLen) {
		return fmt.Errorf("failed to read full memory, read %d bytes, expected %d", n, serMemLen)
	}
	memory := NewMemory()
	err = memory.Deserialize(bytes.NewReader(serMem))
	if err != nil {
		return err
	}
	s.Memory = memory

	// Read the preimage key as a 32-byte hash
	n, err = in.Read(s.PreimageKey[:])
	if err != nil {
		return err
	}
	if n != 32 {
		return fmt.Errorf("failed to read full preimage key")
	}
	// Read the preimage offset as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.PreimageOffset); err != nil {
		return err
	}
	// Read the PC as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.PC); err != nil {
		return err
	}
	// Read the NextPC as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.NextPC); err != nil {
		return err
	}
	// Read the LO register as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.LO); err != nil {
		return err
	}
	// Read the HI register as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.HI); err != nil {
		return err
	}
	// Read the Heap pointer as a big endian uint32
	if err = binary.Read(in, binary.BigEndian, &s.Heap); err != nil {
		return err
	}
	// Read the exit code as a single byte
	var exitCode uint8
	if err = binary.Read(in, binary.BigEndian, &exitCode); err != nil {
		return err
	}
	s.ExitCode = exitCode
	// Read the exited flag as a single byte
	var exited uint8
	if err = binary.Read(in, binary.BigEndian, &exited); err != nil {
		return err
	}
	if exited == 1 {
		s.Exited = true
	} else {
		s.Exited = false
	}
	// Read the step counter as a big endian uint64
	if err = binary.Read(in, binary.BigEndian, &s.Step); err != nil {
		return err
	}
	// Read the registers as big endian uint32s
	for i := range s.Registers {
		if err = binary.Read(in, binary.BigEndian, &s.Registers[i]); err != nil {
			return err
		}
	}

	// Read the length of the last hint as a big endian uint32.
	// Note that the length is set to 0 even if the hint is nil.
	var lastHintLen uint32
	if err = binary.Read(in, binary.BigEndian, &lastHintLen); err != nil {
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

const (
	VMStatusValid      = 0
	VMStatusInvalid    = 1
	VMStatusPanic      = 2
	VMStatusUnfinished = 3
)

func (sw StateWitness) StateHash() (common.Hash, error) {
	if len(sw) != 226 {
		return common.Hash{}, fmt.Errorf("Invalid witness length. Got %d, expected at least 88", len(sw))
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
