package forge

import (
	"bytes"
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type ioStruct struct {
	ID    uint8
	Data  []byte
	Slice []uint32
	Array [3]uint64
}

func TestMinimalSources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cl := NewClient(PathBinary())
	cl.Wd = projDir(t)

	// Build artifacts
	require.NoError(t, cl.Build(ctx))

	// Then copy them somewhere else
	tmpDir := t.TempDir()
	require.NoError(t, copyDir("testdata/testproject/out", path.Join(tmpDir, "out")))
	require.NoError(t, copyDir("testdata/testproject/cache", path.Join(tmpDir, "cache")))
	require.NoError(t, copyDir("testdata/testproject/script", path.Join(tmpDir, "script")))
	require.NoError(t, copyDir("testdata/testproject/foundry.toml", path.Join(tmpDir, "foundry.toml")))

	// The run must not recompile, since the cache is included.
	cl.Wd = tmpDir
	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)
	in := ioStruct{
		ID:    1,
		Data:  []byte{0x01, 0x02, 0x03, 0x04},
		Slice: []uint32{0x01, 0x02, 0x03, 0x04},
		Array: [3]uint64{0x01, 0x02, 0x03},
	}
	out, err := caller(ctx, in)
	require.NoError(t, err)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
	require.NoDirExists(t, path.Join(tmpDir, "broadcast"), "the run must leave nothing in the project")
}

// TestClient_Smoke smoke tests the Client by running the Version command on it.
func TestClient_Smoke(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)

	version, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.Regexp(t, regexp.MustCompile(`\d+\.\d+\.\d+`), version.Semver)
	require.Regexp(t, regexp.MustCompile(`^[a-f0-9]+$`), version.SHA)
}

func TestClient_OutputRedirection(t *testing.T) {
	bin := PathBinary()
	cl := NewClient(bin)
	cl.Stdout = new(bytes.Buffer)

	_, err := cl.Version(context.Background())
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(cl.Stdout.(*bytes.Buffer).String(), "forge Version"))
}

func TestScriptCaller(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bin := PathBinary()
	cl := NewClient(bin)
	cl.Wd = projDir(t)

	require.NoError(t, cl.Clean(ctx))
	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)

	in := ioStruct{
		ID:    1,
		Data:  []byte{0x01, 0x02},
		Slice: []uint32{0x01, 0x02, 0x03, 0x04},
		Array: [3]uint64{0x01, 0x02, 0x03},
	}
	_, err := caller(ctx, in)
	require.ErrorContains(t, err, "compile cache", "a cleaned project has no compile cache")

	require.NoError(t, cl.Build(ctx))
	out, err := caller(ctx, in)
	require.NoError(t, err)
	require.EqualValues(t, ioStruct{
		ID:    2,
		Data:  in.Data,
		Slice: in.Slice,
		Array: in.Array,
	}, out)
}

func TestScriptCaller_RejectsCompile(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// A compile cache that does not match the sources makes forge compile.
	tmpDir := t.TempDir()
	cl := NewClient(PathBinary())
	cl.Wd = projDir(t)
	require.NoError(t, cl.Build(ctx))
	require.NoError(t, copyDir("testdata/testproject/script", path.Join(tmpDir, "script")))
	require.NoError(t, copyDir("testdata/testproject/foundry.toml", path.Join(tmpDir, "foundry.toml")))
	require.NoError(t, copyDir("testdata/testproject/cache", path.Join(tmpDir, "cache")))
	cl.Wd = tmpDir

	caller := NewScriptCaller(
		cl,
		"script/Test.s.sol:TestScript",
		"runWithBytes(bytes)",
		&BytesScriptEncoder[ioStruct]{TypeName: "ioStruct"},
		&BytesScriptDecoder[ioStruct]{TypeName: "ioStruct"},
	)
	_, err := caller(ctx, ioStruct{ID: 1})
	require.ErrorContains(t, err, "forge compiled the project")
}

// stubProject returns a project with a compile cache, for runs with a stub forge.
func stubProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, compileCacheFile), []byte("{}"), 0o644))
	return dir
}

// stubBinary writes an executable shell script and returns it as a Binary.
func stubBinary(t *testing.T, script string) Binary {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stub-forge")
	require.NoError(t, os.WriteFile(p, []byte("#!/bin/sh\n"+script+"\n"), 0o755))
	return StaticBinary(p)
}

// TestClient_TimeoutUnblocksHungCommand reproduces the CI hang shape: a forge
// that never finishes (here: sleeps) while a child process keeps the output
// pipes open even after the parent is killed. Without Client.Timeout the call
// blocks until the process exits on its own; without cmd.WaitDelay it blocks
// on the pipe drain even after the kill.
func TestClient_TimeoutUnblocksHungCommand(t *testing.T) {
	oldDelay := waitDelay
	waitDelay = time.Second
	t.Cleanup(func() { waitDelay = oldDelay })

	cl := NewClient(stubBinary(t, "sleep 60 &\nexec sleep 60"))
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	cl.Timeout = 500 * time.Millisecond

	start := time.Now()
	err := cl.Build(context.Background())
	require.Error(t, err)
	require.Less(t, time.Since(start), 10*time.Second)
}

func TestClient_SlowBroadcast(t *testing.T) {
	tests := []struct {
		name     string
		opts     []string
		wantSlow bool
	}{
		{name: "broadcast", opts: []string{"--broadcast"}, wantSlow: true},
		{name: "read-only", opts: nil, wantSlow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewClient(stubBinary(t, `echo "$@"`))
			cl.Wd = stubProject(t)
			cl.Stdout = io.Discard

			out, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil, tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--slow"))
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--skip-simulation --gas-estimate-multiplier 100"),
				"a broadcast uses the node's gas estimate of each transaction, as txmgr does")
		})
	}
}

// TestClient_RunScriptStopsOnCompile checks that a run stops at the compile, before any broadcast.
func TestClient_RunScriptStopsOnCompile(t *testing.T) {
	// The stub's sleep can start before the kill and hold the output pipe; do not wait for it.
	oldDelay := waitDelay
	waitDelay = 100 * time.Millisecond
	t.Cleanup(func() { waitDelay = oldDelay })
	// A private slot pool, so that forge runs of other tests on the host do not count in the time.
	oldSlots := runSlots
	runSlots = &slotPool{dir: t.TempDir(), size: 1}
	t.Cleanup(func() { runSlots = oldSlots })
	broadcast := filepath.Join(t.TempDir(), "broadcast")
	cl := NewClient(stubBinary(t, "echo 'Compiling 1 files with Solc 0.8.15'\nsleep 5\ntouch "+broadcast))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	start := time.Now()
	_, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil, "--broadcast")
	require.ErrorContains(t, err, "forge compiled the project")
	require.Less(t, time.Since(start), 4*time.Second, "the run must stop when the compile starts")
	require.NoFileExists(t, broadcast)
}

// TestClient_LimitsConcurrentRuns checks that forge runs wait for a free slot.
func TestClient_LimitsConcurrentRuns(t *testing.T) {
	oldSlots := runSlots
	runSlots = &slotPool{dir: t.TempDir(), size: 1}
	t.Cleanup(func() { runSlots = oldSlots })

	cl := NewClient(stubBinary(t, "sleep 0.4"))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	start := time.Now()
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
			require.NoError(t, err)
		}()
	}
	wg.Wait()
	require.GreaterOrEqual(t, time.Since(start), 1200*time.Millisecond, "one slot runs the scripts one at a time")
}

func TestClient_RunScriptErrorHasReason(t *testing.T) {
	cl := NewClient(stubBinary(t, `echo "Error: script failed: boom" >&2; exit 1`))
	cl.Wd = stubProject(t)
	cl.Stderr = io.Discard

	_, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.ErrorContains(t, err, "Error: script failed: boom")
}

// sandboxScript calls the cheatcodes that the sandbox must refuse.
const sandboxScript = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface Vm {
    function ffi(string[] calldata commandInput) external returns (bytes memory result);
    function readFile(string calldata path) external view returns (string memory data);
}

contract Sandbox {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    function runFfi(bytes memory) public returns (bytes memory) {
        string[] memory cmd = new string[](2);
        cmd[0] = "echo";
        cmd[1] = "0x1234";
        return vm.ffi(cmd);
    }

    function readHosts(bytes memory) public view returns (string memory) {
        return vm.readFile("/etc/hosts");
    }
}
`

// TestClient_SandboxesScripts checks that a script can neither run commands nor read files outside
// FsPermissions, whatever the project config allows.
func TestClient_SandboxesScripts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script", "Sandbox.s.sol"), []byte(sandboxScript), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte(`[profile.default]
src = "script"
libs = []
ffi = true
fs_permissions = [{ access = "read", path = "/" }]
`), 0o644))
	cl := NewClient(PathBinary())
	cl.Wd = dir
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	require.NoError(t, cl.Build(ctx))
	entries := func() []string {
		des, err := os.ReadDir(dir)
		require.NoError(t, err)
		var names []string
		for _, de := range des {
			names = append(names, de.Name())
		}
		return names
	}
	before := entries()

	_, err := cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "runFfi(bytes)", nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "forge compiled", "the sandbox config must keep the compile cache valid")
	require.Contains(t, strings.ToLower(err.Error()), "ffi")

	cl.FsPermissions = []FsPermission{}
	_, err = cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "readHosts(bytes)", nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "forge compiled")
	require.Contains(t, err.Error(), "not allowed")

	require.Equal(t, before, entries(), "the runs must leave nothing in the project")
}

// TestClient_ForgeEnvironment checks that script runs inherit only the variables they need: a script
// can read the environment, and FOUNDRY_*, DAPP_* and ETH_* variables configure forge.
func TestClient_ForgeEnvironment(t *testing.T) {
	t.Setenv("FOUNDRY_OPTIMIZER_RUNS", "1")
	t.Setenv("DAPP_LIBRARIES", "x")
	t.Setenv("ETH_RPC_URL", "http://127.0.0.1:1")
	t.Setenv("DEPLOYER_PRIVATE_KEY", "secret")
	t.Setenv("HTTPS_PROXY", "http://proxy")
	// The stub prints only the variables under test, so that a failure does not log the environment.
	cl := NewClient(stubBinary(t, `echo "runs=$FOUNDRY_OPTIMIZER_RUNS libs=$DAPP_LIBRARIES rpc=$ETH_RPC_URL key=$DEPLOYER_PRIVATE_KEY proxy=$HTTPS_PROXY home=${HOME:+set} broadcast=${FOUNDRY_BROADCAST:+set}"`))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	out, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "runs= libs= rpc= key= proxy=http://proxy home=set broadcast=set\n", out, "the client's own variables reach forge")
}

// TestClient_BuildEnvironment checks that a build gets inheritedEnv too: a FOUNDRY_* setting would
// make the build differ from the one that a script run checks against.
func TestClient_BuildEnvironment(t *testing.T) {
	t.Setenv("FOUNDRY_EVM_VERSION", "paris")
	out := new(bytes.Buffer)
	cl := NewClient(stubBinary(t, `echo "evm=$FOUNDRY_EVM_VERSION"`))
	cl.Wd = stubProject(t)
	cl.Stdout = out

	require.NoError(t, cl.Build(context.Background()))
	require.Equal(t, "evm=\n", out.String())
}

// TestClient_VerifyContractEnvironment checks that verify-contract gets the full environment: forge
// reads its API key from ETHERSCAN_API_KEY.
func TestClient_VerifyContractEnvironment(t *testing.T) {
	t.Setenv("ETHERSCAN_API_KEY", "secret")
	cl := NewClient(stubBinary(t, `echo "key=${ETHERSCAN_API_KEY:+set}"`))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	out, err := cl.VerifyContract(context.Background())
	require.NoError(t, err)
	require.Equal(t, "key=set\n", out)
}

func TestClient_WithEnv(t *testing.T) {
	base := NewClient(stubBinary(t, `echo "A=$OP_DEPLOYER_TEST_A B=$OP_DEPLOYER_TEST_B"`))
	base.Wd = stubProject(t)
	base.Stdout = io.Discard
	base = base.WithEnv("OP_DEPLOYER_TEST_A=1")
	withB := base.WithEnv("OP_DEPLOYER_TEST_B=2")

	out, err := withB.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=2\n", out)

	out, err = base.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=\n", out, "WithEnv must not change the client it copies")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		return copyFile(path, targetPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func projDir(t *testing.T) string {
	_, testFilename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Join(filepath.Dir(testFilename), "testdata", "testproject")
	absProjDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	return absProjDir
}
