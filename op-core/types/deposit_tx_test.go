package types_test

import (
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// depositTxTestCase pairs a deposit with its golden encodings, captured from op-geth:
// the canonical binary encoding, the transaction hash, and the RPC JSON encoding.
type depositTxTestCase struct {
	name   string
	tx     optypes.DepositTx
	binary string
	hash   string
	json   string
}

func depositTxTestCases() []depositTxTestCase {
	to := common.HexToAddress("0x4242424242424242424242424242424242424242")
	return []depositTxTestCase{
		{
			name: "all fields set",
			tx: optypes.DepositTx{
				SourceHash:          common.HexToHash("0x0001"),
				From:                common.HexToAddress("0x1111111111111111111111111111111111111111"),
				To:                  &to,
				Mint:                big.NewInt(1234),
				Value:               big.NewInt(5678),
				Gas:                 21000,
				IsSystemTransaction: false,
				Data:                []byte{0xde, 0xad, 0xbe, 0xef},
			},
			binary: "0x7ef85aa000000000000000000000000000000000000000000000000000000000000000019411111111111111111111111111111111111111119442424242424242424242424242424242424242428204d282162e8252088084deadbeef",
			hash:   "0xa549b7517be66c7890e09cf1f598b8faefbe83f997f59f96b9fc99b60e75bd8a",
			json:   `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0x5208","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x162e","input":"0xdeadbeef","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000001","from":"0x1111111111111111111111111111111111111111","mint":"0x4d2","isSystemTx":false,"hash":"0xa549b7517be66c7890e09cf1f598b8faefbe83f997f59f96b9fc99b60e75bd8a"}`,
		},
		{
			name: "contract creation (nil To)",
			tx: optypes.DepositTx{
				SourceHash: common.HexToHash("0x0002"),
				From:       common.HexToAddress("0x2222222222222222222222222222222222222222"),
				To:         nil,
				Mint:       big.NewInt(0),
				Value:      big.NewInt(0),
				Gas:        1_000_000,
				Data:       []byte{0x60, 0x80, 0x60, 0x40},
			},
			binary: "0x7ef843a00000000000000000000000000000000000000000000000000000000000000002942222222222222222222222222222222222222222808080830f4240808460806040",
			hash:   "0x83646ae3ccddb5bc7fe88b7eb68162925c625d7705ac9b5ec1888ad4c7c2963c",
			json:   `{"type":"0x7e","nonce":null,"to":null,"gas":"0xf4240","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x60806040","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000002","from":"0x2222222222222222222222222222222222222222","mint":"0x0","isSystemTx":false,"hash":"0x83646ae3ccddb5bc7fe88b7eb68162925c625d7705ac9b5ec1888ad4c7c2963c"}`,
		},
		{
			name: "nil Mint",
			tx: optypes.DepositTx{
				SourceHash: common.HexToHash("0x0003"),
				From:       common.HexToAddress("0x3333333333333333333333333333333333333333"),
				To:         &to,
				Mint:       nil,
				Value:      big.NewInt(7),
				Gas:        50_000,
				Data:       nil,
			},
			binary: "0x7ef852a00000000000000000000000000000000000000000000000000000000000000003943333333333333333333333333333333333333333944242424242424242424242424242424242424242800782c3508080",
			hash:   "0x610207e56454cd1a5d52bbe5378de8cc5c0ad1981b1476b797b9af24edaadd86",
			json:   `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0xc350","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x7","input":"0x","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000003","from":"0x3333333333333333333333333333333333333333","isSystemTx":false,"hash":"0x610207e56454cd1a5d52bbe5378de8cc5c0ad1981b1476b797b9af24edaadd86"}`,
		},
		{
			name: "zero values",
			tx: optypes.DepositTx{
				SourceHash: common.Hash{},
				From:       common.Address{},
				To:         nil,
				Mint:       nil,
				Value:      big.NewInt(0),
				Gas:        0,
				Data:       []byte{},
			},
			binary: "0x7ef83ca00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000808080808080",
			hash:   "0xa7ab2336fc12e6d4545aa50baf09bddb751a67f9ddd9fd90ef7474d2b15f23ac",
			json:   `{"type":"0x7e","nonce":null,"to":null,"gas":"0x0","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000000","from":"0x0000000000000000000000000000000000000000","isSystemTx":false,"hash":"0xa7ab2336fc12e6d4545aa50baf09bddb751a67f9ddd9fd90ef7474d2b15f23ac"}`,
		},
		{
			name: "nil Value",
			tx: optypes.DepositTx{
				SourceHash: common.HexToHash("0x0004"),
				From:       common.HexToAddress("0x4444444444444444444444444444444444444444"),
				To:         &to,
				Mint:       nil,
				Value:      nil,
				Gas:        21000,
			},
			binary: "0x7ef852a0000000000000000000000000000000000000000000000000000000000000000494444444444444444444444444444444444444444494424242424242424242424242424242424242424280808252088080",
			hash:   "0xfb3ed58bbce8bf29671862ab8717dddf189693640be65e4951942558ff8db142",
			json:   `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0x5208","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000004","from":"0x4444444444444444444444444444444444444444","isSystemTx":false,"hash":"0xfb3ed58bbce8bf29671862ab8717dddf189693640be65e4951942558ff8db142"}`,
		},
		{
			name: "system transaction",
			tx: optypes.DepositTx{
				SourceHash:          common.HexToHash("0x0005"),
				From:                common.HexToAddress("0x5555555555555555555555555555555555555555"),
				To:                  &to,
				Mint:                big.NewInt(0),
				Value:               big.NewInt(0),
				Gas:                 150_000_000,
				IsSystemTransaction: true,
				Data:                []byte{0x01},
			},
			binary: "0x7ef854a0000000000000000000000000000000000000000000000000000000000000000594555555555555555555555555555555555555555594424242424242424242424242424242424242424280808408f0d1800101",
			hash:   "0x446e30e119f710dd0f6deb8fe67038cd3e4351bf090308734ebef6eee719f5c8",
			json:   `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0x8f0d180","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x01","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000005","from":"0x5555555555555555555555555555555555555555","mint":"0x0","isSystemTx":true,"hash":"0x446e30e119f710dd0f6deb8fe67038cd3e4351bf090308734ebef6eee719f5c8"}`,
		},
		{
			name: "large values",
			tx: optypes.DepositTx{
				SourceHash: common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
				From:       common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff"),
				To:         &to,
				Mint:       new(big.Int).Lsh(big.NewInt(1), 255),
				Value:      new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
				Gas:        ^uint64(0),
				Data:       make([]byte, 512),
			},
			binary: "0x7ef9029aa0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff94ffffffffffffffffffffffffffffffffffffffff944242424242424242424242424242424242424242" +
				"a08000000000000000000000000000000000000000000000000000000000000000a0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff88ffffffffffffffff80" +
				"b90200" + strings.Repeat("00", 512),
			hash: "0x23ff7004ab9f4502097a525cc7421b979b7a42ebd61a695a46074d2a3deb9363",
			json: `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0xffffffffffffffff","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",` +
				`"input":"0x` + strings.Repeat("00", 512) + `","v":null,"r":null,"s":null,"sourceHash":"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","from":"0xffffffffffffffffffffffffffffffffffffffff",` +
				`"mint":"0x8000000000000000000000000000000000000000000000000000000000000000","isSystemTx":false,"hash":"0x23ff7004ab9f4502097a525cc7421b979b7a42ebd61a695a46074d2a3deb9363"}`,
		},
	}
}

func (tc depositTxTestCase) gethTx() *gethtypes.Transaction {
	return gethtypes.NewTx(&gethtypes.DepositTx{
		SourceHash:          tc.tx.SourceHash,
		From:                tc.tx.From,
		To:                  tc.tx.To,
		Mint:                tc.tx.Mint,
		Value:               tc.tx.Value,
		Gas:                 tc.tx.Gas,
		IsSystemTransaction: tc.tx.IsSystemTransaction,
		Data:                tc.tx.Data,
	})
}

// TestDepositTxMarshalBinaryDifferential asserts that DepositTx.MarshalBinary is
// byte-for-byte identical to op-geth's encoding of the same transaction.
// It will be removed in the final cutover, when the op-geth dependency is
// replaced with upstream go-ethereum.
func TestDepositTxMarshalBinaryDifferential(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			ours, err := tc.tx.MarshalBinary()
			require.NoError(t, err)

			theirs, err := tc.gethTx().MarshalBinary()
			require.NoError(t, err)

			require.Equal(t, theirs, ours)
		})
	}
}

// TestDepositHashParity asserts DepositTx.Hash is identical to op-geth's
// types.Transaction.Hash for the same deposit. It will be removed in the final
// cutover, when the op-geth dependency is replaced with upstream go-ethereum.
func TestDepositHashParity(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.gethTx().Hash(), tc.tx.Hash())
		})
	}
}

// TestBedrockSystemDepositHashGoldenVector pins the cross-client hash of a pre-Regolith
// L1-info deposit captured from op-geth. The system flag is part of the canonical encoding,
// so both the flag and this hash must stay aligned with op-alloy after op-geth is removed.
func TestBedrockSystemDepositHashGoldenVector(t *testing.T) {
	to := common.HexToAddress("0x4200000000000000000000000000000000000015")
	tx := &optypes.DepositTx{
		SourceHash:          common.HexToHash("0xfd3be1af69bee3e315a5dc5b773c64432230d76db68edae7a64193522118a377"),
		From:                common.HexToAddress("0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001"),
		To:                  &to,
		Mint:                big.NewInt(0),
		Value:               big.NewInt(0),
		Gas:                 150_000_000,
		IsSystemTransaction: true,
		Data: common.FromHex(
			"0x015d8eb90000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000006aad7005000000000000000000000000000000000000000000000000000000003b9aca001b7ca7f22fd5e435bbc1f175d4e7721cee397530e7ef753563b0b8772ec58d86000000000000000000000000000000000000000000000000000000000000000b0000000000000000000000003c44cdddb6a900fa2b585dd299e03d12fa4293bc000000000000000000000000000000000000000000000000000000000000083400000000000000000000000000000000000000000000000000000000000f4240",
		),
	}
	const opAlloyTxHash = "0x67c97e03f0759855b20459dd9b5f9ddebdc094f3669bf039011252bae68f5f24"

	require.Equal(t, opAlloyTxHash, tx.Hash().Hex())
}

// TestDepositTxGoldenVectors pins the binary encoding, the hash and the RPC JSON encoding
// of every deposit in depositTxTestCases, and that the JSON decodes to the same deposit.
func TestDepositTxGoldenVectors(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.tx.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(raw))
			require.Equal(t, tc.hash, tc.tx.Hash().Hex())

			data, err := json.Marshal(&tc.tx)
			require.NoError(t, err)
			require.JSONEq(t, tc.json, string(data))

			var decoded optypes.DepositTx
			require.NoError(t, json.Unmarshal([]byte(tc.json), &decoded))
			reencoded, err := decoded.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, tc.binary, hexutil.Encode(reencoded))
		})
	}
}

func TestUnmarshalDepositTxRoundTrip(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.tx.MarshalBinary()
			require.NoError(t, err)

			decoded, err := optypes.UnmarshalDepositTx(raw)
			require.NoError(t, err)

			reencoded, err := decoded.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, raw, reencoded)
		})
	}
}

func TestUnmarshalDepositTxErrors(t *testing.T) {
	_, err := optypes.UnmarshalDepositTx(nil)
	require.ErrorContains(t, err, "empty")

	_, err = optypes.UnmarshalDepositTx([]byte{0x02, 0xc0})
	require.ErrorContains(t, err, "type byte")

	_, err = optypes.UnmarshalDepositTx([]byte{optypes.DepositTxType, 0xff})
	require.ErrorContains(t, err, "invalid deposit tx payload")

	// trailing bytes after the RLP list must be rejected
	valid, err := depositTxTestCases()[0].tx.MarshalBinary()
	require.NoError(t, err)
	_, err = optypes.UnmarshalDepositTx(append(valid, 0x00))
	require.ErrorContains(t, err, "invalid deposit tx payload")
}

func TestTxTypeByteChecks(t *testing.T) {
	deposit := []byte{optypes.DepositTxType, 0xc0}
	postExec := []byte{optypes.PostExecTxType, 0xc0}
	dynamicFee := []byte{gethtypes.DynamicFeeTxType, 0xc0}

	require.True(t, optypes.IsDepositTxBytes(deposit))
	require.False(t, optypes.IsDepositTxBytes(postExec))
	require.False(t, optypes.IsDepositTxBytes(dynamicFee))
	require.False(t, optypes.IsDepositTxBytes(nil))

	require.True(t, optypes.IsPostExecTxBytes(postExec))
	require.False(t, optypes.IsPostExecTxBytes(deposit))
	require.False(t, optypes.IsPostExecTxBytes(dynamicFee))
	require.False(t, optypes.IsPostExecTxBytes(nil))
}

// TestDepositTxHelpers checks the free-function replacements for op-geth's
// Transaction methods against op-geth's methods themselves, on transactions
// decoded from raw bytes the way they arrive from the Engine API. Note that
// a nil Mint round-trips to zero through the wire encoding, in both
// implementations.
func TestDepositTxHelpers(t *testing.T) {
	for _, tc := range depositTxTestCases() {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := tc.gethTx().MarshalBinary()
			require.NoError(t, err)
			tx := new(gethtypes.Transaction)
			require.NoError(t, tx.UnmarshalBinary(raw))

			require.True(t, optypes.IsDepositTx(tx))
			require.Equal(t, tx.IsDepositTx(), optypes.IsDepositTx(tx))

			sourceHash, err := optypes.SourceHash(tx)
			require.NoError(t, err)
			require.Equal(t, tx.SourceHash(), sourceHash)
			require.Equal(t, tc.tx.SourceHash, sourceHash)

			mint, err := optypes.Mint(tx)
			require.NoError(t, err)
			require.Equal(t, tx.Mint(), mint)

			isSystemTx, err := optypes.IsSystemTx(tx)
			require.NoError(t, err)
			require.Equal(t, tx.IsSystemTx(), isSystemTx)
			require.Equal(t, tc.tx.IsSystemTransaction, isSystemTx)
		})
	}
}

// TestDepositTxHelpersJSONDecoded checks the helpers on deposits decoded from
// JSON-RPC responses, where op-geth uses a different inner type that carries
// the deposit nonce (depositTxWithNonce). The nonce must not leak into the
// re-encoded wire bytes the helpers decode.
func TestDepositTxHelpersJSONDecoded(t *testing.T) {
	tc := depositTxTestCases()[0]
	gethTx := tc.gethTx()

	rpcJSON, err := gethTx.MarshalJSON()
	require.NoError(t, err)
	var obj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(rpcJSON, &obj))
	obj["nonce"] = json.RawMessage(`"0x7"`)
	rpcJSON, err = json.Marshal(obj)
	require.NoError(t, err)

	tx := new(gethtypes.Transaction)
	require.NoError(t, tx.UnmarshalJSON(rpcJSON))
	nonce := tx.EffectiveNonce()
	require.NotNil(t, nonce)
	require.EqualValues(t, 7, *nonce) // proves we hit the depositTxWithNonce path

	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	canonical, err := tc.tx.MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, canonical, raw)

	sourceHash, err := optypes.SourceHash(tx)
	require.NoError(t, err)
	require.Equal(t, tx.SourceHash(), sourceHash)

	mint, err := optypes.Mint(tx)
	require.NoError(t, err)
	require.Equal(t, tx.Mint(), mint)

	isSystemTx, err := optypes.IsSystemTx(tx)
	require.NoError(t, err)
	require.Equal(t, tx.IsSystemTx(), isSystemTx)
}

func TestDepositTxHelpersNonDeposit(t *testing.T) {
	tx := gethtypes.NewTx(&gethtypes.LegacyTx{
		Nonce:    1,
		GasPrice: big.NewInt(1),
		Gas:      21000,
		To:       &common.Address{},
		Value:    big.NewInt(0),
	})

	require.False(t, optypes.IsDepositTx(tx))

	_, err := optypes.SourceHash(tx)
	require.ErrorContains(t, err, "not a deposit transaction")
	_, err = optypes.Mint(tx)
	require.ErrorContains(t, err, "not a deposit transaction")
	_, err = optypes.IsSystemTx(tx)
	require.ErrorContains(t, err, "not a deposit transaction")
}
