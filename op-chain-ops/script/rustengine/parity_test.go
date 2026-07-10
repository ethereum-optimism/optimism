package rustengine

import (
	"encoding/json"
	"fmt"
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

// buildEngine compiles the Rust engine (a no-op if already built) and returns the binary path.
func buildEngine(t *testing.T) string {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not available; skipping Rust engine parity test")
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
		re, err := Spawn(bin, art, 1337, false, logw)
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

		re, err := Spawn(bin, art, 1337, true, logw)
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
