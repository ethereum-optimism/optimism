package script

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

//go:generate ./testdata/generate.sh

func TestScript(t *testing.T) {
	logger, captLog := testlog.CaptureLogger(t, log.LevelInfo)
	af := foundry.OpenArtifactsDir("./testdata/test-artifacts")

	scriptContext := DefaultContext
	h := NewHost(logger, af, scriptContext)
	addr, err := h.LoadContract("ScriptExample.s", "ScriptExample")
	require.NoError(t, err)

	require.NoError(t, h.EnableCheats())

	h.SetEnvVar("EXAMPLE_BOOL", "true")
	input := bytes4("run()")
	returnData, _, err := h.Call(scriptContext.sender, addr, input[:], DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err, "call failed: %x", string(returnData))
	require.NotNil(t, captLog.FindLog(
		testlog.NewAttributesFilter("p0", "sender nonce"),
		testlog.NewAttributesFilter("p1", "1")))

	require.NoError(t, h.cheatcodes.Precompile.DumpState("noop"))
	// and a second time, to see if we can revisit the host state.
	require.NoError(t, h.cheatcodes.Precompile.DumpState("noop"))
}
