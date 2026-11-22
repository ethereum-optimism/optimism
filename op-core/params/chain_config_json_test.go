package params

import (
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func jsonTestConfig(t *testing.T, s forkstest.Schedule) *ChainConfig {
	cfg := &ChainConfig{
		ChainID: big.NewInt(901),
		Optimism: &OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New[uint64](250),
		},
		BedrockBlock: big.NewInt(0),
	}
	for _, fork := range forkstest.TimestampELForks {
		_, ok := reflect.TypeFor[ChainConfig]().FieldByName(forkstest.Title(fork) + "Time")
		require.Truef(t, ok, "ChainConfig lacks field %sTime", forkstest.Title(fork))
		cfg.SetActivationTime(fork, s(fork))
	}
	return cfg
}

func requireConfigEqual(t *testing.T, want, got *ChainConfig) {
	t.Helper()
	bigEq := cmp.Comparer(func(a, b *big.Int) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Cmp(b) == 0
	})
	require.Empty(t, cmp.Diff(want, got, bigEq))
}

func parseJSONObject(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	require.NoError(t, json.Unmarshal(data, &obj))
	return obj
}

// TestChainConfigJSONKeysDisjoint checks that go-ethereum's chain config declares none of
// the ChainConfig JSON keys but chainId: encoding/json drops a key that both embedded
// configs of chainConfigJSON declare.
func TestChainConfigJSONKeysDisjoint(t *testing.T) {
	gethKeys := map[string]bool{}
	gethType := reflect.TypeFor[gethparams.ChainConfig]()
	for i := range gethType.NumField() {
		name, _, _ := strings.Cut(gethType.Field(i).Tag.Get("json"), ",")
		gethKeys[name] = true
	}
	cfgType := reflect.TypeFor[ChainConfig]()
	for i := range cfgType.NumField() {
		name, _, _ := strings.Cut(cfgType.Field(i).Tag.Get("json"), ",")
		if name == "chainId" {
			continue
		}
		require.Falsef(t, gethKeys[name], "go-ethereum's chain config declares %q", name)
	}
}

func TestChainConfigJSONRoundTrip(t *testing.T) {
	opMainnet := jsonTestConfig(t, forkstest.Staggered())
	opMainnet.ChainID = big.NewInt(OPMainnetChainID)
	opMainnet.BedrockBlock = big.NewInt(OPMainnetGenesisBlockNum)
	for name, cfg := range map[string]*ChainConfig{
		"staggered":    jsonTestConfig(t, forkstest.Staggered()),
		"OP Mainnet":   opMainnet,
		"no OP config": {ChainID: big.NewInt(901)},
	} {
		t.Run(name, func(t *testing.T) {
			data, err := json.Marshal(cfg)
			require.NoError(t, err)
			byValue, err := json.Marshal(*cfg)
			require.NoError(t, err)
			require.Equal(t, data, byValue)

			var got ChainConfig
			require.NoError(t, json.Unmarshal(data, &got))
			requireConfigEqual(t, cfg, &got)

			again, err := json.Marshal(&got)
			require.NoError(t, err)
			require.Equal(t, data, again)
		})
	}
}

// TestChainConfigJSONUnmarshalEthereumKeys checks that a decoded Ethereum fork key must
// agree with the schedule derived from the OP fields, while absent and unknown keys are
// accepted.
func TestChainConfigJSONUnmarshalEthereumKeys(t *testing.T) {
	data, err := json.Marshal(jsonTestConfig(t, forkstest.AtGenesisThrough(forks.Jovian)))
	require.NoError(t, err)
	decodeEdited := func(edit func(obj map[string]any)) error {
		obj := parseJSONObject(t, data)
		edit(obj)
		edited, err := json.Marshal(obj)
		require.NoError(t, err)
		var got ChainConfig
		return json.Unmarshal(edited, &got)
	}

	require.NoError(t, decodeEdited(func(o map[string]any) { o["terminalTotalDifficultyPassed"] = true }),
		"unknown keys are ignored")
	require.NoError(t, decodeEdited(func(o map[string]any) {
		delete(o, "shanghaiTime")
		delete(o, "homesteadBlock")
		delete(o, "terminalTotalDifficulty")
	}), "derived keys may be absent")

	for key, value := range map[string]any{
		"shanghaiTime":            1,
		"osakaTime":               9000,
		"bpo1Time":                0,
		"bpo2Time":                0,
		"bpo3Time":                0,
		"bpo4Time":                0,
		"bpo5Time":                0,
		"amsterdamTime":           0,
		"bogotaTime":              0,
		"ubtTime":                 0,
		"enableUBTAtGenesis":      true,
		"londonBlock":             5,
		"daoForkBlock":            0,
		"terminalTotalDifficulty": 1,
		"daoForkSupport":          true,
		"depositContractAddress":  common.Address{0x01}.Hex(),
	} {
		t.Run(key, func(t *testing.T) {
			err := decodeEdited(func(o map[string]any) { o[key] = value })
			require.ErrorContains(t, err, `"`+key+`"`)
		})
	}
}

func TestChainConfigJSONNilOptimism(t *testing.T) {
	cfg := jsonTestConfig(t, forkstest.Staggered())
	cfg.Optimism = nil
	data, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NotContains(t, parseJSONObject(t, data), "optimism")

	var got ChainConfig
	require.NoError(t, json.Unmarshal(data, &got))
	require.Nil(t, got.Optimism)
}
