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
			var oracle PreimageOracle
			if strings.HasPrefix(f.Name(), "oracle") {
				oracle = staticOracle(t, []byte("hello world"))
			}
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
			state := &State{PC: 0, NextPC: 4, Memory: NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)

			for i := 0; i < 1000; i++ {
				if us.state.PC == endAddr {
					break
				}
				if exitGroup && us.state.Exited {
					break
				}
				_, err := us.Step(false)
				require.NoError(t, err)
			}

			if exitGroup {
				require.NotEqual(t, uint32(endAddr), us.state.PC, "must not reach end")
				require.True(t, us.state.Exited, "must set exited state")
				require.Equal(t, uint8(1), us.state.ExitCode, "must exit with 1")
			} else {
				require.Equal(t, uint32(endAddr), us.state.PC, "must reach end")
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
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, PatchStack(state), "add initial stack")

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
	elfProgram, err := elf.Open("../example/bin/claim.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, PatchStack(state), "add initial stack")

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
