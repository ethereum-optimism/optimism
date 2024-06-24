package mipsevm

import (
	"bytes"
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

// 0xbf_c0_00_00 ... baseAddrEnd is used in tests to write the results to
const baseAddrEnd = 0xbf_ff_ff_f0

// endAddr is used as return-address for tests
const endAddr = 0xa7ef00d0

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			oracle := selectOracleFixture(t, f.Name())
			// Short-circuit early for exit_group.bin
			exitGroup := f.Name() == "exit_group.bin"

			// TODO: currently tests are compiled as flat binary objects
			// We can use more standard tooling to compile them to ELF files and get remove maketests.py
			fn := path.Join("open_mips_tests/test/bin", f.Name())
			//elfProgram, err := elf.Open()
			//require.NoError(t, err, "must load test ELF binary")
			//state, err := LoadELF(elfProgram)
			//require.NoError(t, err, "must load ELF into state")
			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &State{Cpu: CpuScalars{PC: 0, NextPC: 4}, Memory: NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)

			for i := 0; i < 1000; i++ {
				if us.state.Cpu.PC == endAddr {
					break
				}
				if exitGroup && us.state.Exited {
					break
				}
				_, err := us.Step(false)
				require.NoError(t, err)
			}

			if exitGroup {
				require.NotEqual(t, uint32(endAddr), us.state.Cpu.PC, "must not reach end")
				require.True(t, us.state.Exited, "must set exited state")
				require.Equal(t, uint8(1), us.state.ExitCode, "must exit with 1")
			} else {
				require.Equal(t, uint32(endAddr), us.state.Cpu.PC, "must reach end")
				done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
				// inspect test result
				require.Equal(t, done, uint32(1), "must be done")
				require.Equal(t, result, uint32(1), "must have success result")
			}
		})
	}
}

// Run through all permutations of `exited` / `exitCode` and ensure that the
// correct witness, state hash, and VM Status is produced.
func TestStateHash(t *testing.T) {
	cases := []struct {
		exited   bool
		exitCode uint8
	}{
		{exited: false, exitCode: 0},
		{exited: false, exitCode: 1},
		{exited: false, exitCode: 2},
		{exited: false, exitCode: 3},
		{exited: true, exitCode: 0},
		{exited: true, exitCode: 1},
		{exited: true, exitCode: 2},
		{exited: true, exitCode: 3},
	}

	exitedOffset := 32*2 + 4*6
	for _, c := range cases {
		state := &State{
			Memory:   NewMemory(),
			Exited:   c.exited,
			ExitCode: c.exitCode,
		}

		actualWitness := state.EncodeWitness()
		actualStateHash, err := StateWitness(actualWitness).StateHash()
		require.NoError(t, err, "Error hashing witness")
		require.Equal(t, len(actualWitness), StateWitnessSize, "Incorrect witness size")

		expectedWitness := make(StateWitness, 226)
		memRoot := state.Memory.MerkleRoot()
		copy(expectedWitness[:32], memRoot[:])
		expectedWitness[exitedOffset] = c.exitCode
		var exited uint8
		if c.exited {
			exited = 1
		}
		expectedWitness[exitedOffset+1] = uint8(exited)
		require.Equal(t, expectedWitness[:], actualWitness[:], "Incorrect witness")

		expectedStateHash := crypto.Keccak256Hash(actualWitness)
		expectedStateHash[0] = vmStatus(c.exited, c.exitCode)
		require.Equal(t, expectedStateHash, actualStateHash, "Incorrect state hash")
	}
}

func TestHello(t *testing.T) {
	state := loadELFProgram(t, "../example/bin/hello.elf")

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := NewInstrumentedState(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	for i := 0; i < 400_000; i++ {
		if us.state.Exited {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}

type testOracle struct {
	hint        func(v []byte)
	getPreimage func(k [32]byte) []byte
}

func (t *testOracle) Hint(v []byte) {
	t.hint(v)
}

func (t *testOracle) GetPreimage(k [32]byte) []byte {
	return t.getPreimage(k)
}

var _ PreimageOracle = (*testOracle)(nil)

func claimTestOracle(t *testing.T) (po PreimageOracle, stdOut string, stdErr string) {
	s := uint64(1000)
	a := uint64(3)
	b := uint64(4)

	encodeU64 := func(x uint64) []byte {
		return binary.BigEndian.AppendUint64(nil, x)
	}

	var diff []byte
	diff = append(diff, crypto.Keccak256(encodeU64(a))...)
	diff = append(diff, crypto.Keccak256(encodeU64(b))...)

	preHash := crypto.Keccak256Hash(encodeU64(s))
	diffHash := crypto.Keccak256Hash(diff)

	images := make(map[[32]byte][]byte)
	images[preimage.LocalIndexKey(0).PreimageKey()] = preHash[:]
	images[preimage.LocalIndexKey(1).PreimageKey()] = diffHash[:]
	images[preimage.LocalIndexKey(2).PreimageKey()] = encodeU64(s*a + b)

	oracle := &testOracle{
		hint: func(v []byte) {
			parts := strings.Split(string(v), " ")
			require.Len(t, parts, 2)
			p, err := hex.DecodeString(parts[1])
			require.NoError(t, err)
			require.Len(t, p, 32)
			h := common.Hash(*(*[32]byte)(p))
			switch parts[0] {
			case "fetch-state":
				require.Equal(t, h, preHash, "expecting request for pre-state pre-image")
				images[preimage.Keccak256Key(preHash).PreimageKey()] = encodeU64(s)
			case "fetch-diff":
				require.Equal(t, h, diffHash, "expecting request for diff pre-images")
				images[preimage.Keccak256Key(diffHash).PreimageKey()] = diff
				images[preimage.Keccak256Key(crypto.Keccak256Hash(encodeU64(a))).PreimageKey()] = encodeU64(a)
				images[preimage.Keccak256Key(crypto.Keccak256Hash(encodeU64(b))).PreimageKey()] = encodeU64(b)
			default:
				t.Fatalf("unexpected hint: %q", parts[0])
			}
		},
		getPreimage: func(k [32]byte) []byte {
			p, ok := images[k]
			if !ok {
				t.Fatalf("missing pre-image %s", k)
			}
			return p
		},
	}

	return oracle, fmt.Sprintf("computing %d * %d + %d\nclaim %d is good!\n", s, a, b, s*a+b), "started!"
}

func TestClaim(t *testing.T) {
	state := loadELFProgram(t, "../example/bin/claim.elf")

	oracle, expectedStdOut, expectedStdErr := claimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := NewInstrumentedState(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	for i := 0; i < 2000_000; i++ {
		if us.state.Exited {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}

func TestAlloc(t *testing.T) {
	t.Skip("TODO(client-pod#906): Currently fails on Single threaded Cannon. Re-enable for the MT FPVM")

	state := loadELFProgram(t, "../example/bin/alloc.elf")
	const numAllocs = 100 // where each alloc is a 32 MiB chunk
	oracle := allocOracle(t, numAllocs)

	// completes in ~870 M steps
	us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
	for i := 0; i < 20_000_000_000; i++ {
		if us.state.Exited {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
		if state.Step%10_000_000 == 0 {
			t.Logf("Completed %d steps", state.Step)
		}
	}
	t.Logf("Completed in %d steps", state.Step)
	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
	require.Less(t, state.Memory.PageCount()*PageSize, 1*1024*1024*1024, "must not allocate more than 1 GiB")
}

func loadELFProgram(t *testing.T, name string) *State {
	elfProgram, err := elf.Open(name)
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, PatchStack(state), "add initial stack")
	return state
}

func staticOracle(t *testing.T, preimageData []byte) *testOracle {
	return &testOracle{
		hint: func(v []byte) {},
		getPreimage: func(k [32]byte) []byte {
			if k != preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey() {
				t.Fatalf("invalid preimage request for %x", k)
			}
			return preimageData
		},
	}
}

func staticPrecompileOracle(t *testing.T, precompile common.Address, input []byte, result []byte) *testOracle {
	return &testOracle{
		hint: func(v []byte) {},
		getPreimage: func(k [32]byte) []byte {
			keyData := append(precompile.Bytes(), input...)
			switch k[0] {
			case byte(preimage.Keccak256KeyType):
				if k != preimage.Keccak256Key(crypto.Keccak256Hash(keyData)).PreimageKey() {
					t.Fatalf("invalid preimage request for %x", k)
				}
				return keyData
			case byte(preimage.PrecompileKeyType):
				if k != preimage.PrecompileKey(crypto.Keccak256Hash(keyData)).PreimageKey() {
					t.Fatalf("invalid preimage request for %x", k)
				}
				return result
			}
			panic("unreachable")
		},
	}
}

func allocOracle(t *testing.T, numAllocs int) *testOracle {
	return &testOracle{
		hint: func(v []byte) {},
		getPreimage: func(k [32]byte) []byte {
			if k != preimage.LocalIndexKey(0).PreimageKey() {
				t.Fatalf("invalid preimage request for %x", k)
			}
			return binary.LittleEndian.AppendUint64(nil, uint64(numAllocs))
		},
	}
}

func selectOracleFixture(t *testing.T, programName string) PreimageOracle {
	if strings.HasPrefix(programName, "oracle_kzg") {
		precompile := common.BytesToAddress([]byte{0xa})
		input := common.FromHex("01e798154708fe7789429634053cbf9f99b619f9f084048927333fce637f549b564c0a11a0f704f4fc3e8acfe0f8245f0ad1347b378fbf96e206da11a5d3630624d25032e67a7e6a4910df5834b8fe70e6bcfeeac0352434196bdf4b2485d5a18f59a8d2a1a625a17f3fea0fe5eb8c896db3764f3185481bc22f91b4aaffcca25f26936857bc3a7c2539ea8ec3a952b7873033e038326e87ed3e1276fd140253fa08e9fc25fb2d9a98527fc22a2c9612fbeafdad446cbc7bcdbdcd780af2c16a")
		blobPrecompileReturnValue := common.FromHex("000000000000000000000000000000000000000000000000000000000000100073eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001")
		return staticPrecompileOracle(t, precompile, input, append([]byte{0x1}, blobPrecompileReturnValue...))
	} else if strings.HasPrefix(programName, "oracle") {
		return staticOracle(t, []byte("hello world"))
	} else {
		return nil
	}
}

func TestStateJSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	stateJSON, err := state.MarshalJSON()
	require.NoError(t, err)

	newState := new(State)
	require.NoError(t, newState.UnmarshalJSON(stateJSON))

	require.Equal(t, state.PreimageKey, newState.PreimageKey)
	require.Equal(t, state.PreimageOffset, newState.PreimageOffset)
	require.Equal(t, state.Cpu, newState.Cpu)
	require.Equal(t, state.Heap, newState.Heap)
	require.Equal(t, state.ExitCode, newState.ExitCode)
	require.Equal(t, state.Exited, newState.Exited)
	require.Equal(t, state.Memory.MerkleRoot(), newState.Memory.MerkleRoot())
	require.Equal(t, state.Registers, newState.Registers)
	require.Equal(t, state.Step, newState.Step)
}
