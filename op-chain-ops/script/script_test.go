package script

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

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
	h := NewHost(logger, af, nil, scriptContext)
	addr, err := h.LoadContract("ScriptExample.s.sol", "ScriptExample")
	require.NoError(t, err)

	require.NoError(t, h.EnableCheats())

	h.SetEnvVar("EXAMPLE_BOOL", "true")
	input := bytes4("run()")
	returnData, _, err := h.Call(scriptContext.Sender, addr, input[:], DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err, "call failed: %x", string(returnData))
	require.NotNil(t, captLog.FindLog(testlog.NewMessageFilter("sender nonce 1")))

	require.NoError(t, h.cheatcodes.Precompile.DumpState("noop"))
	// and a second time, to see if we can revisit the host state.
	require.NoError(t, h.cheatcodes.Precompile.DumpState("noop"))
}

func TestScriptBroadcast(t *testing.T) {
	logger := testlog.Logger(t, log.LevelDebug)
	af := foundry.OpenArtifactsDir("./testdata/test-artifacts")

	mustEncodeCalldata := func(method, input string) []byte {
		packer, err := abi.JSON(strings.NewReader(fmt.Sprintf(`[{"type":"function","name":"%s","inputs":[{"type":"string","name":"input"}]}]`, method)))
		require.NoError(t, err)

		data, err := packer.Pack(method, input)
		require.NoError(t, err)
		return data
	}

	senderAddr := common.HexToAddress("0x5b73C5498c1E3b4dbA84de0F1833c4a029d90519")
	expBroadcasts := []Broadcast{
		{
			From:     senderAddr,
			To:       senderAddr,
			Calldata: mustEncodeCalldata("call1", "single_call1"),
		},
		{
			From:     senderAddr,
			To:       senderAddr,
			Calldata: mustEncodeCalldata("call1", "startstop_call1"),
		},
		{
			From:     senderAddr,
			To:       senderAddr,
			Calldata: mustEncodeCalldata("call2", "startstop_call2"),
		},
		{
			From:     senderAddr,
			To:       senderAddr,
			Calldata: mustEncodeCalldata("nested1", "nested"),
		},
	}

	scriptContext := DefaultContext
	var broadcasts []Broadcast
	hook := func(broadcast Broadcast) {
		broadcasts = append(broadcasts, broadcast)
	}
	h := NewHost(logger, af, nil, scriptContext, WithBroadcastHook(hook))
	addr, err := h.LoadContract("ScriptExample.s.sol", "ScriptExample")
	require.NoError(t, err)

	require.NoError(t, h.EnableCheats())

	input := bytes4("runBroadcast()")
	returnData, _, err := h.Call(scriptContext.Sender, addr, input[:], DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err, "call failed: %x", string(returnData))
	require.EqualValues(t, expBroadcasts, broadcasts)
}
