package params

import (
	"bytes"
	"encoding/json"
	"math/big"
	"reflect"
	"slices"
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

// opGethTestConfig is the op-geth chain config equivalent to jsonTestConfig, built from
// op-geth's own OP Stack test config independently of GethChainConfig.
func opGethTestConfig(t *testing.T, s forkstest.Schedule) *gethparams.ChainConfig {
	cfg := *gethparams.OptimismTestConfig
	cfg.ChainID = big.NewInt(901)
	// OptimismTestConfig inherits an ethash config from go-ethereum's merged test config;
	// the op-geth config that op-chain-ops writes has no consensus engine config.
	cfg.Ethash = nil
	cfg.Optimism = &gethparams.OptimismConfig{
		EIP1559Elasticity:        6,
		EIP1559Denominator:       50,
		EIP1559DenominatorCanyon: ptr.New[uint64](250),
	}
	fields := reflect.ValueOf(&cfg).Elem()
	for _, fork := range forkstest.TimestampELForks {
		name := forkstest.Title(fork) + "Time"
		field := fields.FieldByName(name)
		require.Truef(t, field.IsValid(), "op-geth ChainConfig lacks field %s", name)
		field.Set(reflect.ValueOf(s(fork)))
	}
	cfg.ShanghaiTime = s(forks.Canyon)
	cfg.CancunTime = s(forks.Ecotone)
	cfg.PragueTime = s(forks.Isthmus)
	cfg.OsakaTime = s(forks.Karst)
	return &cfg
}

var jsonTestCases = []struct {
	name     string
	schedule forkstest.Schedule
}{
	{name: "pre-Karst", schedule: forkstest.AtGenesisThrough(forks.Jovian)},
	{name: "Karst", schedule: forkstest.Override(forkstest.AtGenesisThrough(forks.Jovian), forks.Karst, ptr.New[uint64](9000))},
	{name: "staggered", schedule: forkstest.Staggered()},
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

// objectKeys returns the keys of the JSON object data in encoding order.
func objectKeys(t *testing.T, data []byte) []string {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	require.NoError(t, err)
	require.Equal(t, json.Delim('{'), tok)
	var keys []string
	for dec.More() {
		tok, err := dec.Token()
		require.NoError(t, err)
		keys = append(keys, tok.(string))
		var value json.RawMessage
		require.NoError(t, dec.Decode(&value))
	}
	return keys
}

// TestChainConfigJSONFields checks that chainConfigJSON declares every ChainConfig field
// directly, with the same type and JSON tag. A field missing from the direct list would
// otherwise be encoded from go-ethereum's config, or dropped.
func TestChainConfigJSONFields(t *testing.T) {
	cfgType := reflect.TypeFor[ChainConfig]()
	jsonType := reflect.TypeFor[chainConfigJSON]()
	for i := range cfgType.NumField() {
		f := cfgType.Field(i)
		if !f.IsExported() {
			continue
		}
		jf, ok := jsonType.FieldByName(f.Name)
		require.True(t, ok, "chainConfigJSON lacks %s", f.Name)
		require.Len(t, jf.Index, 1, "chainConfigJSON must declare %s directly", f.Name)
		require.Equal(t, f.Type, jf.Type, f.Name)
		require.Equal(t, f.Tag.Get("json"), jf.Tag.Get("json"), f.Name)
	}
	for i := range jsonType.NumField() {
		jf := jsonType.Field(i)
		if jf.Anonymous {
			continue
		}
		_, ok := cfgType.FieldByName(jf.Name)
		require.True(t, ok, "chainConfigJSON field %s is not a ChainConfig field", jf.Name)
	}
}

// TestChainConfigJSONDirectFieldsDominate checks that the direct fields of
// chainConfigJSON hide the same-named fields of the embedded go-ethereum config, even
// where a direct field is omitted.
func TestChainConfigJSONDirectFieldsDominate(t *testing.T) {
	geth := jsonTestConfig(t, forkstest.Staggered()).GethChainConfig()
	geth.ChainID = big.NewInt(1)
	geth.CanyonTime = ptr.New[uint64](999)
	geth.LagoonTime = ptr.New[uint64](999)
	geth.Optimism = &gethparams.OptimismConfig{EIP1559Elasticity: 1, EIP1559Denominator: 1}

	data, err := json.Marshal(chainConfigJSON{
		ChainID:     big.NewInt(901),
		ChainConfig: geth,
		CanyonTime:  ptr.New[uint64](2000),
		Optimism:    &OptimismConfig{EIP1559Elasticity: 6, EIP1559Denominator: 50},
	})
	require.NoError(t, err)
	obj := parseJSONObject(t, data)
	require.Equal(t, float64(901), obj["chainId"])
	require.Equal(t, float64(2000), obj["canyonTime"])
	require.NotContains(t, obj, "lagoonTime")
	require.Equal(t, map[string]any{"eip1559Elasticity": float64(6), "eip1559Denominator": float64(50)}, obj["optimism"])
	require.Equal(t, float64(0), obj["homesteadBlock"], "go-ethereum-only keys are kept")
}

// TestChainConfigJSONMatchesOpGeth pins the encoding to op-geth's encoding of the
// equivalent op-geth chain config.
func TestChainConfigJSONMatchesOpGeth(t *testing.T) {
	for _, tc := range jsonTestCases {
		t.Run(tc.name, func(t *testing.T) {
			want, err := json.Marshal(opGethTestConfig(t, tc.schedule))
			require.NoError(t, err)
			got, err := json.Marshal(jsonTestConfig(t, tc.schedule))
			require.NoError(t, err)
			require.Equal(t, parseJSONObject(t, want), parseJSONObject(t, got))
		})
	}
}

// TestChainConfigJSONKeyOrder checks that keys are encoded in op-geth's order, except
// that the OP Stack keys follow all Ethereum keys: op-geth declares some Ethereum fields
// (terminalTotalDifficulty, depositContractAddress, ...) after its OP fork fields.
func TestChainConfigJSONKeyOrder(t *testing.T) {
	opKeys := map[string]bool{}
	cfgType := reflect.TypeFor[ChainConfig]()
	for i := range cfgType.NumField() {
		if name, _, _ := strings.Cut(cfgType.Field(i).Tag.Get("json"), ","); name != "chainId" {
			opKeys[name] = true
		}
	}
	for _, tc := range jsonTestCases {
		t.Run(tc.name, func(t *testing.T) {
			opGethJSON, err := json.Marshal(opGethTestConfig(t, tc.schedule))
			require.NoError(t, err)
			opGethKeys := objectKeys(t, opGethJSON)
			isOP := func(k string) bool { return opKeys[k] }
			want := slices.Concat(
				slices.DeleteFunc(slices.Clone(opGethKeys), isOP),
				slices.DeleteFunc(slices.Clone(opGethKeys), func(k string) bool { return !isOP(k) }),
			)

			got, err := json.Marshal(jsonTestConfig(t, tc.schedule))
			require.NoError(t, err)
			require.Equal(t, want, objectKeys(t, got))
			require.Equal(t, "chainId", want[0])
		})
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

// TestChainConfigJSONUnmarshalOpGeth decodes op-geth's encoding of the equivalent op-geth
// chain config, and the same with osakaTime absent, as op-chain-ops writes it for Karst.
func TestChainConfigJSONUnmarshalOpGeth(t *testing.T) {
	s := forkstest.Staggered()
	opGeth := opGethTestConfig(t, s)
	for name, osaka := range map[string]*uint64{"with osakaTime": s(forks.Karst), "without osakaTime": nil} {
		t.Run(name, func(t *testing.T) {
			opGeth.OsakaTime = osaka
			data, err := json.Marshal(opGeth)
			require.NoError(t, err)
			var got ChainConfig
			require.NoError(t, json.Unmarshal(data, &got))
			requireConfigEqual(t, jsonTestConfig(t, s), &got)
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
