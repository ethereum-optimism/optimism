package testutil

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
)

func AddHintLengthPrefix(data []byte) []byte {
	dataLen := len(data)
	prefixed := make([]byte, 0, dataLen+4)
	prefixed = binary.BigEndian.AppendUint32(prefixed, uint32(dataLen))
	prefixed = append(prefixed, data...)

	return prefixed
}

func AddPreimageLengthPrefix(data []byte) []byte {
	dataLen := len(data)
	prefixed := make([]byte, 0, dataLen+8)
	prefixed = binary.BigEndian.AppendUint64(prefixed, uint64(dataLen))
	prefixed = append(prefixed, data...)

	return prefixed
}

type StateMutator interface {
	SetPreimageKey(val common.Hash)
	SetPreimageOffset(val arch.Word)
	SetPC(val arch.Word)
	SetNextPC(val arch.Word)
	SetHI(val arch.Word)
	SetLO(val arch.Word)
	SetHeap(addr arch.Word)
	SetExitCode(val uint8)
	SetExited(val bool)
	SetStep(val uint64)
	SetLastHint(val hexutil.Bytes)
	Randomize(randSeed int64)
}

type StateOption func(state StateMutator)

func WithPC(pc arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetPC(pc)
	}
}

func WithNextPC(nextPC arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetNextPC(nextPC)
	}
}

func WithPCAndNextPC(pc arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetPC(pc)
		state.SetNextPC(pc + 4)
	}
}

func WithHI(hi arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetHI(hi)
	}
}

func WithLO(lo arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetLO(lo)
	}
}

func WithHeap(addr arch.Word) StateOption {
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

func WithPreimageOffset(offset arch.Word) StateOption {
	return func(state StateMutator) {
		state.SetPreimageOffset(offset)
	}
}

func WithStep(step uint64) StateOption {
	return func(state StateMutator) {
		state.SetStep(step)
	}
}

func WithRandomization(seed int64) StateOption {
	return func(mut StateMutator) {
		mut.Randomize(seed)
	}
}

func AlignPC(pc arch.Word) arch.Word {
	// Memory-align random pc and leave room for nextPC
	pc = pc & arch.AddressMask // Align address
	if pc >= arch.AddressMask {
		// Leave room to set and then increment nextPC
		pc = arch.AddressMask - 8
	}
	return pc
}

func BoundStep(step uint64) uint64 {
	// Leave room to increment step at least once
	if step == ^uint64(0) {
		step -= 1
	}
	return step
}

func NewExpectedState(t require.TestingT, fromState mipsevm.FPVMState) *ExpectedMTState {
	mtState := ToMTState(t, fromState)
	return NewExpectedMTState(mtState)
}
