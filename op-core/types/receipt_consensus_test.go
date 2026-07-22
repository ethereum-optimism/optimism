package types_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	optypes "github.com/ethereum-optimism/optimism/op-core/types"
)

// consensusReceiptCases covers every receipts-root encoding arm: legacy,
// standard typed, post-exec (0x7D), and the three deposit shapes — plain
// (pre-Regolith), nonce-only (post-Regolith pre-Canyon, whose hash preimage
// deliberately EXCLUDES the nonce), and nonce+version (post-Canyon).
func consensusReceiptCases() []*gethtypes.Receipt {
	mk := func(typ uint8, status uint64, mut func(*gethtypes.Receipt)) *gethtypes.Receipt {
		r := &gethtypes.Receipt{
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
		}
		r.Bloom = gethtypes.CreateBloom(r)
		if mut != nil {
			mut(r)
		}
		return r
	}
	return []*gethtypes.Receipt{
		mk(gethtypes.LegacyTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.DynamicFeeTxType, gethtypes.ReceiptStatusFailed, nil),
		mk(gethtypes.DynamicFeeTxType, gethtypes.ReceiptStatusSuccessful, func(r *gethtypes.Receipt) {
			r.Logs = []*gethtypes.Log{}
			r.Bloom = gethtypes.CreateBloom(r)
		}),
		mk(gethtypes.BlobTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.SetCodeTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.PostExecTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, nil),
		mk(gethtypes.DepositTxType, gethtypes.ReceiptStatusSuccessful, func(r *gethtypes.Receipt) {
			r.DepositNonce = uint64Ptr(7) // pre-Canyon: nonce set, no version
		}),
		mk(gethtypes.DepositTxType, gethtypes.ReceiptStatusFailed, func(r *gethtypes.Receipt) {
			r.DepositNonce = uint64Ptr(8)
			r.DepositReceiptVersion = uint64Ptr(gethtypes.CanyonDepositReceiptVersion)
		}),
	}
}

func asOptypesReceipts(t *testing.T, gethReceipts []*gethtypes.Receipt) optypes.Receipts {
	t.Helper()
	out := make(optypes.Receipts, len(gethReceipts))
	for i, gr := range gethReceipts {
		r := &optypes.Receipt{Receipt: *gr}
		// Field shadowing: the outer OP fields are authoritative.
		r.DepositNonce = gr.DepositNonce
		r.DepositReceiptVersion = gr.DepositReceiptVersion
		out[i] = r
	}
	return out
}

// TestReceiptsEncodeIndexDifferential asserts that receipts-root derivation
// over optypes.Receipts matches op-geth's, for every encoding arm. It will be
// removed in the final cutover, when the op-geth dependency is replaced with
// upstream go-ethereum.
func TestReceiptsEncodeIndexDifferential(t *testing.T) {
	gethReceipts := consensusReceiptCases()
	ours := asOptypesReceipts(t, gethReceipts)

	theirRoot := gethtypes.DeriveSha(gethtypes.Receipts(gethReceipts), trie.NewStackTrie(nil))
	ourRoot := gethtypes.DeriveSha(ours, trie.NewStackTrie(nil))
	require.Equal(t, theirRoot, ourRoot)

	// Also compare per-index encodings, so a mismatch pinpoints the arm.
	for i, gr := range gethReceipts {
		var theirs, oursBuf bytes.Buffer
		gethtypes.Receipts{gr}.EncodeIndex(0, &theirs)
		ours.EncodeIndex(i, &oursBuf)
		require.Equalf(t, theirs.Bytes(), oursBuf.Bytes(), "receipt %d (type %#x)", i, gr.Type)
	}
}

// TestReceiptsEncodeIndexDepositArms pins the deposit hash-preimage rules
// without reference to op-geth, so the receipts-root arms stay covered after
// the differential tests are removed at the cutover:
//   - pre-Canyon (nonce set, version nil): the preimage EXCLUDES the nonce, so
//     it must equal the encoding of the same receipt without a nonce;
//   - post-Canyon (version set): the preimage includes nonce+version, so it
//     must differ from the pre-Canyon encoding.
func TestReceiptsEncodeIndexDepositArms(t *testing.T) {
	var base *gethtypes.Receipt
	for _, r := range consensusReceiptCases() {
		if r.Type == gethtypes.DepositTxType && r.DepositNonce != nil && r.DepositReceiptVersion == nil {
			base = r // pre-Canyon shape: nonce set, version nil
		}
	}
	require.NotNil(t, base)

	encode := func(nonce, version *uint64) []byte {
		cp := *base
		r := &optypes.Receipt{Receipt: cp}
		r.DepositNonce = nonce
		r.DepositReceiptVersion = version
		var buf bytes.Buffer
		optypes.Receipts{r}.EncodeIndex(0, &buf)
		return buf.Bytes()
	}

	preCanyon := encode(base.DepositNonce, nil)
	noNonce := encode(nil, nil)
	postCanyon := encode(base.DepositNonce, uint64Ptr(gethtypes.CanyonDepositReceiptVersion))

	require.Equal(t, noNonce, preCanyon, "pre-Canyon deposit receipt hash preimage must exclude the nonce")
	require.NotEqual(t, preCanyon, postCanyon, "post-Canyon deposit receipt hash preimage must include nonce+version")
}

// TestReceiptsEncodeIndexOuterFieldsAuthoritative pins the Receipt contract
// that the OUTER deposit fields drive the consensus encoding: a wrapper
// hand-built around an embedded receipt whose deposit fields were never
// mirrored to the wrapper encodes without them. Receipts must be produced
// through this package's decoders.
func TestReceiptsEncodeIndexOuterFieldsAuthoritative(t *testing.T) {
	gr := *consensusReceiptCases()[0] // legacy shape, mutate into a deposit
	gr.Type = gethtypes.DepositTxType
	gr.DepositNonce = uint64Ptr(7)
	gr.DepositReceiptVersion = uint64Ptr(gethtypes.CanyonDepositReceiptVersion)

	desynced := &optypes.Receipt{Receipt: gr} // outer deposit fields left nil

	mirrored := &optypes.Receipt{Receipt: gr}
	mirrored.DepositNonce = gr.DepositNonce
	mirrored.DepositReceiptVersion = gr.DepositReceiptVersion

	var desyncedBuf, mirroredBuf bytes.Buffer
	optypes.Receipts{desynced}.EncodeIndex(0, &desyncedBuf)
	optypes.Receipts{mirrored}.EncodeIndex(0, &mirroredBuf)
	require.NotEqual(t, mirroredBuf.Bytes(), desyncedBuf.Bytes())
}

// TestReceiptUnmarshalBinaryDifferential asserts the consensus decode against
// op-geth's MarshalBinary for every encoding arm. It will be removed in the
// final cutover, when the op-geth dependency is replaced with upstream
// go-ethereum.
func TestReceiptUnmarshalBinaryDifferential(t *testing.T) {
	for i, gr := range consensusReceiptCases() {
		raw, err := gr.MarshalBinary()
		require.NoError(t, err)

		var r optypes.Receipt
		require.NoErrorf(t, r.UnmarshalBinary(raw), "receipt %d (type %#x)", i, gr.Type)

		require.Equal(t, gr.Type, r.Type)
		require.Equal(t, gr.Status, r.Status)
		require.Equal(t, gr.CumulativeGasUsed, r.CumulativeGasUsed)
		require.Equal(t, gr.Bloom, r.Bloom)
		require.Equal(t, gr.Logs, r.Logs)
		require.Equal(t, gr.DepositNonce, r.DepositNonce)
		require.Equal(t, gr.DepositReceiptVersion, r.DepositReceiptVersion)
	}
}

// TestReceiptMarshalBinaryDifferential asserts the consensus encode against
// op-geth's MarshalBinary for every encoding arm — including the deposit arm,
// which (unlike the receipts-root derivation) includes the nonce whenever set.
// It will be removed in the final cutover, when the op-geth dependency is
// replaced with upstream go-ethereum.
func TestReceiptMarshalBinaryDifferential(t *testing.T) {
	for i, gr := range consensusReceiptCases() {
		theirs, err := gr.MarshalBinary()
		require.NoError(t, err)

		r := &optypes.Receipt{Receipt: *gr}
		r.DepositNonce = gr.DepositNonce
		r.DepositReceiptVersion = gr.DepositReceiptVersion
		ours, err := r.MarshalBinary()
		require.NoErrorf(t, err, "receipt %d (type %#x)", i, gr.Type)
		require.Equalf(t, theirs, ours, "receipt %d (type %#x)", i, gr.Type)
	}
}

// TestReceiptBinaryRoundTrip pins, op-geth-independently, that the wrapper's
// binary codec round-trips — in particular that a deposit receipt built only
// through this package's own decoder re-encodes identically (P2 of the #21907
// review: the encoder must read the outer deposit fields).
func TestReceiptBinaryRoundTrip(t *testing.T) {
	for i, gr := range consensusReceiptCases() {
		raw, err := gr.MarshalBinary()
		require.NoError(t, err)

		var r optypes.Receipt
		require.NoError(t, r.UnmarshalBinary(raw))
		reencoded, err := r.MarshalBinary()
		require.NoErrorf(t, err, "receipt %d (type %#x)", i, gr.Type)
		require.Equalf(t, raw, reencoded, "receipt %d (type %#x)", i, gr.Type)
	}
}

// TestReceiptRLPDifferential asserts the rlp.Encoder/Decoder implementation
// against op-geth's for every encoding arm: identical encoding, and decode
// equivalence via re-encode. It will be removed in the final cutover, when the
// op-geth dependency is replaced with upstream go-ethereum.
func TestReceiptRLPDifferential(t *testing.T) {
	for i, gr := range consensusReceiptCases() {
		theirs, err := rlp.EncodeToBytes(gr)
		require.NoError(t, err)

		r := &optypes.Receipt{Receipt: *gr}
		r.DepositNonce = gr.DepositNonce
		r.DepositReceiptVersion = gr.DepositReceiptVersion
		ours, err := rlp.EncodeToBytes(r)
		require.NoErrorf(t, err, "receipt %d (type %#x)", i, gr.Type)
		require.Equalf(t, theirs, ours, "receipt %d (type %#x)", i, gr.Type)
	}
}

// TestReceiptRLPRoundTrip pins, op-geth-independently, that rlp round-trips
// through the wrapper's own codec — including a deposit receipt whose
// authoritative fields live only on the wrapper.
func TestReceiptRLPRoundTrip(t *testing.T) {
	for i, gr := range consensusReceiptCases() {
		encoded, err := rlp.EncodeToBytes(&optypes.Receipt{Receipt: *gr, DepositNonce: gr.DepositNonce, DepositReceiptVersion: gr.DepositReceiptVersion})
		require.NoError(t, err)

		var decoded optypes.Receipt
		require.NoErrorf(t, rlp.DecodeBytes(encoded, &decoded), "receipt %d (type %#x)", i, gr.Type)
		require.Equal(t, gr.DepositNonce, decoded.DepositNonce)
		require.Equal(t, gr.DepositReceiptVersion, decoded.DepositReceiptVersion)

		reencoded, err := rlp.EncodeToBytes(&decoded)
		require.NoError(t, err)
		require.Equalf(t, encoded, reencoded, "receipt %d (type %#x)", i, gr.Type)
	}
}

func TestReceiptUnmarshalBinaryErrors(t *testing.T) {
	var r optypes.Receipt
	require.Error(t, r.UnmarshalBinary(nil))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.DepositTxType}))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.DepositTxType, 0xff}))
	require.Error(t, r.UnmarshalBinary([]byte{optypes.PostExecTxType, 0xff}))
}

func TestReceiptsGeth(t *testing.T) {
	rs := asOptypesReceipts(t, consensusReceiptCases())
	view := rs.Geth()
	require.Len(t, view, len(rs))
	for i := range rs {
		require.Same(t, &rs[i].Receipt, view[i])
	}
	// Mutations through the view are visible on the wrapper.
	view[0].CumulativeGasUsed = 123
	require.EqualValues(t, 123, rs[0].CumulativeGasUsed)
}
