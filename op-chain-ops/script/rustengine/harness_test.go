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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
)

// This file holds the engine-only test helpers shared by the *Golden gates. They were relocated
// here from the deleted *Parity test files (which also drove the now-removed Go script host) so the
// golden safety net keeps compiling after the Go engine's deletion. Everything here is pure
// engine/JSON/crypto machinery — no dependency on the deleted script.Host.

const artifactsRel = "../testdata/test-artifacts"

// opcmArtifactsRel points at the compiled OPCMExample synthetic Family-B script (see
// testdata/scripts/OPCMExample.s.sol).
const opcmArtifactsRel = "testdata/test-artifacts"

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
		if RequireEngine() {
			t.Fatal(msg + " (" + RequireEngineEnv + " is set)")
		}
		t.Skip(msg + "; skipping Rust engine golden test")
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

func absArtifacts(t *testing.T) string {
	p, err := filepath.Abs(artifactsRel)
	require.NoError(t, err)
	return p
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

// OPCMExampleInput / OPCMExampleOutput are the Go input/output structs for OPCMExample. Their
// exported fields map to the Solidity getters/setter (owner(), blob(), result()) exactly as the
// real op-deployer OPCM input/output structs (e.g. manage.ScriptInput / InteropMigrationOutput).
type OPCMExampleInput struct {
	Owner common.Address
	Blob  []byte
}

type OPCMExampleOutput struct {
	Result common.Address
}

// The reference fork: OP Sepolia's DisputeGameFactory on L1 Sepolia, pinned to an immutable block.
// version()=1.6.1, owner=deployer, gameImpls(999)=0 at this block (verified when the fixture was
// recorded), so SetDisputeGameImpl runs its full broadcast+assert path.
const (
	forkBlock    = uint64(11250000) // 0xabae10
	forkGameType = uint32(999)
)

var (
	forkDGF      = common.HexToAddress("0x05F9613aDB30026FFd634f38e5C4dFd30a197Fa1")
	forkDeployer = common.HexToAddress("0x1Eb2fFc903729a0F03966B917003800b145F56E2") // DGF owner
)

func versionSelector() []byte { return crypto.Keccak256([]byte("version()"))[:4] }

func decodeVersion(t *testing.T, data []byte) string {
	t.Helper()
	decoded, err := (abi.Arguments{{Type: abi.Type{T: abi.StringTy}}}).Unpack(data)
	require.NoError(t, err)
	return decoded[0].(string)
}

func mustCall(t *testing.T, e *Engine, from, to common.Address, input []byte) []byte {
	t.Helper()
	out, err := e.Call(from, to, input)
	require.NoError(t, err)
	return out
}

func ptrU64(v uint64) *uint64 { return &v }

// pruneForkScaffolding removes the script-scaffolding accounts from a fork-diff-shaped JSON blob, so
// a golden comparison covers the real L1 fork writes rather than the deterministic deploy machinery.
// The set matches the Rust engine's built-in fork-diff exclusion: DefaultSender, VMAddr, Console,
// Script/ForgeDeployer, and the whole ScriptDeployer CREATE range (the input/output precompiles + the
// script contract). Applied symmetrically; a no-op on the already-pruned Rust diff.
func pruneForkScaffolding(t *testing.T, raw []byte) []byte {
	t.Helper()
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &top))
	var acct map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(top["account"], &acct))
	for _, a := range forkScaffoldingAddrs() {
		delete(acct, strings.ToLower(a.Hex()))
	}
	reAcct, err := json.Marshal(acct)
	require.NoError(t, err)
	top["account"] = reAcct
	out, err := json.Marshal(top)
	require.NoError(t, err)
	return out
}

func forkScaffoldingAddrs() []common.Address {
	out := []common.Address{
		addresses.DefaultSenderAddr, addresses.VMAddr, addresses.ConsoleAddr,
		addresses.ScriptDeployer, addresses.ForgeDeployer,
	}
	// A RunScriptVoid mints ≤3 script addresses; 8 is a safe upper bound on the CREATE range.
	for i := uint64(0); i < 8; i++ {
		out = append(out, crypto.CreateAddress(addresses.ScriptDeployer, i))
	}
	return out
}
