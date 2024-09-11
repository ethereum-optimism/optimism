package script

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

	fooBar, err := af.ReadArtifact("ScriptExample.s.sol", "FooBar")
	require.NoError(t, err)

	expectedInitCode := bytes.Clone(fooBar.Bytecode.Object)
	// Add the contract init argument we use in the script
	expectedInitCode = append(expectedInitCode, leftPad32(big.NewInt(1234).Bytes())...)
	salt := uint256.NewInt(42).Bytes32()

	senderAddr := common.HexToAddress("0x5b73C5498c1E3b4dbA84de0F1833c4a029d90519")
	coffeeAddr := common.HexToAddress("0x0000000000000000000000000000000000C0FFEE")
	cafeAddr := common.HexToAddress("0xcafe")
	expBroadcasts := []Broadcast{
		{
			From:    senderAddr,
			To:      senderAddr,
			Input:   mustEncodeCalldata("call1", "single_call1"),
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			GasUsed: 23421,
			Type:    BroadcastCall,
		},
		{
			From:    coffeeAddr,
			To:      senderAddr,
			Input:   mustEncodeCalldata("call1", "startstop_call1"),
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			GasUsed: 1521,
			Type:    BroadcastCall,
		},
		{
			From:    coffeeAddr,
			To:      senderAddr,
			Input:   mustEncodeCalldata("call2", "startstop_call2"),
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			GasUsed: 1565,
			Type:    BroadcastCall,
		},
		{
			From:    common.HexToAddress("0x1234"),
			To:      senderAddr,
			Input:   mustEncodeCalldata("nested1", "nested"),
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			GasUsed: 2763,
			Type:    BroadcastCall,
		},
		{
			From:    common.HexToAddress("0x123456"),
			To:      crypto.CreateAddress(common.HexToAddress("0x123456"), 0),
			Input:   expectedInitCode,
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			GasUsed: 39112,
			Type:    BroadcastCreate,
		},
		{
			From:    cafeAddr,
			To:      crypto.CreateAddress2(cafeAddr, salt, crypto.Keccak256(expectedInitCode)),
			Input:   expectedInitCode,
			Value:   (*hexutil.U256)(uint256.NewInt(0)),
			Type:    BroadcastCreate2,
			GasUsed: 39112,
			Salt:    salt,
		},
	}

	var broadcasts []Broadcast
	hook := func(broadcast Broadcast) {
		broadcasts = append(broadcasts, broadcast)
	}
	h := NewHost(logger, af, nil, DefaultContext, WithBroadcastHook(hook))
	addr, err := h.LoadContract("ScriptExample.s.sol", "ScriptExample")
	require.NoError(t, err)

	require.NoError(t, h.EnableCheats())

	input := bytes4("runBroadcast()")
	returnData, _, err := h.Call(senderAddr, addr, input[:], DefaultFoundryGasLimit, uint256.NewInt(0))
	require.NoError(t, err, "call failed: %x", string(returnData))

	expected, err := json.MarshalIndent(expBroadcasts, "  ", "  ")
	require.NoError(t, err)
	got, err := json.MarshalIndent(broadcasts, "  ", "  ")
	require.NoError(t, err)
	require.Equal(t, string(expected), string(got))

	// Assert that the nonces for accounts participating in the
	// broadcast increase. The senderAddr check is set to 2 to
	// account for the initial deployment of the contract.
	require.EqualValues(t, 2, h.GetNonce(senderAddr))
	require.EqualValues(t, 2, h.GetNonce(coffeeAddr))
	require.EqualValues(t, 1, h.GetNonce(cafeAddr))
}
