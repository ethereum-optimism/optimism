package genesis

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	gethparams "github.com/ethereum/go-ethereum/params"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	forkstest "github.com/ethereum-optimism/optimism/op-core/forks/test"
	"github.com/ethereum-optimism/optimism/op-core/params"
	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

func opChainConfig(s forkstest.Schedule) *params.ChainConfig {
	cfg := &params.ChainConfig{
		ChainID: big.NewInt(901),
		Optimism: &params.OptimismConfig{
			EIP1559Elasticity:        6,
			EIP1559Denominator:       50,
			EIP1559DenominatorCanyon: ptr.New[uint64](250),
		},
		BedrockBlock: big.NewInt(0),
	}
	for _, fork := range forkstest.TimestampELForks {
		cfg.SetActivationTime(fork, s(fork))
	}
	return cfg
}

// opGethChainConfig is the op-geth chain config for the fork schedule s, built from
// op-geth's own OP Stack test config independently of params.ChainConfig.GethChainConfig.
func opGethChainConfig(t *testing.T, s forkstest.Schedule) *gethparams.ChainConfig {
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

func fixtureAlloc() types.GenesisAlloc {
	return types.GenesisAlloc{
		predeploys.L2ToL1MessagePasserAddr: {
			Code:    []byte{0x60, 0x00, 0x60, 0x00, 0xf3},
			Storage: map[common.Hash]common.Hash{{0x01}: {0x02}, {0x03}: {0x04}},
			Balance: big.NewInt(0),
			Nonce:   1,
		},
		common.HexToAddress("0x00000000000000000000000000000000000beef0"): {
			Balance: new(big.Int).Lsh(big.NewInt(1), 200),
		},
	}
}

// fixture returns the same genesis as a Genesis and as the op-geth core.Genesis that
// op-chain-ops builds for it.
func fixture(t *testing.T, s forkstest.Schedule) (*Genesis, *core.Genesis) {
	// Built first so that a fork op-geth's config lacks fails the test by name, before
	// opChainConfig's SetActivationTime panics on a fork ChainConfig lacks.
	gethConfig := opGethChainConfig(t, s)
	g := &Genesis{
		Config:        opChainConfig(s),
		Nonce:         7,
		Timestamp:     1234,
		ExtraData:     eip1559.EncodeJovianExtraData(250, 6, 0),
		GasLimit:      30_000_000,
		Difficulty:    big.NewInt(0),
		Mixhash:       common.Hash{0xaa},
		Coinbase:      predeploys.SequencerFeeVaultAddr,
		Alloc:         fixtureAlloc(),
		BaseFee:       big.NewInt(1_000_000_000),
		ExcessBlobGas: ptr.New[uint64](0),
		BlobGasUsed:   ptr.New[uint64](0),
	}
	gg := &core.Genesis{
		Config:        gethConfig,
		Nonce:         g.Nonce,
		Timestamp:     g.Timestamp,
		ExtraData:     g.ExtraData,
		GasLimit:      g.GasLimit,
		Difficulty:    g.Difficulty,
		Mixhash:       g.Mixhash,
		Coinbase:      g.Coinbase,
		Alloc:         fixtureAlloc(),
		BaseFee:       g.BaseFee,
		ExcessBlobGas: g.ExcessBlobGas,
		BlobGasUsed:   g.BlobGasUsed,
	}
	return g, gg
}

// requireGenesisEqual compares big.Int fields by value: a decoded zero and big.NewInt(0)
// differ in their internal representation.
func requireGenesisEqual(t *testing.T, want, got *Genesis) {
	t.Helper()
	bigEq := cmp.Comparer(func(a, b *big.Int) bool {
		if a == nil || b == nil {
			return a == b
		}
		return a.Cmp(b) == 0
	})
	require.Empty(t, cmp.Diff(want, got, bigEq))
}

func parseJSON(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var obj map[string]any
	require.NoError(t, json.Unmarshal(data, &obj))
	return obj
}

// TestMarshalMatchesOpGeth pins the encoding to the genesis.json that op-geth's
// core.Genesis writes for the equivalent op-geth config, built from op-geth's
// OptimismTestConfig. It is a pre-cutover check: it depends on op-geth's config type and
// test config, and goes away with op-geth.
func TestMarshalMatchesOpGeth(t *testing.T) {
	for _, tc := range []struct {
		name     string
		schedule forkstest.Schedule
	}{
		{name: "pre-Karst", schedule: forkstest.AtGenesisThrough(forks.Jovian)},
		{name: "Karst", schedule: forkstest.Override(forkstest.AtGenesisThrough(forks.Jovian), forks.Karst, ptr.New[uint64](9000))},
		{name: "staggered", schedule: forkstest.Staggered()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g, gg := fixture(t, tc.schedule)
			g.Number, gg.Number = 3, 3
			g.GasUsed, gg.GasUsed = 21_000, 21_000
			g.ParentHash, gg.ParentHash = common.Hash{0xbb}, common.Hash{0xbb}
			g.SlotNumber, gg.SlotNumber = ptr.New[uint64](9), ptr.New[uint64](9)
			opGethJSON, err := json.Marshal(gg)
			require.NoError(t, err)

			got, err := json.Marshal(g)
			require.NoError(t, err)
			obj := parseJSON(t, got)
			require.Equal(t, parseJSON(t, opGethJSON), obj)

			config := obj["config"].(map[string]any)
			if karst := tc.schedule(forks.Karst); karst != nil {
				require.Equal(t, float64(*karst), config["osakaTime"], "Osaka activates with Karst")
			} else {
				require.NotContains(t, config, "osakaTime")
			}
		})
	}
}

// TestUnmarshalOpGeth decodes the genesis.json that op-geth's core.Genesis encodes, and
// the same without osakaTime, as op-chain-ops writes it for Karst.
func TestUnmarshalOpGeth(t *testing.T) {
	for name, withOsaka := range map[string]bool{"with osakaTime": true, "without osakaTime": false} {
		t.Run(name, func(t *testing.T) {
			want, gg := fixture(t, forkstest.Staggered())
			if !withOsaka {
				gg.Config.OsakaTime = nil
			}
			data, err := json.Marshal(gg)
			require.NoError(t, err)

			var got Genesis
			require.NoError(t, json.Unmarshal(data, &got))
			requireGenesisEqual(t, want, &got)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	g, _ := fixture(t, forkstest.Staggered())
	g.Number = 3
	g.GasUsed = 21_000
	g.ParentHash = common.Hash{0xbb}
	g.SlotNumber = ptr.New[uint64](9)

	data, err := json.Marshal(g)
	require.NoError(t, err)
	var got Genesis
	require.NoError(t, json.Unmarshal(data, &got))
	requireGenesisEqual(t, g, &got)

	again, err := json.Marshal(&got)
	require.NoError(t, err)
	require.JSONEq(t, string(data), string(again))
}

func TestEncoding(t *testing.T) {
	g, _ := fixture(t, forkstest.Staggered())
	data, err := json.Marshal(g)
	require.NoError(t, err)
	obj := parseJSON(t, data)

	require.Equal(t, "0x7", obj["nonce"])
	require.Equal(t, "0x4d2", obj["timestamp"])
	require.Equal(t, "0x1c9c380", obj["gasLimit"])
	require.Equal(t, "0x0", obj["difficulty"])
	require.Equal(t, "0x3b9aca00", obj["baseFeePerGas"])
	require.Equal(t, common.Hash{0xaa}.Hex(), obj["mixHash"])
	require.Contains(t, obj["alloc"], "00000000000000000000000000000000000beef0", "alloc keys carry no 0x prefix")

	config := obj["config"].(map[string]any)
	require.Equal(t, float64(901), config["chainId"])
	require.Equal(t, float64(2000), config["canyonTime"])
	require.Equal(t, float64(2000), config["shanghaiTime"])
	require.Equal(t, float64(7000), config["pragueTime"])
	require.Equal(t, map[string]any{
		"eip1559Elasticity":        float64(6),
		"eip1559Denominator":       float64(50),
		"eip1559DenominatorCanyon": float64(250),
	}, config["optimism"])
}

func TestNilConfig(t *testing.T) {
	g, _ := fixture(t, forkstest.Staggered())
	g.Config = nil
	data, err := json.Marshal(g)
	require.NoError(t, err)
	require.Nil(t, parseJSON(t, data)["config"])

	var got Genesis
	require.NoError(t, json.Unmarshal(data, &got))
	require.Nil(t, got.Config)
	require.Nil(t, got.GethGenesis().Config)
}

func TestUnmarshalMinimal(t *testing.T) {
	var got Genesis
	require.NoError(t, json.Unmarshal([]byte(`{"gasLimit":"0x1c9c380","difficulty":"0x0","alloc":{}}`), &got))
	requireGenesisEqual(t, &Genesis{GasLimit: 30_000_000, Difficulty: big.NewInt(0), Alloc: types.GenesisAlloc{}}, &got)
}

// TestUnmarshalConfigKeys checks how decoding treats "config" keys that Genesis does not
// store: unknown keys are ignored, and a derived key must match its derivation.
func TestUnmarshalConfigKeys(t *testing.T) {
	g, _ := fixture(t, forkstest.Staggered())
	data, err := json.Marshal(g)
	require.NoError(t, err)

	decodeWithConfig := func(edit func(config map[string]any)) error {
		obj := parseJSON(t, data)
		edit(obj["config"].(map[string]any))
		edited, err := json.Marshal(obj)
		require.NoError(t, err)
		var got Genesis
		return json.Unmarshal(edited, &got)
	}

	require.NoError(t, decodeWithConfig(func(c map[string]any) { c["terminalTotalDifficultyPassed"] = true }),
		"unknown keys are ignored")
	require.NoError(t, decodeWithConfig(func(c map[string]any) { delete(c, "shanghaiTime") }),
		"derived keys may be absent")
	require.ErrorContains(t, decodeWithConfig(func(c map[string]any) { c["shanghaiTime"] = 1 }), `"shanghaiTime"`)
	require.ErrorContains(t, decodeWithConfig(func(c map[string]any) { c["londonBlock"] = 5 }), `"londonBlock"`)
}

func TestUnmarshalRejects(t *testing.T) {
	g, _ := fixture(t, forkstest.Staggered())
	data, err := json.Marshal(g)
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		edit   func(map[string]any)
		errMsg string
	}{
		{name: "missing gasLimit", edit: func(o map[string]any) { delete(o, "gasLimit") }, errMsg: "'gasLimit'"},
		{name: "missing difficulty", edit: func(o map[string]any) { delete(o, "difficulty") }, errMsg: "'difficulty'"},
		{name: "missing alloc", edit: func(o map[string]any) { delete(o, "alloc") }, errMsg: "'alloc'"},
		{name: "stateHash", edit: func(o map[string]any) { o["stateHash"] = common.Hash{0x01}.Hex() }, errMsg: "stateHash"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := parseJSON(t, data)
			tc.edit(obj)
			edited, err := json.Marshal(obj)
			require.NoError(t, err)
			var got Genesis
			require.ErrorContains(t, json.Unmarshal(edited, &got), tc.errMsg)
		})
	}
}

// TestGethGenesisToBlock checks that the genesis block computed through GethGenesis is
// the one op-geth computes from the genesis that op-chain-ops builds, including the
// Isthmus withdrawals root (the L2ToL1MessagePasser storage root), for a staggered
// schedule and for every timestamp fork as the latest at genesis.
func TestGethGenesisToBlock(t *testing.T) {
	schedules := map[string]forkstest.Schedule{"staggered": forkstest.Staggered()}
	for _, fork := range forkstest.TimestampELForks {
		schedules["through "+string(fork)] = forkstest.AtGenesisThrough(fork)
	}
	for name, s := range schedules {
		t.Run(name, func(t *testing.T) {
			g, gg := fixture(t, s)
			want := gg.ToBlock()
			got := g.GethGenesis().ToBlock()
			require.Equal(t, want.Header(), got.Header())
			require.Equal(t, want.Hash(), got.Hash())
			if g.Config.IsIsthmus(g.Timestamp) {
				require.NotEqual(t, types.EmptyWithdrawalsHash, *got.Header().WithdrawalsHash,
					"Isthmus withdrawals root is the message passer's storage root")
			}
		})
	}
}
