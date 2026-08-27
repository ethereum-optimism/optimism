package proofbatch

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func testBatch() *ProofBatch {
	return &ProofBatch{
		PrevOutputRoot:   common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
		NewOutputRoot:    common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		L1Head:           common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333"),
		RollupConfigHash: common.HexToHash("0x4444444444444444444444444444444444444444444444444444444444444444"),
		DepSetHash:       common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555"),
		ExportPolicyHash: ExportPolicyAllHashes,
		Blocks: []BlockExport{
			{
				Number: 100, Timestamp: 1000,
				Hash:                     common.HexToHash("0xb100"),
				StateRoot:                common.HexToHash("0x5100"),
				MessagePasserStorageRoot: common.HexToHash("0x7100"),
				Logs: []LogExport{
					{Index: 0, Hash: common.HexToHash("0xaa")},
					{Index: 4, Hash: common.HexToHash("0xbb")},
				},
			},
			{
				Number: 101, Timestamp: 1002,
				Hash:                     common.HexToHash("0xb101"),
				StateRoot:                common.HexToHash("0x5101"),
				MessagePasserStorageRoot: common.HexToHash("0x7101"),
			},
			{
				Number: 102, Timestamp: 1004,
				Hash:                     common.HexToHash("0xb102"),
				StateRoot:                common.HexToHash("0x5102"),
				MessagePasserStorageRoot: common.HexToHash("0x7102"),
				Logs:                     []LogExport{{Index: 11, Hash: common.HexToHash("0xcc")}},
			},
		},
	}
}

// testExecMsg is one import, with every field a distinct recognisable value so that a reordering
// inside the Identifier tuple cannot pass the layout test.
func testExecMsg() ExecMsg {
	return ExecMsg{Message: messages.Message{
		Identifier: messages.Identifier{
			Origin:      common.HexToAddress("0x1111111111111111111111111111111111111111"),
			BlockNumber: 42,
			LogIndex:    5,
			Timestamp:   80,
			ChainID:     eth.ChainIDFromUInt64(438),
		},
		PayloadHash: repeatHash(0xf1),
	}}
}

// TestPublicValuesLayout pins the v3 ABI encoding against the layout the spec's Solidity struct
// implies, written out word by word rather than produced by this codec. It is the standalone
// counterpart to the cross-language fixtures: if the canonical set is not in the tree, this is still
// what stops a field being added, reordered or re-typed unnoticed.
//
// The v3 half that needs pinning is not just "there is one more field": ExecMsg is a STATIC tuple
// (an address, three integers and two 32-byte words), so its array is encoded inline with no
// per-element offset words, unlike logs[]. That is exactly the sort of thing two implementations
// disagree about silently.
func TestPublicValuesLayout(t *testing.T) {
	preimage := append(bytes.Repeat([]byte{0x11}, 20), 0xde, 0xad, 0xbe, 0xef)
	preimageLogHash, err := LogHashFromPreimage(preimage)
	require.NoError(t, err)

	batch := &ProofBatch{
		PrevOutputRoot:   repeatHash(0x11),
		NewOutputRoot:    repeatHash(0x22),
		L1Head:           repeatHash(0x33),
		RollupConfigHash: repeatHash(0x44),
		DepSetHash:       repeatHash(0x55),
		ExportPolicyHash: repeatHash(0x66),
		Blocks: []BlockExport{
			{
				Number: 7, Timestamp: 100,
				Hash:                     repeatHash(0x77),
				StateRoot:                repeatHash(0x88),
				MessagePasserStorageRoot: repeatHash(0x99),
				Logs: []LogExport{
					{Index: 3, Hash: repeatHash(0xaa)},
					{Index: 9, Hash: preimageLogHash, Preimage: preimage},
				},
				ExecMsgs: []ExecMsg{testExecMsg()},
			},
			{
				Number: 8, Timestamp: 102,
				Hash:                     repeatHash(0xcc),
				StateRoot:                repeatHash(0xdd),
				MessagePasserStorageRoot: repeatHash(0xee),
			},
		},
	}
	want := words(t,
		"0000000000000000000000000000000000000000000000000000000000000020", // offset of the ProofBatch tuple
		// --- ProofBatch body (base = 0x20) ---
		"1111111111111111111111111111111111111111111111111111111111111111", // prevOutputRoot
		"2222222222222222222222222222222222222222222222222222222222222222", // newOutputRoot
		"3333333333333333333333333333333333333333333333333333333333333333", // l1Head
		"4444444444444444444444444444444444444444444444444444444444444444", // rollupConfigHash
		"5555555555555555555555555555555555555555555555555555555555555555", // depSetHash
		"6666666666666666666666666666666666666666666666666666666666666666", // exportPolicyHash
		"00000000000000000000000000000000000000000000000000000000000000e0", // offset of blocks[] (7 words in)
		// --- blocks[] ---
		"0000000000000000000000000000000000000000000000000000000000000002", // length
		"0000000000000000000000000000000000000000000000000000000000000040", // offset of blocks[0]
		"0000000000000000000000000000000000000000000000000000000000000380", // offset of blocks[1]
		// --- blocks[0] body: SEVEN head words now, logs then execMsgs ---
		"0000000000000000000000000000000000000000000000000000000000000007", // blockNumber
		"0000000000000000000000000000000000000000000000000000000000000064", // timestamp
		"7777777777777777777777777777777777777777777777777777777777777777", // blockHash
		"8888888888888888888888888888888888888888888888888888888888888888", // stateRoot
		"9999999999999999999999999999999999999999999999999999999999999999", // messagePasserStorageRoot
		"00000000000000000000000000000000000000000000000000000000000000e0", // offset of logs (7 words in)
		"0000000000000000000000000000000000000000000000000000000000000260", // offset of execMsgs (after logs)
		// --- blocks[0].logs[] ---
		"0000000000000000000000000000000000000000000000000000000000000002", // length
		"0000000000000000000000000000000000000000000000000000000000000040", // offset of logs[0]
		"00000000000000000000000000000000000000000000000000000000000000c0", // offset of logs[1]
		// --- logs[0] body: an index that is NOT its position, and no preimage ---
		"0000000000000000000000000000000000000000000000000000000000000003", // logIndex
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // logHash
		"0000000000000000000000000000000000000000000000000000000000000060", // offset of preimage (3 words in)
		"0000000000000000000000000000000000000000000000000000000000000000", // preimage length
		// --- logs[1] body: 24 bytes of preimage, right-padded to one word ---
		"0000000000000000000000000000000000000000000000000000000000000009", // logIndex
		strings.TrimPrefix(preimageLogHash.Hex(), "0x"),                    // logHash
		"0000000000000000000000000000000000000000000000000000000000000060", // offset of preimage
		"0000000000000000000000000000000000000000000000000000000000000018", // preimage length (24)
		"1111111111111111111111111111111111111111deadbeef0000000000000000", // preimage
		// --- blocks[0].execMsgs[]: static elements, so NO offset words, six words inline ---
		"0000000000000000000000000000000000000000000000000000000000000001", // length
		"0000000000000000000000001111111111111111111111111111111111111111", // id.origin
		"000000000000000000000000000000000000000000000000000000000000002a", // id.blockNumber (42)
		"0000000000000000000000000000000000000000000000000000000000000005", // id.logIndex
		"0000000000000000000000000000000000000000000000000000000000000050", // id.timestamp (80)
		"00000000000000000000000000000000000000000000000000000000000001b6", // id.chainId (438)
		"f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1f1", // msgHash
		// --- blocks[1] body: no logs and no imports, so both empty arrays are still encoded ---
		"0000000000000000000000000000000000000000000000000000000000000008", // blockNumber
		"0000000000000000000000000000000000000000000000000000000000000066", // timestamp
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", // blockHash
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", // stateRoot
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", // messagePasserStorageRoot
		"00000000000000000000000000000000000000000000000000000000000000e0", // offset of logs
		"0000000000000000000000000000000000000000000000000000000000000100", // offset of execMsgs
		"0000000000000000000000000000000000000000000000000000000000000000", // logs length
		"0000000000000000000000000000000000000000000000000000000000000000", // execMsgs length
	)

	got, err := EncodePublicValuesAs(batch, VersionV3)
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(want), hexutil.Encode(got))

	back, err := DecodePublicValuesAs(got, VersionV3)
	require.NoError(t, err)
	require.Equal(t, batch.Blocks[0].ExecMsgs, back.Blocks[0].ExecMsgs,
		"the import list must round-trip field for field")
	require.Empty(t, back.Blocks[1].ExecMsgs)
	// ...and re-encoding a decoded batch reproduces the bytes, which is what makes the wire an
	// object rather than a serialisation.
	again, err := EncodePublicValuesAs(back, VersionV3)
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(got), hexutil.Encode(again))
}

// TestPublicValuesLayoutV2 keeps the RETIRED version's layout pinned, word for word, unchanged from
// the day it shipped.
//
// It is not nostalgia. The v2 golden fixtures are the cross-language contract for the chain that is
// live right now, and the rotation to v3 is an event with both versions configured at once (see
// ROTATION-V3). A v2 encoder that drifted while nobody was looking at it would turn the fixtures
// from a check into a formality, and would corrupt the one comparison a dark launch is for.
func TestPublicValuesLayoutV2(t *testing.T) {
	// One log carries a preimage, so the `bytes` member's offset/length/padding words are pinned
	// too. Its hash is derived rather than written out — a preimage-bearing log must satisfy the
	// decode-side rehash rule, and keccak cannot be inverted to make a hand-picked hash fit.
	preimage := append(bytes.Repeat([]byte{0x11}, 20), 0xde, 0xad, 0xbe, 0xef)
	preimageLogHash, err := LogHashFromPreimage(preimage)
	require.NoError(t, err)

	batch := &ProofBatch{
		PrevOutputRoot:   repeatHash(0x11),
		NewOutputRoot:    repeatHash(0x22),
		L1Head:           repeatHash(0x33),
		RollupConfigHash: repeatHash(0x44),
		DepSetHash:       repeatHash(0x55),
		ExportPolicyHash: repeatHash(0x66),
		Blocks: []BlockExport{
			{
				Number: 7, Timestamp: 100,
				Hash:                     repeatHash(0x77),
				StateRoot:                repeatHash(0x88),
				MessagePasserStorageRoot: repeatHash(0x99),
				Logs: []LogExport{
					{Index: 3, Hash: repeatHash(0xaa)},
					{Index: 9, Hash: preimageLogHash, Preimage: preimage},
				},
			},
			{
				Number: 8, Timestamp: 102,
				Hash:                     repeatHash(0xcc),
				StateRoot:                repeatHash(0xdd),
				MessagePasserStorageRoot: repeatHash(0xee),
			},
		},
	}
	// `abi.encode(ProofBatch)`: ProofBatch is dynamic (it ends in an array), so the encoding is one
	// head word holding the offset of the tuple, then the tuple body. Offsets inside a tuple or an
	// array are relative to the start of that tuple's body / that array's element region.
	want := words(t,
		"0000000000000000000000000000000000000000000000000000000000000020", // offset of the ProofBatch tuple
		// --- ProofBatch body (base = 0x20) ---
		"1111111111111111111111111111111111111111111111111111111111111111", // prevOutputRoot
		"2222222222222222222222222222222222222222222222222222222222222222", // newOutputRoot
		"3333333333333333333333333333333333333333333333333333333333333333", // l1Head
		"4444444444444444444444444444444444444444444444444444444444444444", // rollupConfigHash
		"5555555555555555555555555555555555555555555555555555555555555555", // depSetHash
		"6666666666666666666666666666666666666666666666666666666666666666", // exportPolicyHash
		"00000000000000000000000000000000000000000000000000000000000000e0", // offset of blocks[] (7 words in)
		// --- blocks[] ---
		"0000000000000000000000000000000000000000000000000000000000000002", // length
		"0000000000000000000000000000000000000000000000000000000000000040", // offset of blocks[0]
		"0000000000000000000000000000000000000000000000000000000000000280", // offset of blocks[1]
		// --- blocks[0] body ---
		"0000000000000000000000000000000000000000000000000000000000000007", // blockNumber
		"0000000000000000000000000000000000000000000000000000000000000064", // timestamp
		"7777777777777777777777777777777777777777777777777777777777777777", // blockHash
		"8888888888888888888888888888888888888888888888888888888888888888", // stateRoot
		"9999999999999999999999999999999999999999999999999999999999999999", // messagePasserStorageRoot
		"00000000000000000000000000000000000000000000000000000000000000c0", // offset of logs (5 words in)
		// --- blocks[0].logs[] ---
		"0000000000000000000000000000000000000000000000000000000000000002", // length
		"0000000000000000000000000000000000000000000000000000000000000040", // offset of logs[0]
		"00000000000000000000000000000000000000000000000000000000000000c0", // offset of logs[1]
		// --- logs[0] body: an index that is NOT its position, and no preimage ---
		"0000000000000000000000000000000000000000000000000000000000000003", // logIndex
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", // logHash
		"0000000000000000000000000000000000000000000000000000000000000060", // offset of preimage (3 words in)
		"0000000000000000000000000000000000000000000000000000000000000000", // preimage length
		// --- logs[1] body: 24 bytes of preimage, right-padded to one word ---
		"0000000000000000000000000000000000000000000000000000000000000009", // logIndex
		strings.TrimPrefix(preimageLogHash.Hex(), "0x"),                    // logHash
		"0000000000000000000000000000000000000000000000000000000000000060", // offset of preimage
		"0000000000000000000000000000000000000000000000000000000000000018", // preimage length (24)
		"1111111111111111111111111111111111111111deadbeef0000000000000000", // preimage
		// --- blocks[1] body: no logs, so an empty array is still encoded ---
		"0000000000000000000000000000000000000000000000000000000000000008", // blockNumber
		"0000000000000000000000000000000000000000000000000000000000000066", // timestamp
		"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", // blockHash
		"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", // stateRoot
		"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", // messagePasserStorageRoot
		"00000000000000000000000000000000000000000000000000000000000000c0", // offset of logs
		"0000000000000000000000000000000000000000000000000000000000000000", // logs length
	)

	got, err := EncodePublicValuesAs(batch, VersionV2)
	require.NoError(t, err)
	require.Equal(t, hexutil.Encode(want), hexutil.Encode(got))

	back, err := DecodePublicValuesAs(got, VersionV2)
	require.NoError(t, err)
	require.Equal(t, batch.L1Head, back.L1Head)
	require.Equal(t, batch.RollupConfigHash, back.RollupConfigHash)
	require.Equal(t, batch.DepSetHash, back.DepSetHash)
	require.Len(t, back.Blocks, 2)
	require.Equal(t, repeatHash(0x77), back.Blocks[0].Hash)
	require.Equal(t, repeatHash(0x88), back.Blocks[0].StateRoot)
	require.Equal(t, repeatHash(0x99), back.Blocks[0].MessagePasserStorageRoot)
	require.Equal(t, uint32(3), back.Blocks[0].Logs[0].Index)
	require.Empty(t, back.Blocks[0].Logs[0].Preimage)
	require.Equal(t, preimage, back.Blocks[0].Logs[1].Preimage)
	require.Empty(t, back.Blocks[1].Logs)
	// A v2 batch says nothing about imports, and decoding one must not invent a claim that it says
	// there were none.
	require.Nil(t, back.Blocks[0].ExecMsgs)
}

// TestEncodeRefusesImportsAtV2 is the other half of the two-version story: a v2 encoding of a batch
// that HAS an import list would be a strictly weaker claim wearing the same roots, so it is refused
// rather than quietly narrowed.
func TestEncodeRefusesImportsAtV2(t *testing.T) {
	b := testBatch()
	b.Blocks[0].ExecMsgs = []ExecMsg{testExecMsg()}
	_, err := EncodePublicValuesAs(b, VersionV2)
	require.ErrorContains(t, err, "no field for them")
	_, err = EncodeAs(b, nil, VersionV2)
	require.ErrorContains(t, err, "no field for them")

	// ...and the same batch at v3 encodes fine, so the refusal is about the version and not about
	// the batch.
	_, err = EncodeAs(b, nil, Version)
	require.NoError(t, err)
}

// TestEnvelopeFraming pins the envelope header against the spec's byte table.
func TestEnvelopeFraming(t *testing.T) {
	batch := testBatch()
	proof := []byte{0xde, 0xad, 0xbe, 0xef}
	enc, err := Encode(batch, proof)
	require.NoError(t, err)

	pv, err := EncodePublicValues(batch)
	require.NoError(t, err)

	require.Equal(t, []byte{'K', 'C', 'P', 'B'}, enc[0:4])
	require.Equal(t, byte(Version), enc[4])
	require.Equal(t, uint32(len(pv)), binary.BigEndian.Uint32(enc[5:9]))
	require.Equal(t, pv, enc[9:9+len(pv)])
	rest := enc[9+len(pv):]
	require.Equal(t, uint32(len(proof)), binary.BigEndian.Uint32(rest[0:4]))
	require.Equal(t, proof, rest[4:])
	require.Len(t, enc, 4+1+4+len(pv)+4+len(proof))

	// The framing itself is version-agnostic — only the byte at offset 4 and the payload change.
	encV2, err := EncodeAs(batch, proof, VersionV2)
	require.NoError(t, err)
	require.Equal(t, byte(0x02), encV2[4])
	pvV2, err := EncodePublicValuesAs(batch, VersionV2)
	require.NoError(t, err)
	require.Equal(t, pvV2, encV2[9:9+len(pvV2)])
	require.Less(t, len(pvV2), len(pv), "v3 adds a field, so its encoding of the same blocks is longer")
}

func repeatHash(b byte) common.Hash {
	var h common.Hash
	for i := range h {
		h[i] = b
	}
	return h
}

func words(t *testing.T, hexWords ...string) []byte {
	t.Helper()
	var out bytes.Buffer
	for _, w := range hexWords {
		require.Len(t, w, 64)
		raw, err := hexutil.Decode("0x" + strings.TrimSpace(w))
		require.NoError(t, err)
		out.Write(raw)
	}
	return out.Bytes()
}

func TestPublicValuesRoundTrip(t *testing.T) {
	batch := testBatch()
	pv, err := EncodePublicValues(batch)
	require.NoError(t, err)
	// Encoded as a single dynamic ABI value: a head offset of 0x20 then the tuple body.
	require.Equal(t, uint64(0x20), binary.BigEndian.Uint64(pv[24:32]))
	require.Zero(t, len(pv)%32)

	got, err := DecodePublicValues(pv)
	require.NoError(t, err)
	require.Equal(t, batch.PrevOutputRoot, got.PrevOutputRoot)
	require.Equal(t, batch.NewOutputRoot, got.NewOutputRoot)
	require.Equal(t, batch.L1Head, got.L1Head)
	require.Equal(t, batch.RollupConfigHash, got.RollupConfigHash)
	require.Equal(t, batch.DepSetHash, got.DepSetHash)
	require.Equal(t, batch.ExportPolicyHash, got.ExportPolicyHash)
	require.Len(t, got.Blocks, 3)
	require.Equal(t, batch.Blocks[0], got.Blocks[0])
	require.Equal(t, uint64(101), got.Blocks[1].Number)
	require.Equal(t, common.HexToHash("0xb101"), got.Blocks[1].Hash)
	require.Empty(t, got.Blocks[1].Logs)
	require.Equal(t, batch.Blocks[2], got.Blocks[2])
}

// TestOutputRootDerivation pins the v2 headline: an output root per block, derived from what the
// proof committed to. The expectation is built from the raw 128-byte preimage rather than through
// op-node's Output types, so a change to either side is visible here.
func TestOutputRootDerivation(t *testing.T) {
	blk := BlockExport{
		Hash:                     repeatHash(0xb1),
		StateRoot:                repeatHash(0x51),
		MessagePasserStorageRoot: repeatHash(0x71),
	}
	var preimage []byte
	preimage = append(preimage, make([]byte, 32)...) // version 0
	preimage = append(preimage, blk.StateRoot[:]...)
	preimage = append(preimage, blk.MessagePasserStorageRoot[:]...)
	preimage = append(preimage, blk.Hash[:]...)
	require.Len(t, preimage, 128)
	require.Equal(t, crypto.Keccak256Hash(preimage), blk.OutputRoot())
}

// TestPreimageHashing pins the preimage encoding: the emitting address, then the topics, then the
// data — the plainest encoding that determines the log hash, chosen so a second implementation cannot drift
// from it through a serialization format.
func TestPreimageHashing(t *testing.T) {
	log := &types.Log{
		Address: common.HexToAddress("0x4200000000000000000000000000000000000023"),
		Topics:  []common.Hash{repeatHash(0x01), repeatHash(0x02)},
		Data:    []byte{0xca, 0xfe},
	}
	preimage := append(log.Address.Bytes(), messages.LogToMessagePayload(log)...)
	require.Len(t, preimage, 20+64+2)

	got, err := LogHashFromPreimage(preimage)
	require.NoError(t, err)
	require.Equal(t, messages.LogToLogHash(log), got)

	t.Run("too short", func(t *testing.T) {
		_, err := LogHashFromPreimage(make([]byte, 19))
		require.ErrorContains(t, err, "emitting address")
	})
	t.Run("empty payload is legal", func(t *testing.T) {
		_, err := LogHashFromPreimage(make([]byte, 20))
		require.NoError(t, err)
	})
}

func TestEnvelopeRoundTrip(t *testing.T) {
	batch := testBatch()
	proof := []byte{1, 2, 3, 4, 5}
	enc, err := Encode(batch, proof)
	require.NoError(t, err)
	require.Equal(t, Magic, string(enc[:4]))
	require.Equal(t, byte(Version), enc[4])

	env, err := Decode(enc)
	require.NoError(t, err)
	require.Equal(t, proof, env.Proof)
	require.Equal(t, batch.NewOutputRoot, env.Batch.NewOutputRoot)
	require.Len(t, env.Batch.Blocks, 3)

	// The retained public values are exactly the bytes a proof commits to.
	pv, err := EncodePublicValues(batch)
	require.NoError(t, err)
	require.Equal(t, pv, env.PublicValues)
}

func TestEnvelopeEmptyProofSlot(t *testing.T) {
	enc, err := Encode(testBatch(), nil)
	require.NoError(t, err)
	env, err := Decode(enc)
	require.NoError(t, err)
	require.Empty(t, env.Proof)
}

func TestDecodeRejections(t *testing.T) {
	valid, err := Encode(testBatch(), []byte{9, 9})
	require.NoError(t, err)

	t.Run("empty", func(t *testing.T) {
		_, err := Decode(nil)
		require.ErrorIs(t, err, ErrTruncated)
	})
	t.Run("bad magic", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[0] = 'X'
		_, err := Decode(bad)
		require.ErrorIs(t, err, ErrBadMagic)
	})
	// v1 is a real format that a still-running submitter could post across a rotation, so it is
	// worth its own case: this codec must refuse it rather than mis-read its blocks.
	t.Run("version 1", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[4] = VersionV1
		_, err := Decode(bad)
		require.ErrorIs(t, err, ErrBadVersion)
	})
	// v2 is an older version. A current-version node refuses it — the point of the rotation is
	// that a node accepts exactly what its config pins — even though this codec can still read it.
	t.Run("version 2 against a current node", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[4] = VersionV2
		_, err := Decode(bad)
		require.ErrorIs(t, err, ErrBadVersion)
		require.ErrorContains(t, err, "this node accepts 4")
	})
	t.Run("version 3 against a v2 node", func(t *testing.T) {
		_, err := DecodeAs(valid, VersionV2)
		require.ErrorIs(t, err, ErrBadVersion)
		require.ErrorContains(t, err, "this node accepts 2")
	})
	t.Run("version 5", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[4] = 5
		_, err := Decode(bad)
		require.ErrorIs(t, err, ErrBadVersion)
	})
	// A node cannot be configured to accept a version this codec does not implement: the refusal is
	// of the CONFIGURATION, before any bytes are read, so a typo in a manifest cannot become a node
	// that accepts nothing while looking healthy.
	t.Run("unimplemented accept version", func(t *testing.T) {
		_, err := DecodeAs(valid, VersionV1)
		require.ErrorIs(t, err, ErrBadVersion)
		require.ErrorContains(t, err, "cannot accept wire version 1")
	})
	t.Run("truncated public values", func(t *testing.T) {
		_, err := Decode(valid[:len(valid)-10])
		require.ErrorIs(t, err, ErrTruncated)
	})
	t.Run("oversized length field", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		binary.BigEndian.PutUint32(bad[5:], 0xffffffff)
		_, err := Decode(bad)
		require.ErrorIs(t, err, ErrTruncated)
	})
	t.Run("trailing data", func(t *testing.T) {
		_, err := Decode(append(append([]byte{}, valid...), 0x00))
		require.ErrorIs(t, err, ErrTrailingData)
	})
	t.Run("public values not ABI", func(t *testing.T) {
		bad := EncodeWithPublicValues([]byte{1, 2, 3}, nil)
		_, err := Decode(bad)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrTruncated)
	})
}

// TestVersionHasExecMsgsIsPinnedToV3 is a regression test against a mistake that cannot be observed
// until the day it is made.
//
// "Does this wire version carry an import list" is a fixed fact about version 3. Written as
// `version >= Version` — a comparison against whatever this codec currently ENCODES — it silently
// inverts on the next version bump: the v3 ABI type loses its execMsgs field, so a v3 batch's imports
// decode to nil, and Fact.ExecMsgsKnown then records them as UNKNOWN. The result is a node configured
// for v3, accepting v3 batches, validating no dependencies at all, with nothing red anywhere.
//
// So the predicate is pinned to the literal 3, and to VersionV2 being below it, INDEPENDENTLY of what
// Version happens to be. If someone bumps Version to 4 and this test still passes, v3 batches still
// carry their imports.
func TestVersionHasExecMsgsIsPinnedToV3(t *testing.T) {
	require.Equal(t, 3, VersionV3, "VersionV3 is the wire version that introduced the import list")
	require.False(t, VersionHasExecMsgs(VersionV1))
	require.False(t, VersionHasExecMsgs(VersionV2), "v2 says nothing about imports")
	require.True(t, VersionHasExecMsgs(VersionV3), "v3 is where the import list starts")
	require.True(t, VersionHasExecMsgs(4), "a future version must not LOSE the field")
	require.True(t, VersionHasExecMsgs(255))

	// And the ABI layout follows the same predicate: the v3 type must carry execMsgs no matter what
	// the codec's current version is. Encoding a batch WITH imports at v3 must succeed, which is the
	// operational form of the claim.
	b := testBatch()
	b.Blocks[0].ExecMsgs = []ExecMsg{testExecMsg()}
	_, err := EncodePublicValuesAs(b, VersionV3)
	require.NoError(t, err, "the v3 ABI type must have a field for the import list, always")

	// ...and a version that does NOT carry them refuses rather than dropping them silently.
	_, err = EncodePublicValuesAs(b, VersionV2)
	require.ErrorContains(t, err, "no field for them")
}

// TestDecodeAnyReadsWhicheverVersionItIsGiven pins the one place leniency is CORRECT.
//
// Acceptance must be strict — a verifier's version is part of its rule (TestDecodeRejections) — but a
// diagnostic has the opposite requirement: an inspector that could only read the codec's current
// version would be unable to read the version the live chain is running, which is exactly when
// somebody reaches for it.
func TestDecodeAnyReadsWhicheverVersionItIsGiven(t *testing.T) {
	for _, version := range []uint8{VersionV2, Version} {
		b := testBatch()
		if VersionHasExecMsgs(version) {
			b.Blocks[0].ExecMsgs = []ExecMsg{testExecMsg()}
		}
		raw, err := EncodeAs(b, []byte{7}, version)
		require.NoError(t, err)

		env, err := DecodeAny(raw)
		require.NoError(t, err, "DecodeAny must read v%d", version)
		require.Equal(t, version, env.Version, "the envelope must report the version it was framed at")
		require.Len(t, env.Batch.Blocks, len(b.Blocks))
		if VersionHasExecMsgs(version) {
			require.Len(t, env.Batch.Blocks[0].ExecMsgs, 1)
		} else {
			require.Nil(t, env.Batch.Blocks[0].ExecMsgs)
		}
	}

	// It is lenient about the VERSION and about nothing else.
	valid, err := Encode(testBatch(), nil)
	require.NoError(t, err)
	t.Run("still refuses an unimplemented version", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[4] = VersionV1
		_, err := DecodeAny(bad)
		require.ErrorIs(t, err, ErrBadVersion)
	})
	t.Run("still refuses bad magic", func(t *testing.T) {
		bad := append([]byte{}, valid...)
		bad[0] = 'X'
		_, err := DecodeAny(bad)
		require.ErrorIs(t, err, ErrBadMagic)
	})
	t.Run("still refuses trailing data", func(t *testing.T) {
		_, err := DecodeAny(append(append([]byte{}, valid...), 0x00))
		require.ErrorIs(t, err, ErrTrailingData)
	})
	t.Run("still refuses a truncated header", func(t *testing.T) {
		_, err := DecodeAny(valid[:3])
		require.ErrorIs(t, err, ErrTruncated)
	})
}

// TestDecodeRejectsBadPreimage covers the one rule that makes the optional preimage field safe to
// ship before any policy uses it: content the proof did not commit to never survives decoding.
func TestDecodeRejectsBadPreimage(t *testing.T) {
	good := append(bytes.Repeat([]byte{0x42}, 20), 0x01, 0x02)
	hash, err := LogHashFromPreimage(good)
	require.NoError(t, err)

	t.Run("mismatched preimage", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs[0] = LogExport{Index: 0, Hash: hash, Preimage: append(append([]byte{}, good...), 0x03)}
		enc, err := Encode(b, nil)
		require.NoError(t, err)
		_, err = Decode(enc)
		require.ErrorIs(t, err, ErrBadPreimage)
	})
	t.Run("preimage too short to carry an address", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs[0] = LogExport{Index: 0, Hash: hash, Preimage: []byte{0x01}}
		enc, err := Encode(b, nil)
		require.NoError(t, err)
		_, err = Decode(enc)
		require.ErrorIs(t, err, ErrBadPreimage)
	})
	t.Run("matching preimage decodes", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs[0] = LogExport{Index: 0, Hash: hash, Preimage: good}
		enc, err := Encode(b, nil)
		require.NoError(t, err)
		env, err := Decode(enc)
		require.NoError(t, err)
		require.Equal(t, good, env.Batch.Blocks[0].Logs[0].Preimage)
	})
}

func TestCheckStructure(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		require.ErrorContains(t, (&ProofBatch{}).CheckStructure(), "empty")
	})
	t.Run("gap", func(t *testing.T) {
		b := testBatch()
		b.Blocks[2].Number = 104
		require.ErrorContains(t, b.CheckStructure(), "expected 102")
	})
	t.Run("timestamp does not advance", func(t *testing.T) {
		b := testBatch()
		b.Blocks[2].Timestamp = b.Blocks[1].Timestamp
		require.ErrorContains(t, b.CheckStructure(), "does not advance")
	})
	t.Run("single block with no logs", func(t *testing.T) {
		require.NoError(t, (&ProofBatch{Blocks: []BlockExport{{Number: 1, Timestamp: 2}}}).CheckStructure())
	})
	// A curated policy exports a subset, so a jump in log index is data, not a defect. Only a
	// repeat or a reversal — two hashes claiming one position — is refused.
	t.Run("sparse log indices are legal", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs = []LogExport{{Index: 2, Hash: common.HexToHash("0xaa")}, {Index: 900, Hash: common.HexToHash("0xbb")}}
		require.NoError(t, b.CheckStructure())
	})
	t.Run("repeated log index", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs[1].Index = b.Blocks[0].Logs[0].Index
		require.ErrorContains(t, b.CheckStructure(), "does not advance")
	})
	t.Run("log indices out of order", func(t *testing.T) {
		b := testBatch()
		b.Blocks[0].Logs[0].Index, b.Blocks[0].Logs[1].Index = 7, 6
		require.ErrorContains(t, b.CheckStructure(), "does not advance")
	})
}

// TestCheckStructureExecMsgs covers the two rules that make an import list a deduplicated set with
// canonical bytes, and the one restriction v3 places on what a proven chain may consume.
func TestCheckStructureExecMsgs(t *testing.T) {
	// withImports puts two imports on block 0 (timestamp 1000), sorted.
	withImports := func(msgs ...ExecMsg) *ProofBatch {
		b := testBatch()
		b.Blocks[0].ExecMsgs = msgs
		return b
	}
	// at builds an import. The argument order is the KEY's order — origin, blockNumber, logIndex,
	// timestamp, chainId, msgHash — deliberately, so that a sorted list of calls reads as a sorted
	// list of keys and a case that is meant to be in order looks like it.
	at := func(origin byte, blockNum uint64, logIdx uint32, ts uint64, chain uint64, hash byte) ExecMsg {
		return ExecMsg{Message: messages.Message{
			Identifier: messages.Identifier{
				Origin:      common.Address{origin},
				BlockNumber: blockNum,
				LogIndex:    logIdx,
				Timestamp:   ts,
				ChainID:     eth.ChainIDFromUInt64(chain),
			},
			PayloadHash: repeatHash(hash),
		}}
	}

	t.Run("sorted set is legal", func(t *testing.T) {
		require.NoError(t, withImports(
			at(0x01, 5, 0, 900, 1, 0x01),
			at(0x01, 5, 1, 900, 1, 0x02),
			at(0x02, 1, 0, 900, 2, 0x03),
		).CheckStructure())
	})
	// ORIGIN LEADS, and it outranks the coordinates. A set grouped by chain id would order these the
	// other way round, so this case is what pins the key's field order at the acceptance layer rather
	// than only in SortKey's own test.
	t.Run("origin outranks the coordinates", func(t *testing.T) {
		require.NoError(t, withImports(
			at(0x01, 999, 9, 900, 7, 0x01),
			at(0x02, 1, 0, 900, 1, 0x02),
		).CheckStructure())
		require.ErrorContains(t, withImports(
			at(0x02, 1, 0, 900, 1, 0x02),
			at(0x01, 999, 9, 900, 7, 0x01),
		).CheckStructure(), "duplicate or unsorted")
	})
	t.Run("no imports is legal", func(t *testing.T) {
		require.NoError(t, withImports().CheckStructure())
	})
	// The set is keyed by the WHOLE message: the same identifier with two different message hashes
	// is a block containing one honest reference and one wrong one, which is exactly what the judge
	// exists to catch. Deduplicating on the identifier would drop the bad one.
	t.Run("same identifier, different msgHash, both survive", func(t *testing.T) {
		require.NoError(t, withImports(at(0x01, 5, 0, 900, 1, 0x01), at(0x01, 5, 0, 900, 1, 0x02)).CheckStructure())
	})
	t.Run("exact duplicate", func(t *testing.T) {
		require.ErrorContains(t, withImports(at(0x01, 5, 0, 900, 1, 0x01), at(0x01, 5, 0, 900, 1, 0x01)).CheckStructure(),
			"duplicate or unsorted")
	})
	t.Run("out of order by origin", func(t *testing.T) {
		require.ErrorContains(t, withImports(at(0x02, 5, 0, 900, 2, 0x01), at(0x01, 5, 0, 900, 1, 0x01)).CheckStructure(),
			"duplicate or unsorted")
	})
	t.Run("out of order by log index", func(t *testing.T) {
		require.ErrorContains(t, withImports(at(0x01, 5, 9, 900, 1, 0x01), at(0x01, 5, 2, 900, 1, 0x01)).CheckStructure(),
			"duplicate or unsorted")
	})
	t.Run("out of order by msgHash only", func(t *testing.T) {
		require.ErrorContains(t, withImports(at(0x01, 5, 0, 900, 1, 0x99), at(0x01, 5, 0, 900, 1, 0x11)).CheckStructure(),
			"duplicate or unsorted")
	})
	// The same-timestamp rule is NOT a structural property, and CheckStructure must not enforce it:
	// the canonical corpus contains a VALID case (`exec-msgs-max-widths`) whose import is stamped at
	// u64::MAX to pin the field widths, so a codec refusing it would disagree with the format's own
	// definition. See TestSameTimestampImportsAreAnAcceptanceRule.
	t.Run("a same-timestamp import is STRUCTURALLY fine", func(t *testing.T) {
		require.NoError(t, withImports(at(0x01, 5, 0, 1000, 1, 0x01)).CheckStructure())
	})
	t.Run("so is one above the block", func(t *testing.T) {
		require.NoError(t, withImports(at(0x01, 5, 0, 1001, 1, 0x01)).CheckStructure())
	})
}

// TestSameTimestampImportsAreAnAcceptanceRule pins the rule at the layer that owns it (G7G D2 /
// G7R D10): a verifier refuses it, the wire format does not.
//
// The split matters in both directions. If CheckStructure enforced it, this codec would reject a
// canonically valid fixture — a byte-identity failure for a reason that has nothing to do with bytes.
// If nothing enforced it, a proven chain's same-timestamp import would reach the stock cycle graph,
// which orders executing messages by a position wire v3 deliberately does not carry.
func TestSameTimestampImportsAreAnAcceptanceRule(t *testing.T) {
	withImport := func(ts uint64) *ProofBatch {
		b := testBatch()
		b.Blocks[0].ExecMsgs = []ExecMsg{{Message: messages.Message{
			Identifier: messages.Identifier{
				Origin:      common.Address{0x01},
				BlockNumber: 5,
				Timestamp:   ts,
				ChainID:     eth.ChainIDFromUInt64(1),
			},
			PayloadHash: repeatHash(0x01),
		}}}
		return b
	}

	t.Run("equal to the consuming block", func(t *testing.T) {
		err := withImport(1000).CheckNoSameTimestampImports()
		require.ErrorIs(t, err, ErrSameTimestampImport)
		require.ErrorContains(t, err, "block 100 (timestamp 1000)")
	})
	t.Run("above the consuming block", func(t *testing.T) {
		require.ErrorIs(t, withImport(1001).CheckNoSameTimestampImports(), ErrSameTimestampImport)
	})
	t.Run("one second below is accepted", func(t *testing.T) {
		require.NoError(t, withImport(999).CheckNoSameTimestampImports())
	})
	t.Run("no imports is accepted", func(t *testing.T) {
		require.NoError(t, testBatch().CheckNoSameTimestampImports())
	})
	// Reachable from the BYTES, which is what makes it an acceptance rule rather than an invariant of
	// a struct a test happened to build.
	t.Run("reachable from a decoded envelope", func(t *testing.T) {
		enc, err := Encode(withImport(1000), nil)
		require.NoError(t, err)
		env, err := Decode(enc)
		require.NoError(t, err, "the envelope is well formed; only the node refuses it")
		require.NoError(t, env.Batch.CheckStructure())
		require.ErrorIs(t, env.Batch.CheckNoSameTimestampImports(), ErrSameTimestampImport)
	})
}

// TestExecMsgChecksumIsDerived pins the binding that makes an import list checkable: a verifier
// computes the checksum from the identifier and the message hash the proof committed to, exactly as
// the frontier view computes one from a real log. Same log, same checksum, from two different
// starting points.
func TestExecMsgChecksumIsDerived(t *testing.T) {
	log := &types.Log{
		Address: common.HexToAddress("0x00000000000000000000000000000000000c0ffee"),
		Topics:  []common.Hash{repeatHash(0x01), repeatHash(0x02)},
		Data:    []byte{0xca, 0xfe},
	}
	logHash := messages.LogToLogHash(log)
	const initBlock, initTS = uint64(77), uint64(1200)
	const initLogIdx = uint32(4)
	initChain := eth.ChainIDFromUInt64(901)

	fromLog := messages.ChecksumArgs{
		BlockNumber: initBlock,
		LogIndex:    initLogIdx,
		Timestamp:   initTS,
		ChainID:     initChain,
		LogHash:     logHash,
	}.Query()

	msg := ExecMsg{Message: messages.Message{
		Identifier: messages.Identifier{
			Origin:      log.Address,
			BlockNumber: initBlock,
			LogIndex:    initLogIdx,
			Timestamp:   initTS,
			ChainID:     initChain,
		},
		PayloadHash: crypto.Keccak256Hash(messages.LogToMessagePayload(log)),
	}}
	exec := msg.Executing()
	require.Equal(t, fromLog.Checksum, exec.Checksum, "the wire's checksum must be the log's checksum")
	require.Equal(t, initChain, exec.ChainID)
	require.Equal(t, initBlock, exec.BlockNum)
	require.Equal(t, initLogIdx, exec.LogIdx)
	require.Equal(t, initTS, exec.Timestamp)

	// A wire that named the right log with the wrong content produces a checksum that matches
	// nothing: this is why a checksum is never taken off the wire.
	tampered := msg
	tampered.PayloadHash = repeatHash(0xff)
	require.NotEqual(t, fromLog.Checksum, tampered.Executing().Checksum)
}

// TestExecMsgSortKeyIsTotal pins the canonical order's field precedence, which is the thing two
// implementations must agree on to produce the same bytes for the same set.
func TestExecMsgSortKeyIsTotal(t *testing.T) {
	base := testExecMsg()
	key := base.SortKey()
	require.Len(t, key[:], 6*32)

	bump := func(mutate func(*ExecMsg)) [execMsgKeyLen]byte {
		m := base
		mutate(&m)
		return m.SortKey()
	}
	// One 32-byte word per field, in DECLARATION order. Each field's word is pinned: changing a field
	// must move bytes inside its own word and nowhere else. That is stricter than "the key changed"
	// and is what catches a field written into the wrong word or at the wrong width.
	for _, tc := range []struct {
		name   string
		word   int
		mutate func(*ExecMsg)
	}{
		{"origin", 0, func(m *ExecMsg) {
			m.Identifier.Origin = common.HexToAddress("0x2222222222222222222222222222222222222222")
		}},
		{"blockNumber", 1, func(m *ExecMsg) { m.Identifier.BlockNumber = 43 }},
		{"logIndex", 2, func(m *ExecMsg) { m.Identifier.LogIndex = 6 }},
		{"timestamp", 3, func(m *ExecMsg) { m.Identifier.Timestamp = 81 }},
		{"chainID", 4, func(m *ExecMsg) { m.Identifier.ChainID = eth.ChainIDFromUInt64(439) }},
		{"msgHash", 5, func(m *ExecMsg) { m.PayloadHash = repeatHash(0xf2) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			other := bump(tc.mutate)
			from, to := tc.word*32, (tc.word+1)*32
			require.Equal(t, key[:from], other[:from],
				"%s changed bytes before its own word", tc.name)
			require.NotEqual(t, key[from:to], other[from:to],
				"%s did not change word %d of the key", tc.name, tc.word)
			require.Equal(t, key[to:], other[to:],
				"%s changed bytes after its own word", tc.name)
		})
	}
	require.Equal(t, 192, execMsgKeyLen, "the key width is part of the cross-language contract")

	// THE PROPERTY THE KEY EXISTS FOR: lexicographic order over the ABI words is the same relation as
	// field-by-field unsigned comparison in declaration order. That equivalence is the whole reason
	// there is only one sort rule for two languages to keep in sync, so it is asserted rather than
	// assumed.
	//
	// Origin dominates every later field: a smaller origin with MAXIMAL everything else still sorts
	// first. A key that packed the fields in any other order would fail this.
	dominated := base
	dominated.Identifier.Origin = common.HexToAddress("0x1111111111111111111111111111111111111110")
	dominated.Identifier.BlockNumber = ^uint64(0)
	dominated.Identifier.LogIndex = ^uint32(0)
	dominated.Identifier.Timestamp = ^uint64(0)
	dominated.Identifier.ChainID = eth.ChainIDFromUInt64(^uint64(0))
	dominated.PayloadHash = repeatHash(0xff)
	dominatedKey := dominated.SortKey()
	require.Negative(t, bytes.Compare(dominatedKey[:], key[:]),
		"declaration order means origin outranks every field after it")

	// ...and the transposition trap, which is why this is worth a test rather than a comment: the
	// CHECKSUM's packing swaps logIndex and timestamp relative to the identifier's declaration order
	// (op-supernode/silhouette/docs/SPEC-WIRE-V3.md §6). A key accidentally built in the checksum's order would order these two the
	// other way round, so they must not be interchangeable here.
	byLogIdx, byTS := base, base
	byLogIdx.Identifier.LogIndex, byLogIdx.Identifier.Timestamp = 9, 1
	byTS.Identifier.LogIndex, byTS.Identifier.Timestamp = 1, 9
	kL, kT := byLogIdx.SortKey(), byTS.SortKey()
	require.Positive(t, bytes.Compare(kL[:], kT[:]),
		"logIndex precedes timestamp in the identifier's declaration order; a key in the checksum's "+
			"packing order would get this backwards")
}

func TestBlobRoundTrip(t *testing.T) {
	for _, blocks := range []int{1, 300, 4000} {
		batch := testBatch()
		batch.Blocks = make([]BlockExport, blocks)
		for i := range batch.Blocks {
			batch.Blocks[i] = BlockExport{
				Number:                   uint64(1000 + i),
				Timestamp:                uint64(2000 + 2*i),
				Hash:                     common.BigToHash(big.NewInt(int64(0xb00000 + i))),
				StateRoot:                common.BigToHash(big.NewInt(int64(0x500000 + i))),
				MessagePasserStorageRoot: common.BigToHash(big.NewInt(int64(0x700000 + i))),
				Logs:                     []LogExport{{Index: uint32(i % 8), Hash: common.BigToHash(big.NewInt(int64(i)))}},
			}
		}
		enc, err := Encode(batch, make([]byte, 356))
		require.NoError(t, err)

		blobs, err := ToBlobs(enc)
		require.NoError(t, err)
		require.NotEmpty(t, blobs)

		back, err := FromBlobs(blobs)
		require.NoError(t, err)
		require.Equal(t, enc, back)

		env, err := Decode(back)
		require.NoError(t, err)
		require.Len(t, env.Batch.Blocks, blocks)
		require.Equal(t, batch.Blocks[blocks-1], env.Batch.Blocks[blocks-1])
	}
}

// TestEnvelopeSizeAtCadence keeps the v2 wire honest about the constraint that shapes it: a
// 10-minute cadence of 2s blocks must still fit in one blob.
func TestEnvelopeSizeAtCadence(t *testing.T) {
	batch := testBatch()
	batch.Blocks = make([]BlockExport, 300)
	for i := range batch.Blocks {
		batch.Blocks[i] = BlockExport{
			Number:                   uint64(1000 + i),
			Timestamp:                uint64(2000 + 2*i),
			Hash:                     common.BigToHash(big.NewInt(int64(0xb00000 + i))),
			StateRoot:                common.BigToHash(big.NewInt(int64(0x500000 + i))),
			MessagePasserStorageRoot: common.BigToHash(big.NewInt(int64(0x700000 + i))),
		}
	}
	enc, err := Encode(batch, make([]byte, 356))
	require.NoError(t, err)
	blobs, err := ToBlobs(enc)
	require.NoError(t, err)
	require.Len(t, blobs, 1, "a cadence of exports must fit one blob (%d bytes)", len(enc))
}

func TestToBlobsEmpty(t *testing.T) {
	_, err := ToBlobs(nil)
	require.Error(t, err)
}

func TestExportPolicyAllHashes(t *testing.T) {
	// Pinned: the prover computes this hash from its actual filter, so a change to the policy
	// string must be a deliberate, visible edit on both sides of the wire.
	require.Equal(t, "0xa6399dfbff2da483c5fcbe6f074297868700f9304ca957a4d61bcf84109430ef", ExportPolicyAllHashes.Hex())
}
