package rustengine

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

// The *Golden tests below are the goldenized form of the corresponding *Parity tests: instead of
// comparing the Rust engine's output to a live Go host, they compare it to a committed fixture that
// was recorded from the Go host at the base commit (see testdata/goldens/README.md). They are the
// safety net that survives the deletion of the Go script engine — the *Parity tests exist only while
// the Go host does, and prove (by also passing) that these fixtures faithfully capture it.

// TestRustEngineStateDumpGolden goldenizes TestRustEngineParity/stateDump: deploy ScriptExample, then
// call1("call A") and call1("call B"), dumping state after each, and pin each dump.
func TestRustEngineStateDumpGolden(t *testing.T) {
	bin := buildEngine(t)
	art := absArtifacts(t)
	logw := testWriter{t}

	re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
	require.NoError(t, err)
	defer re.Close()
	rAddr, err := re.LoadContract("ScriptExample.s.sol", "ScriptExample", script.DefaultContext.Origin)
	require.NoError(t, err)
	require.NoError(t, re.AllowCheatcodes(rAddr))

	dump := func(golden string) {
		rd, err := re.StateDump()
		require.NoError(t, err)
		// Non-vacuity guard (carried over from the parity test): a match over two empty dumps is trivial.
		require.NotEmpty(t, rd.Accounts, "%s: state dump must be non-empty", golden)
		requireJSONMatchesGolden(t, golden, rd)
	}

	dump("scriptexample.dump1.json")

	sender := script.DefaultContext.Sender
	for i, v := range []string{"call A", "call B"} {
		data := encodeStringCall(t, "call1", v)
		_, err := re.Call(sender, rAddr, data)
		require.NoError(t, err)
		dump([]string{"scriptexample.dumpA.json", "scriptexample.dumpB.json"}[i])
	}
}

// TestRustEngineBroadcastGolden goldenizes TestRustEngineParity/broadcast: run runBroadcast() and pin
// the broadcast bundle + the final nonces.
func TestRustEngineBroadcastGolden(t *testing.T) {
	bin := buildEngine(t)
	art := absArtifacts(t)
	logw := testWriter{t}

	re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337, Create2Deployer: true}, logw)
	require.NoError(t, err)
	defer re.Close()
	rAddr, err := re.LoadContract("ScriptExample.s.sol", "ScriptExample", script.DefaultContext.Origin)
	require.NoError(t, err)
	require.NoError(t, re.AllowCheatcodes(rAddr))

	senderAddr := common.HexToAddress("0x0000000000000000000000000000000000Badc0d")
	runBroadcast := []byte{0xbe, 0xf0, 0x3a, 0xbc}
	rustBroadcasts, err := re.Call2Broadcasts(senderAddr, rAddr, runBroadcast)
	require.NoError(t, err)

	// Non-vacuity guard (carried over): runBroadcast emits a fixed non-empty set of broadcasts.
	require.NotEmpty(t, rustBroadcasts, "broadcast list must be non-empty")
	requireJSONMatchesGolden(t, "scriptexample.broadcasts.json", rustBroadcasts)

	// Final-nonce parity: the four addresses the parity test checked, pinned by fixture.
	nonces := map[string]uint64{}
	for _, a := range []common.Address{
		senderAddr, rAddr,
		common.HexToAddress("0x0000000000000000000000000000000000C0FFEE"),
		common.HexToAddress("0xcafe"),
	} {
		n, err := re.GetNonce(a)
		require.NoError(t, err)
		nonces[a.Hex()] = n
	}
	requireJSONMatchesGolden(t, "scriptexample.nonces.json", nonces)
}

// TestRustEngineSetBalanceGolden goldenizes TestRustEngineSetBalance.
func TestRustEngineSetBalanceGolden(t *testing.T) {
	bin := buildEngine(t)
	art := absArtifacts(t)
	logw := testWriter{t}

	addr := common.HexToAddress("0x00000000000000000000000000000000000000aa")
	bal := new(uint256.Int).Mul(uint256.NewInt(7), uint256.NewInt(1_000_000_000_000_000_000)) // 7e18

	re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
	require.NoError(t, err)
	defer re.Close()
	require.NoError(t, re.SetBalance(addr, bal))
	rd, err := re.StateDump()
	require.NoError(t, err)

	// Non-vacuity guards (carried over): the funded account must be present with the exact balance.
	require.Contains(t, rd.Accounts, addr, "funded account must appear in the engine dump")
	require.Equal(t, bal.ToBig(), rd.Accounts[addr].Balance, "engine balance")
	requireJSONMatchesGolden(t, "setbalance.dump.json", rd)
}

// TestRustEngineOPCMGolden goldenizes TestRustEngineOPCMParity: the OPCM RunScriptSingle (input+output)
// and RunScriptVoid (input-only) precompile paths through the Rust engine, pinned to fixture.
func TestRustEngineOPCMGolden(t *testing.T) {
	bin := buildEngine(t)
	art := opcmArtifactsAbs(t)
	logw := testWriter{t}

	input := OPCMExampleInput{
		Owner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Blob:  []byte("hello-opcm-parity"),
	}
	target := common.HexToAddress("0x0000000000000000000000000000000000C0FFEE")

	t.Run("single", func(t *testing.T) {
		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
		require.NoError(t, err)
		defer re.Close()
		rustOutput, err := RunScriptSingle[OPCMExampleInput, OPCMExampleOutput](
			re, input, "OPCMExample.s.sol", "OPCMExample", script.DefaultContext.Origin)
		require.NoError(t, err, "rust RunScriptSingle")
		rustDump, err := re.StateDump()
		require.NoError(t, err)

		// Non-vacuity guards (carried over): the output must be populated and the mutated account present.
		require.Equal(t, target, rustOutput.Result, "output.Result must be populated via the setter-capture replay")
		require.Contains(t, rustDump.Accounts, target, "dump must contain the mutated TARGET account")
		requireJSONMatchesGolden(t, "opcm.single.output.json", rustOutput)
		requireJSONMatchesGolden(t, "opcm.single.dump.json", rustDump)
	})

	t.Run("void", func(t *testing.T) {
		re, err := Spawn(bin, SpawnOpts{ArtifactsDir: art, ChainID: 1337}, logw)
		require.NoError(t, err)
		defer re.Close()
		require.NoError(t, RunScriptVoid[OPCMExampleInput](
			re, input, "OPCMExample.s.sol", "OPCMExampleVoid", script.DefaultContext.Origin), "rust RunScriptVoid")
		rustDump, err := re.StateDump()
		require.NoError(t, err)

		require.Contains(t, rustDump.Accounts, target, "dump must contain the mutated TARGET account")
		requireJSONMatchesGolden(t, "opcm.void.dump.json", rustDump)
	})
}

// opcmArtifactsAbs is the absolute path to the compiled OPCMExample synthetic Family-B script.
func opcmArtifactsAbs(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs(opcmArtifactsRel)
	require.NoError(t, err)
	return p
}
