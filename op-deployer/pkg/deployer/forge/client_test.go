package forge

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
	out, transactions, err := caller(ctx, in)
	require.NoError(t, err)
	require.Empty(t, transactions)
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
	_, _, err := caller(ctx, in)
	require.ErrorContains(t, err, "compile cache", "a cleaned project has no compile cache")

	require.NoError(t, cl.Build(ctx))
	out, _, err := caller(ctx, in)
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
	_, _, err := caller(ctx, ioStruct{ID: 1})
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

// scriptRan is what forge prints after a script that ran.
const scriptRan = "Script ran successfully.\n"

// stubScript returns a stub forge that runs script, then prints what forge prints after a script
// run that sends nothing.
func stubScript(t *testing.T, script string) Binary {
	t.Helper()
	return stubBinary(t, script+"\nprintf '"+scriptRan+"'")
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

// TestClient_SlowBroadcast checks the flags of a forge broadcast, and that it returns no transactions
// to send: forge sent them.
func TestClient_SlowBroadcast(t *testing.T) {
	tests := []struct {
		name     string
		opts     []string
		wantSlow bool
	}{
		{name: "broadcast", opts: []string{"--broadcast"}, wantSlow: true},
		{name: "dry run", opts: nil, wantSlow: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cl := NewClient(stubScript(t, `echo "$@"`))
			cl.Wd = stubProject(t)
			cl.Stdout = io.Discard

			out, transactions, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil, tt.opts...)
			require.NoError(t, err)
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--slow"))
			require.Equal(t, tt.wantSlow, strings.Contains(out, "--skip-simulation --gas-estimate-multiplier 100"),
				"a broadcast uses the node's gas estimate of each transaction, as txmgr does")
			if tt.wantSlow {
				require.Nil(t, transactions)
			} else {
				require.Empty(t, transactions)
				require.NotNil(t, transactions)
			}
		})
	}
}

// TestClient_RunScriptStopsOnCompile checks that a run stops at the compile, before the script runs.
func TestClient_RunScriptStopsOnCompile(t *testing.T) {
	// The stub's sleep can start before the kill and hold the output pipe; do not wait for it.
	oldDelay := waitDelay
	waitDelay = 100 * time.Millisecond
	t.Cleanup(func() { waitDelay = oldDelay })
	// A private slot pool, so that forge runs of other tests on the host do not count in the time.
	oldSlots := runSlots
	runSlots = &slotPool{dir: filepath.Join(t.TempDir(), "slots"), size: 1}
	t.Cleanup(func() { runSlots = oldSlots })
	ran := filepath.Join(t.TempDir(), "ran")
	cl := NewClient(stubBinary(t, "echo 'Compiling 1 files with Solc 0.8.15'\nsleep 5\ntouch "+ran))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	start := time.Now()
	_, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.ErrorIs(t, err, ErrCompiled)
	require.Less(t, time.Since(start), 4*time.Second, "the run must stop when the compile starts")
	require.NoFileExists(t, ran)
}

// TestClient_LimitsConcurrentRuns checks that forge runs wait for a free slot.
func TestClient_LimitsConcurrentRuns(t *testing.T) {
	oldSlots := runSlots
	runSlots = &slotPool{dir: filepath.Join(t.TempDir(), "slots"), size: 1}
	t.Cleanup(func() { runSlots = oldSlots })

	cl := NewClient(stubScript(t, "sleep 0.4"))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	start := time.Now()
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	require.GreaterOrEqual(t, time.Since(start), 1200*time.Millisecond, "one slot runs the scripts one at a time")
}

func TestClient_RunScriptErrorHasReason(t *testing.T) {
	cl := NewClient(stubBinary(t, `echo "Error: script failed: boom" >&2; exit 1`))
	cl.Wd = stubProject(t)
	cl.Stderr = io.Discard

	_, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.ErrorContains(t, err, "Error: script failed: boom")
	require.ErrorIs(t, err, ErrScriptReverted)
}

// TestClient_RunScriptRPCErrorIsNotRevert checks that an RPC error is not reported as a revert: the
// messages are those of forge 1.8.3 for a fork block that the RPC does not serve yet, and for a
// backend that has not seen the fork block: before the run, in an opcode, and in a cheatcode, which
// forge reports as a failed script.
func TestClient_RunScriptRPCErrorIsNotRevert(t *testing.T) {
	for _, msg := range []string{
		"Error: failed to get block number: 500; latest block number: 5",
		"Error: failed to get account for 0x01: server returned an error response: error code -32000: block not found: hash 0x02",
		"Error: EVM error\n\nContext:\n- database error: failed to get account for 0x01: server returned an error response: error code -32000: block not found: hash 0x02",
		"Error: script failed: vm.load: failed to get account for 0x01: server returned an error response: error code -32000: block not found: hash 0x02",
	} {
		cl := NewClient(stubBinary(t, "printf '%s\\n' '"+msg+"' >&2; exit 1"))
		cl.Wd = stubProject(t)
		cl.Stderr = io.Discard

		_, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
		require.ErrorContains(t, err, strings.SplitN(msg, "\n", 2)[0])
		require.NotErrorIs(t, err, ErrScriptReverted)
	}
}

// failureScript reverts in each way that solidity can, and forks from a URL with a cheatcode, which
// fails when the RPC request does.
const failureScript = `// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

interface Vm {
    function createSelectFork(string calldata urlOrAlias) external returns (uint256 forkId);
}

contract Failures {
    Vm internal constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    error Boom(uint256 x);

    function revertReason(bytes memory) public pure returns (bytes memory) {
        require(false, "boom");
        return "";
    }

    function revertCustom(bytes memory) public pure returns (bytes memory) {
        revert Boom(7);
    }

    function revertPanic(bytes memory) public pure returns (bytes memory) {
        assert(false);
        return "";
    }

    function revertEmpty(bytes memory) public pure returns (bytes memory) {
        revert();
    }

    function fork(bytes memory url) public returns (bytes memory) {
        vm.createSelectFork(string(url));
        return "";
    }
}
`

// TestClient_RunScriptReportsOnlyScriptRevertsAsReverts checks with the pinned forge that RunScript
// reports a revert of the script as ErrScriptReverted, and an RPC failure in a cheatcode, which
// forge reports as a failed script too, as an error that the caller may retry. The forge messages
// it relies on are pinned here, so a forge bump that changes them fails this test.
func TestClient_RunScriptReportsOnlyScriptRevertsAsReverts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cl := NewClient(PathBinary())
	cl.Wd = scriptProject(t, ctx, "Failures.s.sol", failureScript, "")
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	cl.FsPermissions = []FsPermission{}

	// The listener stays open and closes each connection, so that no other process can take its port.
	dropping, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = dropping.Close() })
	go func() {
		for {
			conn, err := dropping.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	badGateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	t.Cleanup(badGateway.Close)
	rpcError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"jsonrpc":"2.0","id":0,"error":{"code":-32601,"message":"method not allowed"}}`)
	}))
	t.Cleanup(rpcError.Close)

	for _, tc := range []struct {
		name    string
		sig     string
		args    string
		want    string
		reverts bool
	}{
		{name: "reason", sig: "revertReason(bytes)", want: "boom", reverts: true},
		{name: "custom error", sig: "revertCustom(bytes)", want: "Boom(7)", reverts: true},
		{name: "panic", sig: "revertPanic(bytes)", want: "panic: assertion failed", reverts: true},
		{name: "no data", sig: "revertEmpty(bytes)", want: "EvmError: Revert", reverts: true},
		{name: "connection dropped", sig: "fork(bytes)", args: "http://" + dropping.Addr().String(), want: "vm.createSelectFork"},
		{name: "bad gateway", sig: "fork(bytes)", args: badGateway.URL, want: "vm.createSelectFork"},
		{name: "rpc error", sig: "fork(bytes)", args: rpcError.URL, want: "vm.createSelectFork"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := cl.RunScript(ctx, "script/Failures.s.sol:Failures", tc.sig, []byte(tc.args))
			require.ErrorContains(t, err, tc.want)
			require.NotErrorIs(t, err, ErrCompiled)
			if tc.reverts {
				require.ErrorIs(t, err, ErrScriptReverted)
			} else {
				require.NotErrorIs(t, err, ErrScriptReverted)
			}
		})
	}
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

    function readArtifact(bytes memory) public view returns (string memory) {
        return vm.readFile("forge-artifacts/secret");
    }

    function noop(bytes memory) public pure returns (bytes memory) {
        return "";
    }
}
`

// sandboxProject builds the Sandbox script in a new project, with settings added to its default
// profile, and returns its directory.
func sandboxProject(t *testing.T, ctx context.Context, settings string) string {
	t.Helper()
	return scriptProject(t, ctx, "Sandbox.s.sol", sandboxScript, settings)
}

// scriptProject builds a script with the given file name and source in a new project, with settings
// added to its default profile, and returns its directory.
func scriptProject(t *testing.T, ctx context.Context, file string, source string, settings string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script", file), []byte(source), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte("[profile.default]\nsrc = \"script\"\nlibs = []\n"+settings), 0o644))
	cl := NewClient(PathBinary())
	cl.Wd = dir
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	require.NoError(t, cl.Build(ctx))
	return dir
}

// TestClient_SandboxIgnoresProjectConfig checks that a project cannot turn the sandbox off through
// the parts of its config that forge reads besides the default profile.
func TestClient_SandboxIgnoresProjectConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	appendFile := func(t *testing.T, path string, data string) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		require.NoError(t, err)
		_, err = f.WriteString(data)
		require.NoError(t, err)
		require.NoError(t, f.Close())
	}
	run := func(t *testing.T, dir string, sig string) error {
		cl := NewClient(PathBinary())
		cl.Wd = dir
		cl.Stdout = io.Discard
		cl.Stderr = io.Discard
		cl.FsPermissions = []FsPermission{}
		_, _, err := cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", sig, nil)
		if err != nil {
			require.NotContains(t, err.Error(), "forge compiled", "the sandbox config must keep the compile cache valid")
		}
		return err
	}

	t.Run("profile from .env", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		appendFile(t, filepath.Join(dir, "foundry.toml"), "[profile.alt]\nffi = true\n")
		appendFile(t, filepath.Join(dir, ".env"), "FOUNDRY_PROFILE=alt\n")
		err := run(t, dir, "runFfi(bytes)")
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "ffi")
	})
	t.Run("setting from .env", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		appendFile(t, filepath.Join(dir, ".env"), "FOUNDRY_FFI=true\n")
		err := run(t, dir, "runFfi(bytes)")
		require.Error(t, err)
		require.Contains(t, strings.ToLower(err.Error()), "ffi")
	})
	t.Run("extends", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		base := filepath.Join(dir, "base.toml")
		require.NoError(t, os.WriteFile(base, []byte("[profile.default]\nfs_permissions = [{ access = \"read-write\", path = \"/\" }]\n"), 0o644))
		appendFile(t, filepath.Join(dir, "foundry.toml"), "extends = { path = \""+base+"\", strategy = \"extend-arrays\" }\n")
		err := run(t, dir, "readHosts(bytes)")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not allowed")
	})
	t.Run("solc binary", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		marker := filepath.Join(t.TempDir(), "solc-ran")
		solc := filepath.Join(dir, "solc.sh")
		require.NoError(t, os.WriteFile(solc, []byte("#!/bin/sh\ntouch "+marker+"\nexit 1\n"), 0o755))
		appendFile(t, filepath.Join(dir, "foundry.toml"), "solc = \""+solc+"\"\n")
		require.ErrorContains(t, run(t, dir, "noop(bytes)"), "only a compiler version")
		require.NoFileExists(t, marker, "forge must not run a compiler that the project names")
	})
	t.Run("solc version", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "solc_version = \"0.8.15\"\n")
		require.NoError(t, run(t, dir, "noop(bytes)"), "a pinned compiler version is kept")
	})
	t.Run("deny warnings", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		appendFile(t, filepath.Join(dir, "foundry.toml"), "deny = \"warnings\"\n")
		require.NoError(t, run(t, dir, "noop(bytes)"))
	})
}

// TestClient_RunScriptKeepsLibRemappings checks that a script run keeps the remappings that forge
// detects in the project's libraries.
func TestClient_RunScriptKeepsLibRemappings(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// The project is reached through a link, as on macOS, whose temporary directories are under a
	// link; forge resolves the link.
	dir := filepath.Join(t.TempDir(), "project")
	require.NoError(t, os.Symlink(t.TempDir(), dir))
	dep := filepath.Join(dir, "lib", "dep", "src")
	require.NoError(t, os.MkdirAll(dep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dep, "Dep.sol"), []byte(`// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

library Dep {
    function one() internal pure returns (uint256) {
        return 1;
    }
}
`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "script"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "script", "Dep.s.sol"), []byte(`// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import {Dep} from "dep/Dep.sol";

contract DepScript {
    function run(bytes memory) public pure returns (uint256) {
        return Dep.one();
    }
}
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte("[profile.default]\nsrc = \"script\"\n"), 0o644))
	cl := NewClient(PathBinary())
	cl.Wd = dir
	cl.Stdout = io.Discard
	cl.Stderr = io.Discard
	cl.FsPermissions = []FsPermission{}
	require.NoError(t, cl.Build(ctx))

	_, _, err := cl.RunScript(ctx, "script/Dep.s.sol:DepScript", "run(bytes)", nil)
	require.NoError(t, err)
}

// TestClient_RunScriptRefusesEqualsInPath checks that a script run refuses a project whose path
// holds a '=', in which forge would compile.
func TestClient_RunScriptRefusesEqualsInPath(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "case=1")
	require.NoError(t, os.Mkdir(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foundry.toml"), []byte("[profile.default]\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, compileCacheFile), []byte("{}"), 0o644))
	cl := NewClient(PathBinary())
	cl.Wd = dir
	cl.FsPermissions = []FsPermission{}

	_, _, err := cl.RunScript(context.Background(), "script/A.s.sol:A", "run()", nil)
	require.ErrorContains(t, err, "holds a '='")
}

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

	_, _, err := cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "runFfi(bytes)", nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "forge compiled", "the sandbox config must keep the compile cache valid")
	require.Contains(t, strings.ToLower(err.Error()), "ffi")

	cl.FsPermissions = []FsPermission{}
	_, _, err = cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "readHosts(bytes)", nil)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "forge compiled")
	require.Contains(t, err.Error(), "not allowed")

	require.Equal(t, before, entries(), "the runs must leave nothing in the project")
}

// TestClient_SandboxRefusesLinkedArtifacts checks that a project cannot widen the read access to its
// artifacts by making forge-artifacts a link: forge grants access to the target of the link.
func TestClient_SandboxRefusesLinkedArtifacts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	writeSecret := func(t *testing.T, dir string) {
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "secret"), []byte("s3cret"), 0o644))
	}
	run := func(t *testing.T, dir string) (string, error) {
		cl := NewClient(PathBinary())
		cl.Wd = dir
		cl.Stdout = io.Discard
		cl.Stderr = io.Discard
		cl.FsPermissions = []FsPermission{ArtifactsReadPermission}
		out, _, err := cl.RunScript(ctx, "script/Sandbox.s.sol:Sandbox", "readArtifact(bytes)", nil)
		return out, err
	}

	t.Run("directory", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		writeSecret(t, filepath.Join(dir, "forge-artifacts"))
		out, err := run(t, dir)
		require.NoError(t, err)
		require.Contains(t, out, "s3cret")
	})
	t.Run("link", func(t *testing.T) {
		dir := sandboxProject(t, ctx, "")
		outside := t.TempDir()
		writeSecret(t, outside)
		require.NoError(t, os.Symlink(outside, filepath.Join(dir, "forge-artifacts")))
		out, err := run(t, dir)
		require.ErrorContains(t, err, "is a link")
		require.NotContains(t, out, "s3cret")
	})
}

// TestClient_SandboxRefusesLinksInArtifacts checks that a link inside forge-artifacts is refused
// before forge runs: forge refuses a short one itself, but not a chain whose resolved path is longer
// than PATH_MAX.
func TestClient_SandboxRefusesLinksInArtifacts(t *testing.T) {
	project := t.TempDir()
	artifacts := filepath.Join(project, "forge-artifacts", "Foo.sol")
	require.NoError(t, os.MkdirAll(artifacts, 0o755))
	require.NoError(t, os.Symlink(t.TempDir(), filepath.Join(artifacts, "x")))
	cl := NewClient(nil)
	cl.FsPermissions = []FsPermission{ArtifactsReadPermission}
	_, _, err := cl.sandboxConfig(project, t.TempDir())
	require.ErrorContains(t, err, "is a link")

	cl.FsPermissions = []FsPermission{}
	_, _, err = cl.sandboxConfig(project, t.TempDir())
	require.NoError(t, err, "a link outside the granted paths is not read")
}

// TestClient_ForgeEnvironment checks that script runs inherit only inheritedEnvNames and the
// client's own variables.
func TestClient_ForgeEnvironment(t *testing.T) {
	t.Setenv("FOUNDRY_OPTIMIZER_RUNS", "1")
	t.Setenv("DAPP_LIBRARIES", "x")
	t.Setenv("ETH_RPC_URL", "http://127.0.0.1:1")
	t.Setenv("DEPLOYER_PRIVATE_KEY", "secret")
	t.Setenv("HTTPS_PROXY", "http://proxy")
	// The stub prints only the variables under test, so that a failure does not log the environment.
	cl := NewClient(stubScript(t, `echo "runs=$FOUNDRY_OPTIMIZER_RUNS libs=$DAPP_LIBRARIES rpc=$ETH_RPC_URL key=$DEPLOYER_PRIVATE_KEY proxy=$HTTPS_PROXY home=${HOME:+set} broadcast=${FOUNDRY_BROADCAST:+set}"`))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	out, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "runs= libs= rpc= key= proxy=http://proxy home=set broadcast=set\n"+scriptRan, out, "the client's own variables reach forge")
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

// TestClient_CertDirs checks that forge trusts the certificates that Go trusts.
func TestClient_CertDirs(t *testing.T) {
	want := "dirs=\n"
	if runtime.GOOS == "linux" {
		want = "dirs=/etc/ssl/certs:/etc/pki/tls/certs\n"
	}
	t.Setenv("SSL_CERT_DIR", "")
	require.NoError(t, os.Unsetenv("SSL_CERT_DIR"))
	cl := NewClient(stubScript(t, `echo "dirs=$SSL_CERT_DIR"`))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard

	out, _, err := cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, want+scriptRan, out, "script run")
	out, err = cl.VerifyContract(context.Background())
	require.NoError(t, err)
	require.Equal(t, want+scriptRan, out, "verify-contract")

	t.Setenv("SSL_CERT_DIR", "")
	out, _, err = cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, want+scriptRan, out, "an empty SSL_CERT_DIR counts as unset, as in Go")

	t.Setenv("SSL_CERT_DIR", "/custom")
	out, _, err = cl.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "dirs=/custom\n"+scriptRan, out, "the user's SSL_CERT_DIR wins")
}

// TestClient_ScriptTarget checks that a script path is made absolute in the project, since forge runs
// in a scratch directory, and that a bare contract name is left for forge to look up.
func TestClient_ScriptTarget(t *testing.T) {
	// The target is the argument before the script's input.
	cl := NewClient(stubScript(t, `prev=""; for arg in "$@"; do target="$prev"; prev="$arg"; done; echo "target=$target"`))
	cl.Wd = stubProject(t)
	cl.Stdout = io.Discard
	root, err := filepath.Abs(cl.Wd)
	require.NoError(t, err)

	for target, want := range map[string]string{
		"script/Test.s.sol:TestScript": filepath.Join(root, "script/Test.s.sol") + ":TestScript",
		"script/Test.s.sol":            filepath.Join(root, "script/Test.s.sol"),
		"TestScript":                   "TestScript",
	} {
		out, _, err := cl.RunScript(context.Background(), target, "run()", nil)
		require.NoError(t, err)
		require.Equal(t, "target="+want+"\n"+scriptRan, out, target)
	}
}

// TestClient_SandboxConfigKebabKeys checks that a kebab-case key of the project config is treated as
// forge treats it: as its snake-case name.
func TestClient_SandboxConfigKebabKeys(t *testing.T) {
	project := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(project, "foundry.toml"), []byte(`[profile.default]
optimizer-runs = 5
eth-rpc-url = "http://127.0.0.1:1"
`), 0o644))
	path, _, err := NewClient(nil).sandboxConfig(project, t.TempDir())
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "optimizer_runs = 5")
	require.NotContains(t, string(data), "rpc")
}

func TestClient_WithEnv(t *testing.T) {
	base := NewClient(stubScript(t, `echo "A=$OP_DEPLOYER_TEST_A B=$OP_DEPLOYER_TEST_B"`))
	base.Wd = stubProject(t)
	base.Stdout = io.Discard
	base = base.WithEnv("OP_DEPLOYER_TEST_A=1")
	withB := base.WithEnv("OP_DEPLOYER_TEST_B=2")

	out, _, err := withB.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=2\n"+scriptRan, out)

	out, _, err = base.RunScript(context.Background(), "Script.s.sol", "run()", nil)
	require.NoError(t, err)
	require.Equal(t, "A=1 B=\n"+scriptRan, out, "WithEnv must not change the client it copies")
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
