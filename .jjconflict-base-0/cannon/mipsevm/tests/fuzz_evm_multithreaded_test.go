package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/register"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func FuzzStateSyscallCloneMT(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		nextThreadId Word
		stackPtr     Word
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		// Update existing threads to avoid collision with nextThreadId
		if mtutil.FindThread(state, c.nextThreadId) != nil {
			for i, t := range mtutil.GetAllThreads(state) {
				t.ThreadId = c.nextThreadId - Word(i+1)
			}
		}

		state.NextThreadId = c.nextThreadId
		testutil.StoreInstruction(state.GetMemory(), state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysClone
		state.GetRegistersRef()[4] = exec.ValidCloneFlags
		state.GetRegistersRef()[5] = c.stackPtr
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.Step += 1
		// Set original thread expectations
		prestateNextPC := expected.PrestateActiveThread().NextPC
		expected.PrestateActiveThread().PC = prestateNextPC
		expected.PrestateActiveThread().NextPC = prestateNextPC + 4
		expected.PrestateActiveThread().Registers[2] = c.nextThreadId
		expected.PrestateActiveThread().Registers[7] = 0
		// Set expectations for new, cloned thread
		expectedNewThread := expected.ExpectNewThread()
		expectedNewThread.PC = prestateNextPC
		expectedNewThread.NextPC = prestateNextPC + 4
		expectedNewThread.Registers[register.RegSyscallNum] = 0
		expectedNewThread.Registers[register.RegSyscallErrno] = 0
		expectedNewThread.Registers[register.RegSP] = c.stackPtr
		expected.ExpectActiveThreadId(c.nextThreadId)
		expected.ExpectNextThreadId(c.nextThreadId + 1)
		expected.ExpectContextSwitch()
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, nextThreadId, stackPtr Word, seed int64) {
		tests := []testCase{{nextThreadId, stackPtr}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed)...)
	})
}
