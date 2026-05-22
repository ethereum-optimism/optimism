package opcm

import (
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestStaticInputMappingValidatesAgainstABI(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  owner:
    from: intent.owner
  amount:
    coalesce:
      - from: chain.amount
      - value: 1
    transform: bigint
  enabled:
    value: false
  root:
    value: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games:
    value: []
`))
	require.NoError(t, err)

	require.NoError(t, ValidateStaticInputMapping(contractABI, *mapping))
}

func TestStaticInputMappingRejectsMissingABIInput(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  owner:
    from: intent.owner
`))
	require.NoError(t, err)

	require.ErrorContains(t, ValidateStaticInputMapping(contractABI, *mapping), `missing ABI input "amount"`)
}

func TestStaticInputMappingRejectsUnknownTarget(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  owner:
    from: intent.owner
  amount:
    value: 1
  enabled:
    value: false
  root:
    value: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  games:
    value: []
  extra:
    value: true
`))
	require.NoError(t, err)

	require.ErrorContains(t, ValidateStaticInputMapping(contractABI, *mapping), `unknown ABI input "extra"`)
}

func TestEvaluateStaticInputMappingForABI(t *testing.T) {
	contractABI, err := abi.JSON(strings.NewReader(inputYAMLTestABI))
	require.NoError(t, err)

	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  owner:
    from: intent.superchainRoles.owner
  amount:
    coalesce:
      - from: chain.deployOverrides.amount
      - from: intent.globalDeployOverrides.amount
      - value: 1
    transform: bigint
  enabled:
    from: chain
    transform: isCustomGasTokenEnabled
  root:
    from: state.create2Salt
  games:
    value: []
`))
	require.NoError(t, err)

	type roles struct {
		Owner common.Address `json:"owner"`
	}
	type intentSource struct {
		SuperchainRoles      roles          `json:"superchainRoles"`
		GlobalDeployOverride map[string]any `json:"globalDeployOverrides"`
	}
	type chainSource struct {
		DeployOverrides map[string]any `json:"deployOverrides"`
		CustomGasToken  struct {
			Name   string `json:"name"`
			Symbol string `json:"symbol"`
		} `json:"customGasToken"`
	}
	type stateSource struct {
		Create2Salt common.Hash `json:"create2Salt"`
	}

	var chain chainSource
	chain.CustomGasToken.Name = "Token"
	chain.CustomGasToken.Symbol = "TOK"
	intent := intentSource{
		SuperchainRoles: roles{
			Owner: common.HexToAddress("0x1111111111111111111111111111111111111111"),
		},
		GlobalDeployOverride: map[string]any{
			"amount": "2",
		},
	}
	state := stateSource{
		Create2Salt: common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
	}

	input, err := EvaluateStaticInputMappingForABI(contractABI, *mapping, StaticInputSources{
		"intent": intent,
		"chain":  &chain,
		"state":  state,
	})
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0x1111111111111111111111111111111111111111"), input.Address("owner"))
	require.Equal(t, big.NewInt(2), input["amount"])
	require.Equal(t, true, input["enabled"])
	require.Equal(t, common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), input["root"])

	_, err = packMethodInput(contractABI, contractABI.Methods["run"], input)
	require.NoError(t, err)
}

func TestEvaluateStaticInputMappingMissingRequiredPath(t *testing.T) {
	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  owner:
    from: intent.owner
`))
	require.NoError(t, err)

	_, err = EvaluateStaticInputMapping(*mapping, StaticInputSources{"intent": map[string]any{}})
	require.ErrorContains(t, err, `evaluate static input "owner": no value resolved`)
}

func TestEvaluateStaticInputMappingCoalesceFallsBack(t *testing.T) {
	mapping, err := LoadStaticInputMappingYAML([]byte(`
version: 1
kind: legacy-input-mapping
script:
  artifact: Example.s.sol/Example.json
  contract: Example
  function: run
input:
  amount:
    coalesce:
      - from: chain.deployOverrides.amount
      - value: 7
    transform: bigint
`))
	require.NoError(t, err)

	input, err := EvaluateStaticInputMapping(*mapping, StaticInputSources{
		"chain": map[string]any{"deployOverrides": map[string]any{}},
	})
	require.NoError(t, err)
	require.Equal(t, big.NewInt(7), input["amount"])
}
