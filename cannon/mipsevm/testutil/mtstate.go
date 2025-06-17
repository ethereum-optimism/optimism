package testutil

import (
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
)

func GetMtState(t require.TestingT, vm mipsevm.FPVM) *multithreaded.State {
	return ToMTState(t, vm.GetState())
}

func ToMTState(t require.TestingT, state mipsevm.FPVMState) *multithreaded.State {
	mtState, ok := state.(*multithreaded.State)
	if !ok {
		require.Fail(t, "Failed to cast FPVMState to multithreaded State type")
	}
	return mtState
}

func RandomState(seed int) *multithreaded.State {
	state := multithreaded.CreateEmptyState()
	mut := StateMutatorMultiThreaded{state}
	mut.Randomize(int64(seed))
	return state
}
