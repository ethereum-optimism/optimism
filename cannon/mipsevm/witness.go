package mipsevm

import (
	"encoding/binary"
	"errors"
	"fmt"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

type StepWitness struct {
	// encoded state witness
	State []byte

	MemProof []byte

	PreimageKey    [32]byte // zeroed when no pre-image is accessed
	PreimageValue  []byte   // including the 8-byte length prefix
	PreimageOffset uint32
}

func uint32ToBytes32(v uint32) []byte {
	var out [32]byte
	binary.BigEndian.PutUint32(out[32-4:], v)
	return out[:]
}

func (wit *StepWitness) EncodeStepInput() []byte {
	var input []byte
	input = append(input, StepBytes4...)
	input = append(input, uint32ToBytes32(32*2)...)                           // state data offset in bytes
	input = append(input, uint32ToBytes32(32*2+32+uint32(len(wit.State)))...) // proof data offset in bytes

	input = append(input, uint32ToBytes32(uint32(len(wit.State)))...) // state data length in bytes
	input = append(input, wit.State[:]...)
	input = append(input, uint32ToBytes32(uint32(len(wit.MemProof)))...) // proof data length in bytes
	input = append(input, wit.MemProof[:]...)
	return input
}

func (wit *StepWitness) HasPreimage() bool {
	return wit.PreimageKey != ([32]byte{})
}

func (wit *StepWitness) EncodePreimageOracleInput() ([]byte, error) {
	if wit.PreimageKey == ([32]byte{}) {
		return nil, errors.New("cannot encode pre-image oracle input, witness has no pre-image to proof")
	}

	switch preimage.KeyType(wit.PreimageKey[0]) {
	case preimage.LocalKeyType:
		if len(wit.PreimageValue) > 32+8 {
			return nil, fmt.Errorf("local pre-image exceeds maximum size of 32 bytes with key 0x%x", wit.PreimageKey)
		}
		var input []byte
		input = append(input, LoadLocalDataBytes4...)
		input = append(input, wit.PreimageKey[:]...)

		preimagePart := wit.PreimageValue[8:]
		var tmp [32]byte
		copy(tmp[:], preimagePart)
		input = append(input, tmp[:]...)
		input = append(input, uint32ToBytes32(uint32(len(wit.PreimageValue)-8))...)
		input = append(input, uint32ToBytes32(wit.PreimageOffset)...)
		// Note: we can pad calldata to 32 byte multiple, but don't strictly have to
		return input, nil
	case preimage.Keccak256KeyType:
		var input []byte
		input = append(input, LoadKeccak256PreimagePartBytes4...)
		input = append(input, uint32ToBytes32(wit.PreimageOffset)...)
		input = append(input, uint32ToBytes32(32+32)...) // partOffset, calldata offset
		input = append(input, uint32ToBytes32(uint32(len(wit.PreimageValue))-8)...)
		input = append(input, wit.PreimageValue[8:]...)
		// Note: we can pad calldata to 32 byte multiple, but don't strictly have to
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported pre-image type %d, cannot prepare preimage with key %x offset %d for oracle",
			wit.PreimageKey[0], wit.PreimageKey, wit.PreimageOffset)
	}
}
