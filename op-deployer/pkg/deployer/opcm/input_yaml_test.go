package opcm

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/stretchr/testify/require"
)

const inputYAMLTestABI = `[
  {
    "type": "function",
    "name": "run",
    "inputs": [
      {
        "name": "_input",
        "type": "tuple",
        "components": [
          {"name": "owner", "type": "address"},
          {"name": "amount", "type": "uint256"},
          {"name": "enabled", "type": "bool"},
          {"name": "root", "type": "bytes32"},
          {
            "name": "games",
            "type": "tuple[]",
            "components": [
              {"name": "gameType", "type": "uint32"},
              {"name": "impl", "type": "address"}
            ]
          }
        ]
      }
    ],
    "outputs": [
      {
        "name": "",
        "type": "tuple",
        "components": [
          {"name": "deployed", "type": "address"}
        ]
      }
    ]
  }
]`

func TestLoadScriptInputYAMLValidatesAgainstABI(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	input, err := LoadScriptInputYAML(contractABI, "run", []byte(`
input:
  owner: "0x1111111111111111111111111111111111111111"
  amount: 123456789012345678901234567890
  enabled: true
  root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games:
    - gameType: 0
      impl: "0x2222222222222222222222222222222222222222"
`))
	require.NoError(t, err)
	require.Equal(t, "0x1111111111111111111111111111111111111111", input["owner"])
	require.Len(t, input["games"], 1)

	_, err = packMethodInput(contractABI, contractABI.Methods["run"], input)
	require.NoError(t, err)
}

func TestLoadScriptInputYAMLAcceptsV2EnvelopeWithOptionalVersion(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	data := []byte(`
version: 2
input:
  owner: "0x1111111111111111111111111111111111111111"
  amount: 123456789012345678901234567890
  enabled: true
  root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games: []
`)
	isNative, err := IsNativeScriptInputYAML(data)
	require.NoError(t, err)
	require.True(t, isNative)

	input, err := LoadScriptInputYAML(contractABI, "run", data)
	require.NoError(t, err)
	require.NotContains(t, input, "version")
	require.Equal(t, "0x1111111111111111111111111111111111111111", input["owner"])
}

func TestLoadScriptInputYAMLAcceptsV2EnvelopeWithoutVersion(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	data := []byte(`
input:
  owner: "0x1111111111111111111111111111111111111111"
  amount: 1
  enabled: true
  root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games: []
`)
	isNative, err := IsNativeScriptInputYAML(data)
	require.NoError(t, err)
	require.True(t, isNative)

	input, err := LoadScriptInputYAML(contractABI, "run", data)
	require.NoError(t, err)
	require.Equal(t, "1", input["amount"])
}

func TestLoadScriptInputYAMLRejectsV2EnvelopeExtraField(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	_, err = LoadScriptInputYAML(contractABI, "run", []byte(`
version: 2
input:
  owner: "0x1111111111111111111111111111111111111111"
  amount: 1
  enabled: true
  root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games: []
legacy: true
`))
	require.ErrorContains(t, err, `v2 input envelope supports only "input" and optional "version" fields`)
}

func TestIsNativeScriptInputYAMLFalseWithoutInputEnvelope(t *testing.T) {
	isNative, err := IsNativeScriptInputYAML([]byte(`
owner: "0x1111111111111111111111111111111111111111"
amount: 1
`))
	require.NoError(t, err)
	require.False(t, isNative)
}

func TestLoadScriptInputYAMLRejectsUnknownField(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	_, err = LoadScriptInputYAML(contractABI, "run", []byte(`
owner: "0x1111111111111111111111111111111111111111"
amount: 1
enabled: true
root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
games: []
extra: true
`))
	require.ErrorContains(t, err, `unknown ABI input "extra"`)
}

func TestLoadScriptInputYAMLRejectsMissingField(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	_, err = LoadScriptInputYAML(contractABI, "run", []byte(`
owner: "0x1111111111111111111111111111111111111111"
amount: 1
enabled: true
root: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
`))
	require.ErrorContains(t, err, `missing required ABI input "games"`)
}
