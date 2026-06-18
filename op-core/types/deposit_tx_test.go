package types_test

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

type depositTxTestCase struct {
	name string
	tx   optypes.DepositTx
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
