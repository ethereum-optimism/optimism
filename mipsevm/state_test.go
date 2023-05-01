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

	"github.com/ethereum-optimism/cannon/preimage"
)

// baseAddrStart - baseAddrEnd is used in tests to write the results to
const baseAddrEnd = 0xbf_ff_ff_f0
const baseAddrStart = 0xbf_c0_00_00

// endAddr is used as return-address for tests
const endAddr = 0xa7ef00d0

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			if f.Name() == "oracle.bin" {
				t.Skip("oracle test needs to be updated to use syscall pre-image oracle")
			}
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

			//err = state.SetMemoryRange(baseAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			//require.NoError(t, err, "must allocate page for the result data")
			//
			//err = state.SetMemoryRange(endAddr&^pageAddrMask, bytes.NewReader(make([]byte, pageSize)))
			//require.NoError(t, err, "must allocate page to return to")

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()

			require.NoError(t, mu.MemMap(baseAddrStart, ((baseAddrEnd-baseAddrStart)&^pageAddrMask)+pageSize))
			require.NoError(t, mu.MemMap(endAddr&^pageAddrMask, pageSize))

			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")

			us, err := NewUnicornState(mu, state, nil, os.Stdout, os.Stderr)
			require.NoError(t, err, "hook unicorn to state")

			for i := 0; i < 1000; i++ {
				if us.state.PC == endAddr {
					break
				}
				us.Step(false)
			}
			require.Equal(t, uint32(endAddr), us.state.PC, "must reach end")
			// inspect test result
			done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}

func TestHello(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = patchVM(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")

	mu, err := NewUnicorn()
	require.NoError(t, err, "load unicorn")
	defer mu.Close()
	err = LoadUnicorn(state, mu)
	require.NoError(t, err, "load state into unicorn")
	var stdOutBuf, stdErrBuf bytes.Buffer
	us, err := NewUnicornState(mu, state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))
	require.NoError(t, err, "hook unicorn to state")

	for i := 0; i < 400_000; i++ {
		if us.state.Exited {
			break
		}
		us.Step(false)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!", stdOutBuf.String(), "stdout says hello")
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

	err = patchVM(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")

	mu, err := NewUnicorn()
	require.NoError(t, err, "load unicorn")
	defer mu.Close()
	err = LoadUnicorn(state, mu)
	require.NoError(t, err, "load state into unicorn")

	oracle, expectedStdOut, expectedStdErr := claimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us, err := NewUnicornState(mu, state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))
	require.NoError(t, err, "hook unicorn to state")

	for i := 0; i < 2000_000; i++ {
		if us.state.Exited {
			break
		}
		us.Step(false)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}
