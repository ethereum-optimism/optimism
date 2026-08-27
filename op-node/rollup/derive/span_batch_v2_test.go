package derive

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

const (
	spanBatchV2GenesisTimestamp = 1000
	spanBatchV2BlockTime        = 2
)

func spanBatchV2ChainID() *big.Int { return big.NewInt(1234) }

func spanBatchV2RollupConfig() *rollup.Config {
	return &rollup.Config{
		Genesis:   rollup.Genesis{L2Time: spanBatchV2GenesisTimestamp},
		BlockTime: spanBatchV2BlockTime,
		L2ChainID: spanBatchV2ChainID(),
	}
}

// spanBatchV2Tx builds a deterministic signed transaction, so the golden vectors below are stable.
func spanBatchV2Tx(t *testing.T, nonce uint64) hexutil.Bytes {
	t.Helper()
	key, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
	require.NoError(t, err)
	to := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	tx := types.MustSignNewTx(key, types.NewLondonSigner(spanBatchV2ChainID()), &types.DynamicFeeTx{
		ChainID:   spanBatchV2ChainID(),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(10),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(int64(nonce) + 1),
		Data:      []byte{byte(nonce)},
	})
	raw, err := tx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

// spanBatchV2Element builds a singular batch at the given timestamp with txCount deterministic txs.
func spanBatchV2Element(t *testing.T, timestamp uint64, txCount int) *SingularBatch {
	t.Helper()
	txs := make([]hexutil.Bytes, 0, txCount)
	for i := 0; i < txCount; i++ {
		txs = append(txs, spanBatchV2Tx(t, timestamp+uint64(i)))
	}
	return &SingularBatch{
		ParentHash:   common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		EpochNum:     200,
		EpochHash:    common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		Timestamp:    timestamp,
		Transactions: txs,
	}
}

// buildSpanBatchV2 assembles a span batch v2 from (timestamp, tx count) pairs, on a parent block
// with the given timestamp. seqNum of the first element is non-zero, so its origin bit stays clear.
func buildSpanBatchV2(t *testing.T, parentTimestamp uint64, elems [][2]uint64) *SpanBatch {
	t.Helper()
	sb := NewSpanBatch(SpanBatchV2Type, spanBatchV2GenesisTimestamp, spanBatchV2ChainID(), parentTimestamp)
	for i, elem := range elems {
		require.NoError(t, sb.AppendSingularBatch(spanBatchV2Element(t, elem[0], int(elem[1])), uint64(5+i)))
	}
	return sb
}

func TestSpanBatchV2RoundTrip(t *testing.T) {
	for _, test := range []struct {
		name            string
		parentTimestamp uint64
		elems           [][2]uint64
		expTimestamps   []uint64
		expSameTsBits   []uint
	}{
		{
			name:            "group of three, bit 0 clear",
			parentTimestamp: 1008,
			elems:           [][2]uint64{{1010, 1}, {1010, 0}, {1010, 2}, {1012, 1}, {1012, 0}},
			expTimestamps:   []uint64{1010, 1010, 1010, 1012, 1012},
			expSameTsBits:   []uint{0, 1, 1, 0, 1},
		},
		{
			name:            "first element continues the parent's group, bit 0 set",
			parentTimestamp: 1010,
			elems:           [][2]uint64{{1010, 1}, {1010, 0}, {1012, 2}},
			expTimestamps:   []uint64{1010, 1010, 1012},
			expSameTsBits:   []uint{1, 1, 0},
		},
		{
			name:            "no siblings at all",
			parentTimestamp: 1008,
			elems:           [][2]uint64{{1010, 1}, {1012, 1}},
			expTimestamps:   []uint64{1010, 1012},
			expSameTsBits:   []uint{0, 0},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			sb := buildSpanBatchV2(t, test.parentTimestamp, test.elems)
			raw, err := sb.ToRawSpanBatch()
			require.NoError(t, err)
			require.Equal(t, SpanBatchV2Type, raw.GetBatchType())
			for i, exp := range test.expSameTsBits {
				require.Equalf(t, exp, raw.sameTsBits.Bit(i), "same_ts_bit %d", i)
			}

			encoded, err := NewBatchData(raw).MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, byte(SpanBatchV2Type), encoded[0])

			var decoded BatchData
			require.NoError(t, decoded.UnmarshalBinary(encoded))
			require.Equal(t, uint8(SpanBatchV2Type), decoded.GetBatchType())

			derived, err := DeriveSpanBatch(&decoded, spanBatchV2RollupConfig())
			require.NoError(t, err)
			require.Equal(t, SpanBatchV2Type, derived.GetBatchType())

			gotTimestamps := make([]uint64, 0, len(derived.Batches))
			for _, elem := range derived.Batches {
				gotTimestamps = append(gotTimestamps, elem.Timestamp)
			}
			require.Equal(t, test.expTimestamps, gotTimestamps)

			// re-encoding the decoded batch must reproduce the exact same bytes
			reEncoded, err := decoded.MarshalBinary()
			require.NoError(t, err)
			require.Equal(t, encoded, reEncoded)
		})
	}
}

// TestSpanBatchV2RejectsTruncatedSameTsBits checks that a v2 batch whose same_ts_bits bitlist is cut
// short is rejected instead of being read into the following fields.
func TestSpanBatchV2RejectsTruncatedSameTsBits(t *testing.T) {
	sb := buildSpanBatchV2(t, 1008, [][2]uint64{{1010, 1}, {1010, 0}, {1012, 1}})
	raw, err := sb.ToRawSpanBatch()
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, raw.encodePrefix(&buf))
	require.NoError(t, raw.encodeBlockCount(&buf))
	require.NoError(t, raw.encodeOriginBits(&buf))
	// same_ts_bits omitted entirely, so the block tx counts are read as the bitlist and the payload
	// runs out of bytes
	require.NoError(t, raw.encodeBlockTxCounts(&buf))

	decoded := RawSpanBatch{version: SpanBatchV2Type}
	require.Error(t, decoded.decode(bytes.NewReader(buf.Bytes())))
}

// TestSpanBatchV1HasNoSameTsBits pins that the v1 wire format is untouched: the same elements
// encoded as v1 are shorter by exactly the same_ts_bits bitlist, and decode without it.
func TestSpanBatchV1HasNoSameTsBits(t *testing.T) {
	elems := [][2]uint64{{1010, 1}, {1012, 0}, {1014, 2}}

	v1 := NewSpanBatch(SpanBatchType, spanBatchV2GenesisTimestamp, spanBatchV2ChainID(), 1008)
	for i, elem := range elems {
		require.NoError(t, v1.AppendSingularBatch(spanBatchV2Element(t, elem[0], int(elem[1])), uint64(5+i)))
	}
	v1Raw, err := v1.ToRawSpanBatch()
	require.NoError(t, err)
	require.Nil(t, v1Raw.sameTsBits)
	v1Bytes, err := NewBatchData(v1Raw).MarshalBinary()
	require.NoError(t, err)
	require.Equal(t, byte(SpanBatchType), v1Bytes[0])

	v2Raw, err := buildSpanBatchV2(t, 1008, elems).ToRawSpanBatch()
	require.NoError(t, err)
	v2Bytes, err := NewBatchData(v2Raw).MarshalBinary()
	require.NoError(t, err)

	// one byte holds all three bits of the bitlist
	require.Equal(t, len(v1Bytes)+1, len(v2Bytes))
}

// TestSpanBatchV2GoldenVectors pins the v2 wire format against checked-in vectors that kona's
// `test_decode_encode_raw_span_batch_v2` decodes byte-for-byte. Each file holds the ASCII hex of a
// complete typed batch, i.e. the 0x02 type byte followed by the span batch prefix and payload.
// Set UPDATE_GOLDEN=1 to rewrite them after an intentional format change — and copy the new bytes
// to kona's `crates/protocol/protocol/src/batch/testdata/`.
func TestSpanBatchV2GoldenVectors(t *testing.T) {
	for _, test := range []struct {
		file            string
		parentTimestamp uint64
		elems           [][2]uint64
	}{
		{
			file:            "raw_batch_v2.hex",
			parentTimestamp: 1008,
			elems:           [][2]uint64{{1010, 1}, {1010, 0}, {1010, 2}, {1012, 1}, {1012, 0}},
		},
		{
			file:            "raw_batch_v2_bit0_set.hex",
			parentTimestamp: 1010,
			elems:           [][2]uint64{{1010, 1}, {1010, 0}, {1012, 2}},
		},
	} {
		t.Run(test.file, func(t *testing.T) {
			raw, err := buildSpanBatchV2(t, test.parentTimestamp, test.elems).ToRawSpanBatch()
			require.NoError(t, err)
			encoded, err := NewBatchData(raw).MarshalBinary()
			require.NoError(t, err)

			path := filepath.Join("testdata", test.file)
			if os.Getenv("UPDATE_GOLDEN") != "" {
				require.NoError(t, os.WriteFile(path, []byte(hex.EncodeToString(encoded)+"\n"), 0o644))
			}
			want, err := os.ReadFile(path)
			require.NoError(t, err)
			wantBytes, err := hex.DecodeString(strings.TrimSpace(string(want)))
			require.NoError(t, err)
			require.Equal(t, wantBytes, encoded)

			var decoded BatchData
			require.NoError(t, decoded.UnmarshalBinary(wantBytes))
			require.Equal(t, uint8(SpanBatchV2Type), decoded.GetBatchType())
		})
	}
}
