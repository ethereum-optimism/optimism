package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// depositTxTestCase pairs a deposit with its golden encodings: the canonical binary
// encoding, the transaction hash, and the RPC JSON encoding. They are those that op-geth
// computes, and op-alloy computes the same binary encoding and hash.
type depositTxTestCase struct {
	name   string
	tx     optypes.DepositTx
	binary hexutil.Bytes
	hash   common.Hash
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
			binary: hexutil.MustDecode("0x7ef85aa000000000000000000000000000000000000000000000000000000000000000019411111111111111111111111111111111111111119442424242424242424242424242424242424242428204d282162e8252088084deadbeef"),
			hash:   common.HexToHash("0xa549b7517be66c7890e09cf1f598b8faefbe83f997f59f96b9fc99b60e75bd8a"),
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
			binary: hexutil.MustDecode("0x7ef843a00000000000000000000000000000000000000000000000000000000000000002942222222222222222222222222222222222222222808080830f4240808460806040"),
			hash:   common.HexToHash("0x83646ae3ccddb5bc7fe88b7eb68162925c625d7705ac9b5ec1888ad4c7c2963c"),
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
			binary: hexutil.MustDecode("0x7ef852a00000000000000000000000000000000000000000000000000000000000000003943333333333333333333333333333333333333333944242424242424242424242424242424242424242800782c3508080"),
			hash:   common.HexToHash("0x610207e56454cd1a5d52bbe5378de8cc5c0ad1981b1476b797b9af24edaadd86"),
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
			binary: hexutil.MustDecode("0x7ef83ca00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000808080808080"),
			hash:   common.HexToHash("0xa7ab2336fc12e6d4545aa50baf09bddb751a67f9ddd9fd90ef7474d2b15f23ac"),
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
			binary: hexutil.MustDecode("0x7ef852a0000000000000000000000000000000000000000000000000000000000000000494444444444444444444444444444444444444444494424242424242424242424242424242424242424280808252088080"),
			hash:   common.HexToHash("0xfb3ed58bbce8bf29671862ab8717dddf189693640be65e4951942558ff8db142"),
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
			binary: hexutil.MustDecode("0x7ef854a0000000000000000000000000000000000000000000000000000000000000000594555555555555555555555555555555555555555594424242424242424242424242424242424242424280808408f0d1800101"),
			hash:   common.HexToHash("0x446e30e119f710dd0f6deb8fe67038cd3e4351bf090308734ebef6eee719f5c8"),
			json:   `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0x8f0d180","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0x0","input":"0x01","v":null,"r":null,"s":null,"sourceHash":"0x0000000000000000000000000000000000000000000000000000000000000005","from":"0x5555555555555555555555555555555555555555","mint":"0x0","isSystemTx":true,"hash":"0x446e30e119f710dd0f6deb8fe67038cd3e4351bf090308734ebef6eee719f5c8"}`,
		},
		{
			name: "large values",
			tx: optypes.DepositTx{
				SourceHash: common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
				From:       common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff"),
				To:         &to,
				Mint:       new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1)),
				Value:      new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)),
				Gas:        ^uint64(0),
				Data:       make([]byte, 512),
			},
			binary: hexutil.MustDecode("0x7ef9028aa0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff94ffffffffffffffffffffffffffffffffffffffff944242424242424242424242424242424242424242" +
				"90ffffffffffffffffffffffffffffffffa0ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff88ffffffffffffffff80" +
				"b90200" + zeroBytesHex(512)),
			hash: common.HexToHash("0x37dc99204b60f5f7cf42429fcdb586d3b59fa48b80519112afef0c49841ed440"),
			json: `{"type":"0x7e","nonce":null,"to":"0x4242424242424242424242424242424242424242","gas":"0xffffffffffffffff","gasPrice":null,"maxPriorityFeePerGas":null,"maxFeePerGas":null,"value":"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",` +
				`"input":"0x` + zeroBytesHex(512) + `","v":null,"r":null,"s":null,"sourceHash":"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff","from":"0xffffffffffffffffffffffffffffffffffffffff",` +
				`"mint":"0xffffffffffffffffffffffffffffffff","isSystemTx":false,"hash":"0x37dc99204b60f5f7cf42429fcdb586d3b59fa48b80519112afef0c49841ed440"}`,
		},
	}
}

// TestBedrockSystemDepositHashGoldenVector pins the cross-client hash of a pre-Regolith
// L1-info deposit captured from op-geth. The system flag is part of the canonical encoding,
// so both the flag and this hash must stay aligned with op-alloy.
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
