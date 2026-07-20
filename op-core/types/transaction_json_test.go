package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

func toJSONMap(t *testing.T, data []byte) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &m))
	return m
}

// TestDepositTxJSONDifferential asserts the deposit JSON codec against op-geth's
// RPC transaction marshaling: op-geth JSON decodes into an identical canonical
// binary encoding, and our JSON is accepted by op-geth's decoder with an
// identical result. It will be removed in the final cutover, when the op-geth
// dependency is replaced with upstream go-ethereum.
func TestDepositTxJSONDifferential(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			gethJSON, err := tc.gethTx().MarshalJSON()
			require.NoError(t, err)

			// Reject-parity: where op-geth's decoder rejects its own encoding
			// (e.g. a nil Value marshals to null, which is invalid on decode),
			// ours must reject too.
			if err := new(gethtypes.Transaction).UnmarshalJSON(gethJSON); err != nil {
				require.Error(t, json.Unmarshal(gethJSON, new(optypes.DepositTx)))
				return
			}

			// op-geth JSON -> optypes -> binary equals op-geth binary
			var d optypes.DepositTx
			require.NoError(t, json.Unmarshal(gethJSON, &d))
			ours, err := d.MarshalBinary()
			require.NoError(t, err)
			theirs, err := tc.gethTx().MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, theirs, ours)

			// optypes JSON is object-identical to op-geth's
			ourJSON, err := json.Marshal(&d)
			require.NoError(t, err)
			require.Equal(t, toJSONMap(t, gethJSON), toJSONMap(t, ourJSON))

			// and op-geth's decoder accepts our JSON with the same result
			var back gethtypes.Transaction
			require.NoError(t, back.UnmarshalJSON(ourJSON))
			raw, err := back.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, theirs, raw)
		})
	}
}

// TestDepositTxJSONWithNonceDifferential exercises the post-Regolith RPC shape:
// op-geth's depositTxWithNonce marshaling carries a non-zero effective nonce,
// which must be accepted and ignored — it is not part of the canonical
// encoding. It will be removed in the final cutover, when the op-geth
// dependency is replaced with upstream go-ethereum.
func TestDepositTxJSONWithNonceDifferential(t *testing.T) {
	tc := depositTxTestCases()[0]

	// Route through op-geth's real depositTxWithNonce path: JSON with a nonce
	// decodes into the nonce-carrying inner type, whose re-marshal is the
	// post-Regolith RPC response shape.
	gethJSON, err := tc.gethTx().MarshalJSON()
	require.NoError(t, err)
	obj := toJSONMap(t, gethJSON)
	obj["nonce"] = json.RawMessage(`"0x7"`)
	withNonceJSON, err := json.Marshal(obj)
	require.NoError(t, err)

	var gethTx gethtypes.Transaction
	require.NoError(t, gethTx.UnmarshalJSON(withNonceJSON))
	nonce := gethTx.EffectiveNonce()
	require.NotNil(t, nonce)
	require.EqualValues(t, 7, *nonce) // proves the depositTxWithNonce path
	rpcJSON, err := gethTx.MarshalJSON()
	require.NoError(t, err)

	var d optypes.DepositTx
	require.NoError(t, json.Unmarshal(rpcJSON, &d))
	ours, err := d.MarshalBinary()
	require.NoError(t, err)
	canonical, err := tc.tx.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, canonical, ours)
}

func TestDepositTxUnmarshalJSONErrors(t *testing.T) {
	valid := func() map[string]json.RawMessage {
		gethJSON, err := depositTxTestCases()[0].gethTx().MarshalJSON()
		require.NoError(t, err)
		return toJSONMap(t, gethJSON)
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

func postExecGethTx(data []byte) *gethtypes.Transaction {
	return gethtypes.NewTx(&gethtypes.PostExecTx{Data: data})
}

// TestPostExecTxJSONDifferential mirrors TestDepositTxJSONDifferential for the
// 0x7D post-exec transaction. It will be removed in the final cutover, when
// the op-geth dependency is replaced with upstream go-ethereum.
func TestPostExecTxJSONDifferential(t *testing.T) {
	for _, data := range [][]byte{{0x01}, {0xc2, 0x80, 0x80}, make([]byte, 64)} {
		gethTx := postExecGethTx(data)
		gethJSON, err := gethTx.MarshalJSON()
		require.NoError(t, err)

		var p optypes.PostExecTx
		require.NoError(t, json.Unmarshal(gethJSON, &p))
		ours, err := p.MarshalBinary()
		require.NoError(t, err)
		theirs, err := gethTx.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, theirs, ours)

		require.Equal(t, gethTx.Hash(), p.Hash())

		ourJSON, err := json.Marshal(&p)
		require.NoError(t, err)
		require.Equal(t, toJSONMap(t, gethJSON), toJSONMap(t, ourJSON))

		var back gethtypes.Transaction
		require.NoError(t, back.UnmarshalJSON(ourJSON))
		raw, err := back.MarshalBinary()
		require.NoError(t, err)
		require.Equal(t, theirs, raw)
	}
}

func TestPostExecTxUnmarshalJSONErrors(t *testing.T) {
	valid := func() map[string]json.RawMessage {
		gethJSON, err := postExecGethTx([]byte{0x01}).MarshalJSON()
		require.NoError(t, err)
		return toJSONMap(t, gethJSON)
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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var p optypes.PostExecTx
			require.ErrorContains(t, json.Unmarshal(tc.input, &p), tc.errStr)
		})
	}
}
