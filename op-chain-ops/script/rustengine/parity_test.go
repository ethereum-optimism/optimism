package rustengine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

const artifactsRel = "../testdata/test-artifacts"

// buildEngine returns the Rust engine binary path. It prefers a pre-built binary named by
// RUST_BINARY_PATH_OP_SCRIPT_ENGINE (how CI supplies it to the cargo-less Go executors); otherwise
// it cargo-builds locally, and skips only when neither a pre-built binary nor cargo is available.
func buildEngine(t *testing.T) string {
	if p, ok := PrebuiltEngineBinary(); ok {
		return p
	}
	if _, err := exec.LookPath("cargo"); err != nil {
		msg := "no pre-built engine (RUST_BINARY_PATH_OP_SCRIPT_ENGINE) and cargo unavailable"
		// CI sets REQUIRE_RUST_ENGINE so a missing binary fails the byte-parity gate loudly instead
		// of skipping it silently; local dev without cargo still skips.
		if os.Getenv("REQUIRE_RUST_ENGINE") != "" {
			t.Fatal(msg + " (REQUIRE_RUST_ENGINE is set)")
		}
		t.Skip(msg + "; skipping Rust engine parity test")
	}
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	rustDir := filepath.Join(root, "rust")
	cmd := exec.Command("cargo", "build", "-p", "op-script-engine", "--bin", "op-script-engine")
	cmd.Dir = rustDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cargo build op-script-engine failed: %v\n%s", err, out)
	}
	return filepath.Join(rustDir, "target", "debug", "op-script-engine")
}

func encodeStringCall(t *testing.T, method, input string) []byte {
	packer, err := abi.JSON(strings.NewReader(fmt.Sprintf(
		`[{"type":"function","name":"%s","inputs":[{"type":"string","name":"input"}]}]`, method)))
	require.NoError(t, err)
	data, err := packer.Pack(method, input)
	require.NoError(t, err)
	return data
}

func requireAllocsEqual(t *testing.T, label string, want, got *foundry.ForgeAllocs) {
	t.Helper()
	wb, err := json.MarshalIndent(want, "", "  ")
	require.NoError(t, err)
	gb, err := json.MarshalIndent(got, "", "  ")
	require.NoError(t, err)
	require.JSONEq(t, string(wb), string(gb), "state dump mismatch: %s", label)
}

func absArtifacts(t *testing.T) string {
	p, err := filepath.Abs(artifactsRel)
	require.NoError(t, err)
	return p
}

// TestRustEngineParity drives the two goldens (TestScriptStateDump, TestScriptBroadcast)
// against both the Go script host and the Rust engine and requires identical output.
func TestRustEngineParity(t *testing.T) {
	bin := buildEngine(t)
	art := absArtifacts(t)
	logw := testWriter{t}

	t.Run("stateDump", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		af := foundry.OpenArtifactsDir(artifactsRel)

		// Go host
		gh := script.NewHost(logger, af, nil, script.DefaultContext)
		require.NoError(t, gh.EnableCheats())
		gAddr, err := gh.LoadContract("ScriptExample.s.sol", "ScriptExample")
		require.NoError(t, err)
		gh.AllowCheatcodes(gAddr)

		// Rust engine
		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
		require.NoError(t, err)
		defer re.Close()
		rAddr, err := re.LoadContract("ScriptExample.s.sol", "ScriptExample", script.DefaultContext.Origin)
		require.NoError(t, err)
		require.Equal(t, gAddr, rAddr, "deploy address")
		require.NoError(t, re.AllowCheatcodes(rAddr))

		dumpBoth := func(label string) {
			gd, err := gh.StateDump()
			require.NoError(t, err)
			rd, err := re.StateDump()
			require.NoError(t, err)
			// Non-vacuity guard: a parity assertion over two empty dumps would pass trivially.
			require.NotEmpty(t, gd.Accounts, "%s: state dump must be non-empty", label)
			requireAllocsEqual(t, label, gd, rd)
		}

		dumpBoth("dump1")

		sender := script.DefaultContext.Sender
		for _, v := range []string{"call A", "call B"} {
			data := encodeStringCall(t, "call1", v)
			_, _, err := gh.Call(sender, gAddr, data, script.DefaultFoundryGasLimit, uint256.NewInt(0))
			require.NoError(t, err)
			_, err = re.Call(sender, rAddr, data)
			require.NoError(t, err)
			dumpBoth("dump after " + v)
		}
	})

	t.Run("broadcast", func(t *testing.T) {
		logger := testlog.Logger(t, log.LevelError)
		af := foundry.OpenArtifactsDir(artifactsRel)

		var goBroadcasts []script.Broadcast
		gh := script.NewHost(logger, af, nil, script.DefaultContext,
			script.WithBroadcastHook(func(b script.Broadcast) { goBroadcasts = append(goBroadcasts, b) }),
			script.WithCreate2Deployer())
		require.NoError(t, gh.EnableCheats())
		gAddr, err := gh.LoadContract("ScriptExample.s.sol", "ScriptExample")
		require.NoError(t, err)
		gh.AllowCheatcodes(gAddr)

		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337, Create2Deployer: true}, logw)
		require.NoError(t, err)
		defer re.Close()
		rAddr, err := re.LoadContract("ScriptExample.s.sol", "ScriptExample", script.DefaultContext.Origin)
		require.NoError(t, err)
		require.Equal(t, gAddr, rAddr, "deploy address")
		require.NoError(t, re.AllowCheatcodes(rAddr))

		senderAddr := common.HexToAddress("0x0000000000000000000000000000000000Badc0d")
		runBroadcast := []byte{0xbe, 0xf0, 0x3a, 0xbc}
		_, _, err = gh.Call(senderAddr, gAddr, runBroadcast, script.DefaultFoundryGasLimit, uint256.NewInt(0))
		require.NoError(t, err)
		rustBroadcasts, err := re.Call2Broadcasts(senderAddr, rAddr, runBroadcast)
		require.NoError(t, err)

		// Non-vacuity guard: runBroadcast emits a fixed set of broadcasts; an empty list on both
		// sides would make the parity assertion pass trivially.
		require.NotEmpty(t, goBroadcasts, "broadcast list must be non-empty")

		gb, err := json.MarshalIndent(goBroadcasts, "", "  ")
		require.NoError(t, err)
		rb, err := json.MarshalIndent(rustBroadcasts, "", "  ")
		require.NoError(t, err)
		require.JSONEq(t, string(gb), string(rb), "broadcast list mismatch")

		// Final nonce parity.
		for _, a := range []common.Address{
			senderAddr, gAddr,
			common.HexToAddress("0x0000000000000000000000000000000000C0FFEE"),
			common.HexToAddress("0xcafe"),
		} {
			rn, err := re.GetNonce(a)
			require.NoError(t, err)
			require.Equal(t, gh.GetNonce(a), rn, "nonce parity for %s", a)
		}
	})
}

// Call2Broadcasts runs a call then drains the captured broadcasts.
func (e *Engine) Call2Broadcasts(from, to common.Address, input []byte) ([]script.Broadcast, error) {
	if _, err := e.Call(from, to, input); err != nil {
		return nil, err
	}
	return e.TakeBroadcasts()
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", strings.TrimRight(string(p), "\n"))
	return len(p), nil
}

// TestRustEngineSetBalance exercises the Engine.SetBalance client + script_setBalance RPC (used by
// the L1 prefund-dev-genesis stage) and checks it matches the Go host's SetBalance byte-for-byte in
// the resulting state dump. SetBalance is not hit by the L2Genesis / L1-deploy parity legs, so this
// is its dedicated coverage.
func TestRustEngineSetBalance(t *testing.T) {
	bin := buildEngine(t)
	art := absArtifacts(t)
	logw := testWriter{t}
	logger := testlog.Logger(t, log.LevelError)
	af := foundry.OpenArtifactsDir(artifactsRel)

	addr := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	bal := new(uint256.Int).Mul(uint256.NewInt(7), uint256.NewInt(1_000_000_000_000_000_000)) // 7e18

	// Go host
	gh := script.NewHost(logger, af, nil, script.DefaultContext)
	require.NoError(t, gh.EnableCheats())
	gh.SetBalance(addr, bal)
	gd, err := gh.StateDump()
	require.NoError(t, err)

	// Rust engine
	re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
	require.NoError(t, err)
	defer re.Close()
	require.NoError(t, re.SetBalance(addr, bal))
	rd, err := re.StateDump()
	require.NoError(t, err)

	require.Contains(t, rd.Accounts, addr, "funded account must appear in the engine dump")
	require.Equal(t, bal.ToBig(), rd.Accounts[addr].Balance, "engine balance")
	requireAllocsEqual(t, "setBalance", gd, rd)
}
