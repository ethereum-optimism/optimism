package types_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// consensusReceiptCases covers every receipts-root encoding arm: legacy,
// standard typed, post-exec (0x7D), and the three deposit shapes — plain
// (pre-Regolith), nonce-only (post-Regolith pre-Canyon, whose hash preimage
// deliberately EXCLUDES the nonce), and nonce+version (post-Canyon).
func consensusReceiptCases() []*optypes.Receipt {
	mk := func(typ uint8, status uint64, mut func(*optypes.Receipt)) *optypes.Receipt {
		r := &optypes.Receipt{Receipt: gethtypes.Receipt{
			Type:              typ,
			Status:            status,
			CumulativeGasUsed: 42_000,
			Logs: []*gethtypes.Log{
				{
					Address: common.HexToAddress("0x2222222222222222222222222222222222222222"),
					Topics:  []common.Hash{common.HexToHash("0x02"), common.HexToHash("0x03")},
					Data:    []byte{0x01, 0x02},
				},
			},
		}}
		r.Bloom = gethtypes.CreateBloom(&r.Receipt)
		if mut != nil {
			mut(r)
		}
		return r
	}
	return []*optypes.Receipt{
		mk(gethtypes.LegacyTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.DynamicFeeTxType, gethtypes.ReceiptStatusFailed, nil),
		mk(gethtypes.DynamicFeeTxType, gethtypes.ReceiptStatusSuccessful, func(r *optypes.Receipt) {
			r.Logs = []*gethtypes.Log{}
			r.Bloom = gethtypes.CreateBloom(&r.Receipt)
		}),
		mk(gethtypes.BlobTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.SetCodeTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(optypes.PostExecTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(optypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(optypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, func(r *optypes.Receipt) {
			r.DepositNonce = uint64Ptr(7) // pre-Canyon: nonce set, no version
		}),
		mk(optypes.DepositTxType, gethtypes.ReceiptStatusFailed, func(r *optypes.Receipt) {
			r.DepositNonce = uint64Ptr(8)
			r.DepositReceiptVersion = uint64Ptr(optypes.CanyonDepositReceiptVersion)
		}),
	}
}

// TestReceiptsEncodeIndexDepositArms pins the deposit hash-preimage rules:
//   - pre-Canyon (nonce set, version nil): the preimage EXCLUDES the nonce, so
//     it must equal the encoding of the same receipt without a nonce;
//   - post-Canyon (version set): the preimage includes nonce+version, so it
//     must differ from the pre-Canyon encoding.
func TestReceiptsEncodeIndexDepositArms(t *testing.T) {
	var base *optypes.Receipt
	for _, r := range consensusReceiptCases() {
		if r.Type == optypes.DepositTxType && r.DepositNonce != nil && r.DepositReceiptVersion == nil {
			base = r // pre-Canyon shape: nonce set, version nil
		}
	}
	require.NotNil(t, base)

	encode := func(nonce, version *uint64) []byte {
		r := &optypes.Receipt{Receipt: base.Receipt}
		r.DepositNonce = nonce
		r.DepositReceiptVersion = version
		var buf bytes.Buffer
		optypes.Receipts{r}.EncodeIndex(0, &buf)
		return buf.Bytes()
	}

	preCanyon := encode(base.DepositNonce, nil)
	noNonce := encode(nil, nil)
	postCanyon := encode(base.DepositNonce, uint64Ptr(optypes.CanyonDepositReceiptVersion))

	require.Equal(t, noNonce, preCanyon, "pre-Canyon deposit receipt hash preimage must exclude the nonce")
	require.NotEqual(t, preCanyon, postCanyon, "post-Canyon deposit receipt hash preimage must include nonce+version")
}

// TestReceiptBinaryRoundTrip pins that the wrapper's binary codec round-trips.
// That the encoding carries the outer deposit fields is pinned by
// TestReceiptGoldenVectors.
func TestReceiptBinaryRoundTrip(t *testing.T) {
	for i, want := range consensusReceiptCases() {
		raw, err := want.MarshalBinary()
		require.NoError(t, err)

		var r optypes.Receipt
		require.NoError(t, r.UnmarshalBinary(raw))
		reencoded, err := r.MarshalBinary()
		require.NoErrorf(t, err, "receipt %d (type %#x)", i, want.Type)
		require.Equalf(t, raw, reencoded, "receipt %d (type %#x)", i, want.Type)
	}
}

// TestReceiptRLPRoundTrip pins that rlp round-trips
// through the wrapper's own codec — including a deposit receipt whose
// authoritative fields live only on the wrapper.
func TestReceiptRLPRoundTrip(t *testing.T) {
	for i, want := range consensusReceiptCases() {
		encoded, err := rlp.EncodeToBytes(want)
		require.NoError(t, err)

		var decoded optypes.Receipt
		require.NoErrorf(t, rlp.DecodeBytes(encoded, &decoded), "receipt %d (type %#x)", i, want.Type)
		require.Equal(t, want.DepositNonce, decoded.DepositNonce)
		require.Equal(t, want.DepositReceiptVersion, decoded.DepositReceiptVersion)

		reencoded, err := rlp.EncodeToBytes(&decoded)
		require.NoError(t, err)
		require.Equalf(t, encoded, reencoded, "receipt %d (type %#x)", i, want.Type)
	}
}

func TestReceiptUnmarshalBinaryErrors(t *testing.T) {
	var r optypes.Receipt
	require.Error(t, r.UnmarshalBinary(nil))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.DepositTxType}))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.DepositTxType, 0xff}))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.PostExecTxType, 0xff}))
}

// TestReceiptSizeCountsExtensions pins that Size accounts for the wrapper's
// extension fields — the promoted go-ethereum method measured only the
// embedded receipt.
func TestReceiptSizeCountsExtensions(t *testing.T) {
	gr := gethReceipt()
	bare := &optypes.Receipt{Receipt: *gr}
	full := &optypes.Receipt{Receipt: *gr}
	full.L1GasPrice = big.NewInt(42_000_000_000)
	full.L1Fee = big.NewInt(77_000)
	full.OperatorFeeScalar = uint64Ptr(7)

	require.Greater(t, uint64(bare.Size()), uint64(gr.Size()), "wrapper overhead counted")
	require.Greater(t, uint64(full.Size()), uint64(bare.Size()), "pointees counted")
}

func TestReceiptsGeth(t *testing.T) {
	rs := optypes.Receipts(consensusReceiptCases())
	view := rs.Geth()
	require.Len(t, view, len(rs))
	for i := range rs {
		require.Same(t, &rs[i].Receipt, view[i])
	}
	// Mutations through the view are visible on the wrapper.
	view[0].CumulativeGasUsed = 123
	require.EqualValues(t, 123, rs[0].CumulativeGasUsed)
}

func TestFromGethReceipt(t *testing.T) {
	for _, typ := range []uint8{gethtypes.LegacyTxType, gethtypes.DynamicFeeTxType, optypes.PostExecTxType} {
		src := &gethtypes.Receipt{Type: typ, TxHash: common.Hash{typ}}
		r, err := optypes.FromGethReceipt(src)
		require.NoError(t, err, "type %#x", typ)
		require.Equal(t, *src, r.Receipt)
	}

	rs, err := optypes.FromGethReceipts(gethtypes.Receipts{
		{Type: gethtypes.LegacyTxType},
		{Type: optypes.PostExecTxType},
	})
	require.NoError(t, err)
	require.Len(t, rs, 2)

	_, err = optypes.FromGethReceipt(&gethtypes.Receipt{Type: optypes.DepositTxType})
	require.ErrorContains(t, err, "deposit receipt")
	_, err = optypes.FromGethReceipts(gethtypes.Receipts{
		{Type: gethtypes.LegacyTxType},
		{Type: optypes.DepositTxType},
	})
	require.ErrorContains(t, err, "deposit receipt")
}

func TestReceiptsUserReceipts(t *testing.T) {
	mk := func(typ uint8) *optypes.Receipt {
		return &optypes.Receipt{Receipt: gethtypes.Receipt{Type: typ, TxHash: common.Hash{typ}}}
	}
	rs := optypes.Receipts{
		mk(optypes.DepositTxType),
		mk(gethtypes.LegacyTxType),
		mk(optypes.PostExecTxType),
		mk(gethtypes.DynamicFeeTxType),
	}
	user := rs.UserReceipts()
	require.Equal(t, optypes.Receipts{rs[1], rs[3]}, user)
	require.Same(t, rs[1], user[0], "elements alias the receiver's")
	require.Empty(t, optypes.Receipts{mk(optypes.DepositTxType)}.UserReceipts())
}
