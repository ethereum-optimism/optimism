package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

func toJSONMap(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// TestDepositTxJSONNilValueRoundTrip pins that a nil Value — valid for the
// binary codec — marshals as zero and round-trips through this type's own
// JSON codec (a null value would be rejected by UnmarshalJSON).
func TestDepositTxJSONNilValueRoundTrip(t *testing.T) {
	d := &optypes.DepositTx{
		SourceHash: common.HexToHash("0x06"),
		From:       common.HexToAddress("0x6666666666666666666666666666666666666666"),
		Value:      nil,
		Gas:        21000,
	}
	data, err := json.Marshal(d)
	require.NoError(t, err)
	require.Contains(t, string(data), `"value":"0x0"`)

	var decoded optypes.DepositTx
	require.NoError(t, json.Unmarshal(data, &decoded))
	require.Zero(t, decoded.Value.Sign())

	// Binary encodings of the nil and decoded-zero forms agree.
	origBin, err := d.MarshalBinary()
	require.NoError(t, err)
	decodedBin, err := decoded.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, origBin, decodedBin)
}

func TestDepositTxUnmarshalJSONErrors(t *testing.T) {
	valid := func() map[string]json.RawMessage {
		return toJSONMap(t, []byte(depositTxTestCases()[0].json))
	}
	mutate := func(mut func(map[string]json.RawMessage)) []byte {
		obj := valid()
		mut(obj)
		data, err := json.Marshal(obj)
		require.NoError(t, err)
		return data
	}
	cases := []struct {
		name   string
		input  []byte
		errStr string
	}{
		{"wrong type", mutate(func(o map[string]json.RawMessage) { o["type"] = json.RawMessage(`"0x2"`) }), "not deposit"},
		{"truncated type match", mutate(func(o map[string]json.RawMessage) { o["type"] = json.RawMessage(`"0x17e"`) }), "not deposit"},
		{"missing type", mutate(func(o map[string]json.RawMessage) { delete(o, "type") }), "not deposit"},
		{"access list present", mutate(func(o map[string]json.RawMessage) { o["accessList"] = json.RawMessage(`[]`) }), "unexpected field"},
		{"maxFeePerGas present", mutate(func(o map[string]json.RawMessage) { o["maxFeePerGas"] = json.RawMessage(`"0x1"`) }), "unexpected field"},
		{"non-zero gasPrice", mutate(func(o map[string]json.RawMessage) { o["gasPrice"] = json.RawMessage(`"0x1"`) }), "GasPrice must be 0"},
		{"non-zero signature", mutate(func(o map[string]json.RawMessage) { o["v"] = json.RawMessage(`"0x1"`) }), "signature must be 0"},
		{"missing gas", mutate(func(o map[string]json.RawMessage) { delete(o, "gas") }), "'gas'"},
		{"missing value", mutate(func(o map[string]json.RawMessage) { o["value"] = json.RawMessage(`null`) }), "'value'"},
		{"missing input", mutate(func(o map[string]json.RawMessage) { o["input"] = json.RawMessage(`null`) }), "'input'"},
		{"missing from", mutate(func(o map[string]json.RawMessage) { delete(o, "from") }), "'from'"},
		{"missing sourceHash", mutate(func(o map[string]json.RawMessage) { delete(o, "sourceHash") }), "'sourceHash'"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var d optypes.DepositTx
			require.ErrorContains(t, json.Unmarshal(tc.input, &d), tc.errStr)
		})
	}
}

func TestPostExecTxUnmarshalJSONErrors(t *testing.T) {
	valid := func() map[string]json.RawMessage {
		data, err := json.Marshal(&optypes.PostExecTx{Data: []byte{0x01}})
		require.NoError(t, err)
		return toJSONMap(t, data)
	}
	mutate := func(mut func(map[string]json.RawMessage)) []byte {
		obj := valid()
		mut(obj)
		data, err := json.Marshal(obj)
		require.NoError(t, err)
		return data
	}
	cases := []struct {
		name   string
		input  []byte
		errStr string
	}{
		{"wrong type", mutate(func(o map[string]json.RawMessage) { o["type"] = json.RawMessage(`"0x7e"`) }), "not post-exec"},
		{"truncated type match", mutate(func(o map[string]json.RawMessage) { o["type"] = json.RawMessage(`"0x17d"`) }), "not post-exec"},
		{"to present", mutate(func(o map[string]json.RawMessage) {
			o["to"] = json.RawMessage(`"0x4242424242424242424242424242424242424242"`)
		}), "unexpected field"},
		{"mint present", mutate(func(o map[string]json.RawMessage) { o["mint"] = json.RawMessage(`"0x1"`) }), "unexpected field"},
		{"non-zero from", mutate(func(o map[string]json.RawMessage) {
			o["from"] = json.RawMessage(`"0x4242424242424242424242424242424242424242"`)
		}), "from must be zero"},
		{"non-zero nonce", mutate(func(o map[string]json.RawMessage) { o["nonce"] = json.RawMessage(`"0x1"`) }), "nonce must be 0"},
		{"non-zero value", mutate(func(o map[string]json.RawMessage) { o["value"] = json.RawMessage(`"0x1"`) }), "value must be 0"},
		{"non-zero gas", mutate(func(o map[string]json.RawMessage) { o["gas"] = json.RawMessage(`"0x1"`) }), "gas must be 0"},
		{"non-zero signature", mutate(func(o map[string]json.RawMessage) { o["r"] = json.RawMessage(`"0x1"`) }), "signature must be 0"},
		{"missing input", mutate(func(o map[string]json.RawMessage) { o["input"] = json.RawMessage(`null`) }), "'input'"},
		{"empty input", mutate(func(o map[string]json.RawMessage) { o["input"] = json.RawMessage(`"0x"`) }), "input must not be empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p optypes.PostExecTx
			require.ErrorContains(t, json.Unmarshal(tc.input, &p), tc.errStr)
		})
	}
}
