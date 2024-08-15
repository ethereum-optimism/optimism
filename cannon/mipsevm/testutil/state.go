package testutil

import (
	"math/rand"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func RandomRegisters(seed int64) [32]uint32 {
	r := rand.New(rand.NewSource(seed))
	var registers [32]uint32
	for i := 0; i < 32; i++ {
		registers[i] = r.Uint32()
	}
	return registers
}

func CopyRegisters(state mipsevm.FPVMState) *[32]uint32 {
	copy := new([32]uint32)
	*copy = *state.GetRegistersRef()
	return copy
}

type StateMutator interface {
	SetPreimageKey(val common.Hash)
	SetPreimageOffset(val uint32)
	SetPC(val uint32)
	SetNextPC(val uint32)
	SetHI(val uint32)
	SetLO(val uint32)
	SetHeap(addr uint32)
	SetExitCode(val uint8)
	SetExited(val bool)
	SetStep(val uint64)
	SetLastHint(val hexutil.Bytes)
}

type StateOption func(state StateMutator)

func WithPC(pc uint32) StateOption {
	return func(state StateMutator) {
		state.SetPC(pc)
	}
}

func WithNextPC(nextPC uint32) StateOption {
	return func(state StateMutator) {
		state.SetNextPC(nextPC)
	}
}

func WithHeap(addr uint32) StateOption {
	return func(state StateMutator) {
		state.SetHeap(addr)
	}
}

func WithLastHint(lastHint hexutil.Bytes) StateOption {
	return func(state StateMutator) {
		state.SetLastHint(lastHint)
	}
}

func WithPreimageKey(key common.Hash) StateOption {
	return func(state StateMutator) {
		state.SetPreimageKey(key)
	}
}

func WithPreimageOffset(offset uint32) StateOption {
	return func(state StateMutator) {
		state.SetPreimageOffset(offset)
	}
}

func WithStep(step uint64) StateOption {
	return func(state StateMutator) {
		state.SetStep(step)
	}
}
