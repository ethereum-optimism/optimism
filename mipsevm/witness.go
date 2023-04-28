package mipsevm

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/cannon/preimage"
)

type StepWitness struct {
	state []byte

	memProof []byte

	preimageKey    [32]byte // zeroed when no pre-image is accessed
	preimageValue  []byte   // including the 8-byte length prefix
	preimageOffset uint32
}

func uint32ToBytes32(v uint32) []byte {
	var out [32]byte
	binary.BigEndian.PutUint32(out[32-4:], v)
	return out[:]
}

func (wit *StepWitness) EncodeStepInput() []byte {
	stateHash := crypto.Keccak256Hash(wit.state)
	var input []byte
	input = append(input, StepBytes4...)
	input = append(input, stateHash[:]...)
	input = append(input, uint32ToBytes32(32*3)...)                           // state data offset in bytes
	input = append(input, uint32ToBytes32(32*3+32+uint32(len(wit.state)))...) // proof data offset in bytes

	input = append(input, uint32ToBytes32(uint32(len(wit.state)))...) // state data length in bytes
	input = append(input, wit.state[:]...)
	input = append(input, uint32ToBytes32(uint32(len(wit.memProof)))...) // proof data length in bytes
	input = append(input, wit.memProof[:]...)
	return input
}

func (wit *StepWitness) HasPreimage() bool {
	return wit.preimageKey != ([32]byte{})
}

func (wit *StepWitness) EncodePreimageOracleInput() ([]byte, error) {
	if wit.preimageKey == ([32]byte{}) {
		return nil, errors.New("cannot encode pre-image oracle input, witness has no pre-image to proof")
	}

	switch preimage.KeyType(wit.preimageKey[0]) {
	case preimage.LocalKeyType:
		// We have no on-chain form of preparing the bootstrap pre-images onchain yet.
		// So instead we cheat them in.
		// In production usage there should be an on-chain contract that exposes this,
		// rather than going through the global keccak256 oracle.
		var input []byte
		input = append(input, CheatBytes4...)
		input = append(input, uint32ToBytes32(wit.preimageOffset)...)
		input = append(input, wit.preimageKey[:]...)
		var tmp [32]byte
		copy(tmp[:], wit.preimageValue[wit.preimageOffset:])
		input = append(input, tmp[:]...)
		input = append(input, uint32ToBytes32(uint32(len(wit.preimageValue))-8)...)
		// TODO: do we want to pad the end to a multiple of 32 bytes?
		return input, nil
	case preimage.Keccak256KeyType:
		var input []byte
		input = append(input, LoadKeccak256PreimagePartBytes4...)
		input = append(input, uint32ToBytes32(wit.preimageOffset)...)
		input = append(input, uint32ToBytes32(32+32)...) // partOffset, calldata offset
		input = append(input, uint32ToBytes32(uint32(len(wit.preimageValue))-8)...)
		input = append(input, wit.preimageValue[8:]...)
		// TODO: do we want to pad the end to a multiple of 32 bytes?
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported pre-image type %d, cannot prepare preimage with key %x offset %d for oracle",
			wit.preimageKey[0], wit.preimageKey, wit.preimageOffset)
	}
}
