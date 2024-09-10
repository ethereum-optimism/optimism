package testutil

import (
	"fmt"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
)

func CopyRegisters(state mipsevm.FPVMState) *[32]uint64 {
	copy := new([32]uint64)
	*copy = *state.GetRegistersRef()
	return copy
}

type StateMutator interface {
	SetPreimageKey(val common.Hash)
	SetPreimageOffset(val uint64)
	SetPC(val uint64)
	SetNextPC(val uint64)
	SetHI(val uint64)
	SetLO(val uint64)
	SetHeap(addr uint64)
	SetExitCode(val uint8)
	SetExited(val bool)
	SetStep(val uint64)
	SetLastHint(val hexutil.Bytes)
	Randomize(randSeed int64)
}

type StateOption func(state StateMutator)

func WithPC(pc uint64) StateOption {
	return func(state StateMutator) {
		state.SetPC(pc)
	}
}

func WithNextPC(nextPC uint64) StateOption {
	return func(state StateMutator) {
		state.SetNextPC(nextPC)
	}
}

func WithHeap(addr uint64) StateOption {
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

func WithPreimageOffset(offset uint64) StateOption {
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

func AlignPC(pc uint64) uint64 {
	// Memory-align random pc and leave room for nextPC
	pc = pc & 0xFFFF_FFFF_FFFF_FFF8 // Align address
	if pc >= 0xFFFF_FFFF_FFFF_FFF8 {
		// Leave room to set and then increment nextPC
		pc = 0xFFFF_FFFF_FFFF_FFF8 - 8
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

type ExpectedState struct {
	PreimageKey    common.Hash
	PreimageOffset uint64
	PC             uint64
	NextPC         uint64
	HI             uint64
	LO             uint64
	Heap           uint64
	ExitCode       uint8
	Exited         bool
	Step           uint64
	LastHint       hexutil.Bytes
	Registers      [32]uint64
	MemoryRoot     common.Hash
}

func NewExpectedState(fromState mipsevm.FPVMState) *ExpectedState {
	return &ExpectedState{
		PreimageKey:    fromState.GetPreimageKey(),
		PreimageOffset: fromState.GetPreimageOffset(),
		PC:             fromState.GetPC(),
		NextPC:         fromState.GetCpu().NextPC,
		HI:             fromState.GetCpu().HI,
		LO:             fromState.GetCpu().LO,
		Heap:           fromState.GetHeap(),
		ExitCode:       fromState.GetExitCode(),
		Exited:         fromState.GetExited(),
		Step:           fromState.GetStep(),
		LastHint:       fromState.GetLastHint(),
		Registers:      *fromState.GetRegistersRef(),
		MemoryRoot:     fromState.GetMemory().MerkleRoot(),
	}
}

type StateValidationFlags int

// TODO(cp-983) - Remove these validation hacks
const (
	SkipMemoryValidation StateValidationFlags = iota
	SkipHintValidation
	SkipPreimageKeyValidation
)

func (e *ExpectedState) Validate(t testing.TB, actualState mipsevm.FPVMState, flags ...StateValidationFlags) {
	if !slices.Contains(flags, SkipPreimageKeyValidation) {
		require.Equal(t, e.PreimageKey, actualState.GetPreimageKey(), fmt.Sprintf("Expect preimageKey = %v", e.PreimageKey))
	}
	require.Equal(t, e.PreimageOffset, actualState.GetPreimageOffset(), fmt.Sprintf("Expect preimageOffset = %v", e.PreimageOffset))
	require.Equal(t, e.PC, actualState.GetCpu().PC, fmt.Sprintf("Expect PC = 0x%x", e.PC))
	require.Equal(t, e.NextPC, actualState.GetCpu().NextPC, fmt.Sprintf("Expect nextPC = 0x%x", e.NextPC))
	require.Equal(t, e.HI, actualState.GetCpu().HI, fmt.Sprintf("Expect HI = 0x%x", e.HI))
	require.Equal(t, e.LO, actualState.GetCpu().LO, fmt.Sprintf("Expect LO = 0x%x", e.LO))
	require.Equal(t, e.Heap, actualState.GetHeap(), fmt.Sprintf("Expect heap = 0x%x", e.Heap))
	require.Equal(t, e.ExitCode, actualState.GetExitCode(), fmt.Sprintf("Expect exitCode = 0x%x", e.ExitCode))
	require.Equal(t, e.Exited, actualState.GetExited(), fmt.Sprintf("Expect exited = %v", e.Exited))
	require.Equal(t, e.Step, actualState.GetStep(), fmt.Sprintf("Expect step = %d", e.Step))
	if !slices.Contains(flags, SkipHintValidation) {
		require.Equal(t, e.LastHint, actualState.GetLastHint(), fmt.Sprintf("Expect lastHint = %v", e.LastHint))
	}
	require.Equal(t, e.Registers, *actualState.GetRegistersRef(), fmt.Sprintf("Expect registers = %v", e.Registers))
	if !slices.Contains(flags, SkipMemoryValidation) {
		require.Equal(t, e.MemoryRoot, common.Hash(actualState.GetMemory().MerkleRoot()), fmt.Sprintf("Expect memory root = %v", e.MemoryRoot))
	}
}
