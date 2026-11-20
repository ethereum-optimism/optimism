package genesis

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/eip1559"
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

// fixture returns a genesis with the fork schedule s, like the ones op-chain-ops builds. It
// has the blob gas fields only if Ecotone, and with it Cancun, is active at genesis.
func fixture(s forkstest.Schedule) *Genesis {
	g := &Genesis{
		Config:     opChainConfig(s),
		Nonce:      7,
		Timestamp:  1234,
		ExtraData:  eip1559.EncodeJovianExtraData(250, 6, 0),
		GasLimit:   30_000_000,
		Difficulty: big.NewInt(0),
		Mixhash:    common.Hash{0xaa},
		Coinbase:   predeploys.SequencerFeeVaultAddr,
		Alloc:      fixtureAlloc(),
		BaseFee:    big.NewInt(1_000_000_000),
	}
	if g.Config.IsEcotone(g.Timestamp) {
		g.ExcessBlobGas = ptr.New[uint64](0)
		g.BlobGasUsed = ptr.New[uint64](0)
	}
	return g
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

func TestRoundTrip(t *testing.T) {
	g := fixture(forkstest.Staggered())
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
	g := fixture(forkstest.Staggered())
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
	g := fixture(forkstest.Staggered())
	g.Config = nil
	data, err := json.Marshal(g)
	require.NoError(t, err)
	require.Nil(t, parseJSON(t, data)["config"])

	var got Genesis
	require.NoError(t, json.Unmarshal(data, &got))
	require.Nil(t, got.Config)
	require.Nil(t, got.gethGenesis().Config)
}

func TestUnmarshalMinimal(t *testing.T) {
	var got Genesis
	require.NoError(t, json.Unmarshal([]byte(`{"gasLimit":"0x1c9c380","difficulty":"0x0","alloc":{}}`), &got))
	requireGenesisEqual(t, &Genesis{GasLimit: 30_000_000, Difficulty: big.NewInt(0), Alloc: types.GenesisAlloc{}}, &got)
}

// TestUnmarshalConfigKeys checks how decoding treats "config" keys that Genesis does not
// store: unknown keys are ignored, and a derived key must match its derivation.
func TestUnmarshalConfigKeys(t *testing.T) {
	g := fixture(forkstest.Staggered())
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
	g := fixture(forkstest.Staggered())
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

// setBlockFields sets the block fields that are zero or absent in a genesis.json
// op-chain-ops writes.
func setBlockFields(g *Genesis) {
	g.Number = 3
	g.GasUsed = 21_000
	g.ParentHash = common.Hash{0xbb}
	g.ExcessBlobGas = ptr.New[uint64](0)
	g.BlobGasUsed = ptr.New[uint64](0)
	g.SlotNumber = ptr.New[uint64](9)
}

// fixtureAllocJSON is the encoding of fixtureAlloc.
const fixtureAllocJSON = `"alloc":{"00000000000000000000000000000000000beef0":{"balance":"0x100000000000000000000000000000000000000000000000000"},` +
	`"4200000000000000000000000000000000000016":{"code":"0x60006000f3","storage":{"0x0100000000000000000000000000000000000000000000000000000000000000":"0x0200000000000000000000000000000000000000000000000000000000000000","0x0300000000000000000000000000000000000000000000000000000000000000":"0x0400000000000000000000000000000000000000000000000000000000000000"},"balance":"0x0","nonce":"0x1"}}`

// TestMarshalGoldenVector pins the encoding of the staggered fixture with the block fields
// that are zero in a genesis.json op-chain-ops writes, as op-geth encodes it.
func TestMarshalGoldenVector(t *testing.T) {
	const want = `{"config":{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
		`"shanghaiTime":2000,"cancunTime":3000,"pragueTime":7000,"osakaTime":9000,"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000",` +
		`"bedrockBlock":0,"regolithTime":1000,"canyonTime":2000,"ecotoneTime":3000,"fjordTime":4000,"graniteTime":5000,"holoceneTime":6000,"isthmusTime":7000,"jovianTime":8000,"karstTime":9000,"lagoonTime":10000,` +
		`"optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}},` +
		`"nonce":"0x7","timestamp":"0x4d2","extraData":"0x01000000fa000000060000000000000000","gasLimit":"0x1c9c380","difficulty":"0x0","mixHash":"0xaa00000000000000000000000000000000000000000000000000000000000000","coinbase":"0x4200000000000000000000000000000000000011",` +
		fixtureAllocJSON + `,` +
		`"number":"0x3","gasUsed":"0x5208","parentHash":"0xbb00000000000000000000000000000000000000000000000000000000000000","baseFeePerGas":"0x3b9aca00","excessBlobGas":"0x0","blobGasUsed":"0x0","slotNumber":9}`

	g := fixture(forkstest.Staggered())
	setBlockFields(g)
	data, err := json.Marshal(g)
	require.NoError(t, err)
	require.JSONEq(t, want, string(data))
}

// opGethGenesisJSON is the genesis.json that op-geth's core.Genesis writes for the
// staggered fixture, which has the OP Stack config keys among the Ethereum keys, and
// opGethGenesisJSONWithoutOsaka the same without osakaTime, as op-chain-ops writes it for
// Karst.
const (
	opGethGenesisJSONHead = `{"config":{"chainId":901,"homesteadBlock":0,"eip150Block":0,"eip155Block":0,"eip158Block":0,"byzantiumBlock":0,"constantinopleBlock":0,"petersburgBlock":0,"istanbulBlock":0,"muirGlacierBlock":0,"berlinBlock":0,"londonBlock":0,"arrowGlacierBlock":0,"grayGlacierBlock":0,"mergeNetsplitBlock":0,` +
		`"shanghaiTime":2000,"cancunTime":3000,"pragueTime":7000,`
	opGethGenesisJSONTail = `"bedrockBlock":0,"regolithTime":1000,"canyonTime":2000,"ecotoneTime":3000,"fjordTime":4000,"graniteTime":5000,"holoceneTime":6000,"isthmusTime":7000,"jovianTime":8000,"karstTime":9000,"lagoonTime":10000,` +
		`"terminalTotalDifficulty":0,"depositContractAddress":"0x0000000000000000000000000000000000000000","optimism":{"eip1559Elasticity":6,"eip1559Denominator":50,"eip1559DenominatorCanyon":250}},` +
		`"nonce":"0x7","timestamp":"0x4d2","extraData":"0x01000000fa000000060000000000000000","gasLimit":"0x1c9c380","difficulty":"0x0","mixHash":"0xaa00000000000000000000000000000000000000000000000000000000000000","coinbase":"0x4200000000000000000000000000000000000011",` +
		fixtureAllocJSON + `,` +
		`"number":"0x0","gasUsed":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","baseFeePerGas":"0x3b9aca00","excessBlobGas":null,"blobGasUsed":null,"slotNumber":null}`

	opGethGenesisJSON             = opGethGenesisJSONHead + `"osakaTime":9000,` + opGethGenesisJSONTail
	opGethGenesisJSONWithoutOsaka = opGethGenesisJSONHead + opGethGenesisJSONTail
)

// TestUnmarshalOpGethGoldenVector decodes the genesis.json that op-geth's core.Genesis
// writes, and the same without osakaTime.
func TestUnmarshalOpGethGoldenVector(t *testing.T) {
	for name, data := range map[string]string{
		"with osakaTime":    opGethGenesisJSON,
		"without osakaTime": opGethGenesisJSONWithoutOsaka,
	} {
		t.Run(name, func(t *testing.T) {
			var got Genesis
			require.NoError(t, json.Unmarshal([]byte(data), &got))
			want := fixture(forkstest.Staggered())
			requireGenesisEqual(t, want, &got)
		})
	}
}
