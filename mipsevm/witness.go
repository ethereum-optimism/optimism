package mipsevm

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/crypto"
)

type StepWitness struct {
	state []byte

	memProof []byte

	preimageKey    [32]byte // zeroed when no pre-image is accessed
	preimageValue  []byte
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

func (wit *StepWitness) EncodePreimageOracleInput() []byte {
	if wit.preimageKey == ([32]byte{}) {
		return nil
	}
	// TODO
	return nil
}
