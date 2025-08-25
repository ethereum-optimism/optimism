package multithreaded

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestInstrumentedState_Hello(t *testing.T) {
	runTestAcrossVms(t, "Hello", func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath("hello", goTarget), CreateInitialState)

		var stdOutBuf, stdErrBuf bytes.Buffer
		us := vmFactory(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)

		maxSteps := 500_000
		for i := 0; i < maxSteps; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
		}

		require.Truef(t, state.GetExited(), "must complete program. reached %d of max %d steps", state.GetStep(), maxSteps)
		require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

		require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
		require.Equal(t, "", stdErrBuf.String(), "stderr silent")
	})
}

func TestInstrumentedState_Claim(t *testing.T) {
	runTestAcrossVms(t, "Claim", func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath("claim", goTarget), CreateInitialState)

		oracle, expectedStdOut, expectedStdErr := testutil.ClaimTestOracle(t)

		var stdOutBuf, stdErrBuf bytes.Buffer
		us := vmFactory(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)

		for i := 0; i < 2000_000; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
		}

		require.True(t, state.GetExited(), "must complete program")
		require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

		require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
		require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
	})
}

func TestInstrumentedState_Random(t *testing.T) {
	state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath("random", testutil.Go1_24), CreateInitialState)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := latestVm(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)

	for i := 0; i < 500_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}
	t.Logf("Completed in %d steps", state.Step)

	require.True(t, state.GetExited(), "must complete program")
	require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

	// Check output
	// Define the regex pattern we expect to match against stdOut
	pattern := `Random (hex data|int): (\w+)\s*`
	re, err := regexp.Compile(pattern)
	require.NoError(t, err)

	// Check that stdOut matches the expected regex
	expectedMatches := 3
	output := stdOutBuf.String()
	matches := re.FindAllStringSubmatch(output, -1)
	require.Equal(t, expectedMatches, len(matches))

	// Check each match and validate the random values that are printed to stdOut
	for i := 0; i < expectedMatches; i++ {
		match := matches[i]
		require.Contains(t, match[0], "Random")

		// Check that the generated random number is not zero
		dataType := match[1]
		dataValue := match[2]
		switch dataType {
		case "hex data":
			randVal, success := new(big.Int).SetString(dataValue, 16)
			require.True(t, success, "should successfully set hex value")
			require.NotEqual(t, 0, randVal.Sign(), "random data should be non-zero")
		case "int":
			randVal, err := strconv.ParseUint(dataValue, 10, 64)
			require.NoError(t, err)
			require.NotEqual(t, uint64(0), randVal, "random int should be non-zero")
		}
	}
}

func TestInstrumentedState_SyscallEventFdProgram(t *testing.T) {
	runTestAcrossVms(t, "SyscallEventFdProgram", func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath("syscall-eventfd", goTarget), CreateInitialState)

		var stdOutBuf, stdErrBuf bytes.Buffer
		us := vmFactory(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)
		err := us.InitDebug()
		require.NoError(t, err)

		for i := 0; i < 550_000; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
		}
		t.Logf("Completed in %d steps", state.Step)

		require.True(t, state.GetExited(), "must complete program")
		if state.GetExitCode() != 0 {
			us.Traceback()
		}
		require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

		// Check output
		output := stdOutBuf.String()
		require.Contains(t, output, "call eventfd with valid flags: '0x80080'")
		require.Contains(t, output, "call eventfd with valid flags: '0xFFFFFFFFFFFFFFFF'")
		require.Contains(t, output, "call eventfd with valid flags: '0x80'")
		require.Contains(t, output, "call eventfd with invalid flags: '0x0'")
		require.Contains(t, output, "call eventfd with invalid flags: '0xFFFFFFFFFFFFFF7F'")
		require.Contains(t, output, "call eventfd with invalid flags: '0x80000'")
		require.Contains(t, output, "write to eventfd object")
		require.Contains(t, output, "read from eventfd object")
		require.Contains(t, output, "done")

		// Check fd value
		pattern := `eventfd2 fd = '(.+)'`
		re, err := regexp.Compile(pattern)
		require.NoError(t, err)
		matches := re.FindAllStringSubmatch(output, -1)

		expectedMatches := 3
		require.Equal(t, expectedMatches, len(matches))
		for i := 0; i < expectedMatches; i++ {
			require.Equal(t, "100", matches[i][1])
		}
	})
}

func TestInstrumentedState_UtilsCheck(t *testing.T) {
	// Sanity check that test running utilities will return a non-zero exit code on failure
	type TestCase struct {
		name           string
		expectedOutput string
	}

	cases := []TestCase{
		{name: "utilscheck", expectedOutput: "Test failed: ShouldFail"},
		{name: "utilscheck2", expectedOutput: "Test failed: ShouldFail (subtest 2)"},
		{name: "utilscheck3", expectedOutput: "Test panicked: ShouldFail (panic test)"},
		{name: "utilscheck4", expectedOutput: "Test panicked: ShouldFail"},
	}

	testNamer := func(vm string, testCase TestCase) string {
		return fmt.Sprintf("%v-%v", testCase.name, vm)
	}

	runTestsAcrossVms(t, testNamer, cases, func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget, test TestCase) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath(test.name, goTarget), CreateInitialState)
		oracle := testutil.StaticOracle(t, []byte{})

		var stdOutBuf, stdErrBuf bytes.Buffer
		us := vmFactory(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)

		for i := 0; i < 1_000_000; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
		}
		t.Logf("Completed in %d steps", state.Step)

		require.True(t, state.Exited, "must complete program")
		require.Equal(t, uint8(1), state.ExitCode, "exit with 1")
		require.Contains(t, stdOutBuf.String(), test.expectedOutput)
		require.NotContains(t, stdOutBuf.String(), "Passed test that should have failed")
		require.Equal(t, "", stdErrBuf.String(), "should not print any errors")
	})
}

func TestInstrumentedState_MultithreadedProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	type TestCase struct {
		name           string
		expectedOutput []string
		programName    string
		steps          int
	}

	cases := []TestCase{
		{
			name: "general concurrency test",
			expectedOutput: []string{
				"waitgroup result: 42",
				"channels result: 1234",
				"GC complete!",
			},
			programName: "mt-general",
			steps:       5_000_000,
		},
		{
			name: "atomic test",
			expectedOutput: []string{
				"Atomic tests passed",
			},
			programName: "mt-atomic",
			steps:       350_000_000,
		},
		{
			name: "waitgroup test",
			expectedOutput: []string{
				"WaitGroup tests passed",
			},
			programName: "mt-wg",
			steps:       15_000_000,
		},
		{
			name: "mutex test",
			expectedOutput: []string{
				"Mutex test passed",
			},
			programName: "mt-mutex",
			steps:       5_000_000,
		},
		{
			name: "cond test",
			expectedOutput: []string{
				"Cond test passed",
			},
			programName: "mt-cond",
			steps:       5_000_000,
		},
		{
			name: "rwmutex test",
			expectedOutput: []string{
				"RWMutex test passed",
			},
			programName: "mt-rwmutex",
			steps:       5_000_000,
		},
		{
			name: "once test",
			expectedOutput: []string{
				"Once test passed",
			},
			programName: "mt-once",
			steps:       5_000_000,
		},
		{
			name: "oncefunc test",
			expectedOutput: []string{
				"OnceFunc tests passed",
			},
			programName: "mt-oncefunc",
			steps:       15_000_000,
		},
		{
			name: "map test",
			expectedOutput: []string{
				"Map test passed",
			},
			programName: "mt-map",
			steps:       150_000_000,
		},
		{
			name: "pool test",
			expectedOutput: []string{
				"Pool test passed",
			},
			programName: "mt-pool",
			steps:       50_000_000,
		},
		{
			name: "value test",
			expectedOutput: []string{
				"Value tests passed",
			},
			programName: "mt-value",
			steps:       3_000_000,
		},
	}

	testNamer := func(vm string, testCase TestCase) string {
		return fmt.Sprintf("%v-%v", testCase.name, vm)
	}

	runTestsAcrossVms(t, testNamer, cases, func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget, test TestCase) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath(test.programName, goTarget), CreateInitialState)
		oracle := testutil.StaticOracle(t, []byte{})

		var stdOutBuf, stdErrBuf bytes.Buffer
		us := vmFactory(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger(), meta)
		for i := 0; i < test.steps; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
		}
		t.Logf("Completed in %d steps", state.Step)

		require.True(t, state.Exited, "must complete program")
		require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
		for _, expected := range test.expectedOutput {
			require.Contains(t, stdOutBuf.String(), expected)
		}
		require.Equal(t, "", stdErrBuf.String(), "should not print any errors")
	})
}

func TestInstrumentedState_Alloc(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	const MiB = 1024 * 1024

	type TestCase struct {
		name                string
		numAllocs           int
		allocSize           int
		maxMemoryUsageCheck int
	}

	cases := []TestCase{
		{name: "10 32MiB allocations", numAllocs: 10, allocSize: 32 * MiB, maxMemoryUsageCheck: 256 * MiB},
		{name: "5 64MiB allocations", numAllocs: 5, allocSize: 64 * MiB, maxMemoryUsageCheck: 256 * MiB},
		{name: "5 128MiB allocations", numAllocs: 5, allocSize: 128 * MiB, maxMemoryUsageCheck: 128 * 3 * MiB},
	}

	testNamer := func(vm string, testCase TestCase) string {
		return fmt.Sprintf("%v-%v", testCase.name, vm)
	}

	runTestsAcrossVms(t, testNamer, cases, func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget, test TestCase) {
		state, meta := testutil.LoadELFProgram(t, testutil.ProgramPath("alloc", goTarget), CreateInitialState)
		oracle := testutil.AllocOracle(t, test.numAllocs, test.allocSize)

		us := vmFactory(state, oracle, os.Stdout, os.Stderr, testutil.CreateLogger(), meta)
		require.NoError(t, us.InitDebug())
		// emulation shouldn't take more than 20 B steps
		for i := 0; i < 20_000_000_000; i++ {
			if us.GetState().GetExited() {
				break
			}
			_, err := us.Step(false)
			require.NoError(t, err)
			if state.Step%10_000_000 == 0 {
				t.Logf("Completed %d steps", state.Step)
			}
		}
		memUsage := state.Memory.PageCount() * memory.PageSize
		t.Logf("Completed in %d steps. cannon memory usage: %d KiB", state.Step, memUsage/1024/1024.0)
		require.True(t, state.Exited, "must complete program")
		require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
		require.Less(t, memUsage, test.maxMemoryUsageCheck, "memory allocation is too large")
	})
}

type VMTest func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget)
type VMTestCase[T any] func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget, testCase T)

type TestNamer[T any] func(vmName string, testCase T) string

func runTestAcrossVms(t *testing.T, testName string, vmTest VMTest) {
	testNamer := func(vm string, _ any) string {
		return fmt.Sprintf("%v-%v", testName, vm)
	}

	runTestsAcrossVms[any](t, testNamer, []any{nil}, func(t *testing.T, vmFactory VMFactory[*State], goTarget testutil.GoTarget, _ any) {
		vmTest(t, vmFactory, goTarget)
	})
}

func runTestsAcrossVms[T any](t *testing.T, testNamer TestNamer[T], testCases []T, vmTestCase VMTestCase[T]) {
	t.Parallel()
	type VMVariations struct {
		name     string
		goTarget testutil.GoTarget
		features mipsevm.FeatureToggles
	}

	variations := []VMVariations{
		{name: "Go 1.23 VM", goTarget: testutil.Go1_23, features: mipsevm.FeatureToggles{}},
		{name: "Go 1.24 VM", goTarget: testutil.Go1_24, features: allFeaturesEnabled()},
	}

	for _, testCase := range testCases {
		for _, variation := range variations {
			testName := testNamer(variation.name, testCase)
			testCase := testCase
			variation := variation
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				vmTestCase(t, getVmFactory(variation.features), variation.goTarget, testCase)
			})
		}
	}
}

func getVmFactory(featureToggles mipsevm.FeatureToggles) VMFactory[*State] {
	return func(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, meta *program.Metadata) mipsevm.FPVM {
		return NewInstrumentedState(state, po, stdOut, stdErr, log, meta, featureToggles)
	}
}

func latestVm(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, meta *program.Metadata) mipsevm.FPVM {
	vmFactory := getVmFactory(allFeaturesEnabled())
	return vmFactory(state, po, stdOut, stdErr, log, meta)
}

// allFeaturesEnabled returns a FeatureToggles with all toggles enabled.
func allFeaturesEnabled() mipsevm.FeatureToggles {
	toggles := mipsevm.FeatureToggles{}
	tRef := reflect.ValueOf(&toggles).Elem() // Get a pointer and then dereference

	for i := 0; i < tRef.NumField(); i++ {
		field := tRef.Field(i)
		if field.Kind() == reflect.Bool && field.CanSet() {
			field.SetBool(true)
		}
	}

	return toggles
}

// Unit test splitmix64 based on Apache Commons RNG unit tests
// See: https://github.com/apache/commons-rng/blob/df772c2f5b0644a71398e925206039a2ae516ab2/commons-rng-core/src/test/java/org/apache/commons/rng/core/source64/SplitMix64Test.java
func TestSplitmix64(t *testing.T) {
	expectedSequence := []uint64{
		0x4141302768c9e9d0, 0x64df48c4eab51b1a, 0x4e723b53dbd901b3, 0xead8394409dd6454,
		0x3ef60e485b412a0a, 0xb2a23aee63aecf38, 0x6cc3b8933c4fa332, 0x9c9e75e031e6fccb,
		0x0fddffb161c9f30f, 0x2d1d75d4e75c12a3, 0xcdcf9d2dde66da2e, 0x278ba7d1d142cfec,
		0x4ca423e66072e606, 0x8f2c3c46ebc70bb7, 0xc9def3b1eeae3e21, 0x8e06670cd3e98bce,
		0x2326dee7dd34747f, 0x3c8fff64392bb3c1, 0xfc6aa1ebe7916578, 0x3191fb6113694e70,
		0x3453605f6544dac6, 0x86cf93e5cdf81801, 0x0d764d7e59f724df, 0xae1dfb943ebf8659,
		0x012de1babb3c4104, 0xa5a818b8fc5aa503, 0xb124ea2b701f4993, 0x18e0374933d8c782,
		0x2af8df668d68ad55, 0x76e56f59daa06243, 0xf58c016f0f01e30f, 0x8eeafa41683dbbf4,
		0x7bf121347c06677f, 0x4fd0c88d25db5ccb, 0x99af3be9ebe0a272, 0x94f2b33b74d0bdcb,
		0x24b5d9d7a00a3140, 0x79d983d781a34a3c, 0x582e4a84d595f5ec, 0x7316fe8b0f606d20,
	}

	var currentSeed uint64 = 0x1a2b3c4d5e6f7531
	for i := 0; i < len(expectedSequence); i++ {
		expectedOutput := expectedSequence[i]
		actual := splitmix64(currentSeed)
		require.Equal(t, expectedOutput, actual, fmt.Sprintf("Failed at index %d", i))
		currentSeed += 0x9e3779b97f4a7c15
	}
}

type VMFactory[T mipsevm.FPVMState] func(state T, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, meta *program.Metadata) mipsevm.FPVM
