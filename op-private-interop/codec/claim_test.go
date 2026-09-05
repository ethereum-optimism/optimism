package codec

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func sampleClaim() *RangeClaim {
	return &RangeClaim{
		FirstBlock:               9000001,
		LastBlock:                9000300,
		PrivateTerminalBlockHash: allBytes(0xa1),
		L1Head:                   allBytes(0xb2),
		RollupConfigHash:         allBytes(0xc3),
		DepSetHash:               allBytes(0xd4),
		PrivateDataHash:          allBytes(0xe5),
	}
}

func TestRoundTrip(t *testing.T) {
	proof := make([]byte, 200)
	for i := range proof {
		proof[i] = byte(i * 7)
	}
	for _, tc := range []struct {
		name  string
		claim *RangeClaim
		mode  Mode
	}{
		{"zero value", &RangeClaim{}, ModeAttested},
		{"populated", sampleClaim(), ModeAttested},
		{"single block range", &RangeClaim{FirstBlock: 7, LastBlock: 7}, ModeAttested},
		{"max widths", &RangeClaim{
			FirstBlock: ^uint64(0) - 1, LastBlock: ^uint64(0),
			PrivateTerminalBlockHash: allBytes(0xff), L1Head: allBytes(0xff),
			RollupConfigHash: allBytes(0xff), DepSetHash: allBytes(0xff),
			PrivateDataHash: allBytes(0xff),
		}, ModeAttested},
		{"with proof", func() *RangeClaim {
			e := sampleClaim()
			e.Proof = proof
			return e
		}(), ModeProven},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := Encode(tc.claim)
			require.NoError(t, err)

			got, err := DecodeMode(data, tc.mode)
			require.NoError(t, err)
			require.Equal(t, tc.claim, got, "decoded value must equal the encoded one")

			again, err := Encode(got)
			require.NoError(t, err)
			require.Equal(t, data, again, "the format has exactly one encoding of a given claim")
		})
	}
}

// TestEncodedSizeIsIndependentOfRange is the headline property of v1 and the whole reason the
// claim can be an ordinary L2 transaction: it carries no per-block data, so a range of one block
// and a range of a hundred million cost the same 352 bytes.
func TestEncodedSizeIsIndependentOfRange(t *testing.T) {
	for _, blocks := range []uint64{1, 300, 1_000_000, ^uint64(0) - 1} {
		e := sampleClaim()
		e.FirstBlock = 1
		e.LastBlock = blocks
		data, err := Encode(e)
		require.NoError(t, err)
		require.Len(t, data, EncodedSizeEmptyProof, "range of %d blocks", blocks)
	}
}

// TestABIWordLayout pins the encoding word by word against a HAND-WRITTEN table.
//
// This is the test that catches a change no round-trip can see. Encode and Decode agree with each
// other by construction; they can agree with each other and disagree with Solidity, with alloy, or
// with the previous release. The words below were written out by hand from the ABI rules and are
// the thing a second implementation is really being asked to match.
func TestABIWordLayout(t *testing.T) {
	e := &RangeClaim{
		FirstBlock:                0x0102030405060708,
		LastBlock:                 0x1112131415161718,
		PrivateTerminalBlockHash:  allBytes(0x22),
		PrivateTerminalParentHash: allBytes(0x77),
		L1Head:                    allBytes(0x33),
		RollupConfigHash:          allBytes(0x44),
		DepSetHash:                allBytes(0x55),
		PrivateDataHash:           allBytes(0x66),
		Proof:                     []byte{0xaa, 0xbb, 0xcc},
	}
	data, err := Encode(e)
	require.NoError(t, err)

	want := []string{
		// word 0 — abi.encode of a DYNAMIC struct leads with the offset to the tuple. A static
		// struct would encode in place with no such word; `proof` is what makes this one dynamic.
		"0000000000000000000000000000000000000000000000000000000000000020",
		// word 1 — version, uint8 left-padded into a full word.
		"0000000000000000000000000000000000000000000000000000000000000001",
		// word 2 — firstBlock, uint64 left-padded, big-endian.
		"0000000000000000000000000000000000000000000000000102030405060708",
		// word 3 — lastBlock.
		"0000000000000000000000000000000000000000000000001112131415161718",
		// word 4 — privateTerminalBlockHash, a bytes32, in place and unpadded.
		"2222222222222222222222222222222222222222222222222222222222222222",
		// word 5 — privateTerminalParentHash, the field that completes the public follow ref.
		"7777777777777777777777777777777777777777777777777777777777777777",
		// word 6 — l1Head.
		"3333333333333333333333333333333333333333333333333333333333333333",
		// word 7 — rollupConfigHash.
		"4444444444444444444444444444444444444444444444444444444444444444",
		// word 8 — depSetHash.
		"5555555555555555555555555555555555555555555555555555555555555555",
		// word 9 — privateDataHash.
		"6666666666666666666666666666666666666666666666666666666666666666",
		// word 10 — offset to `proof`, relative to the START OF THE TUPLE (word 1), not to the start
		// of the encoding: ten head words = 320 = 0x140. Getting this relative-to-what wrong by one
		// word is the classic hand-rolled-encoder bug, which is why the fixture corpus also carries
		// a non-minimal-offset refusal case.
		"0000000000000000000000000000000000000000000000000000000000000140",
		// word 11 — proof length, 3.
		"0000000000000000000000000000000000000000000000000000000000000003",
		// word 12 — proof bytes, LEFT-aligned and right-padded (dynamic bytes pad on the right,
		// the opposite of every integer above).
		"aabbcc0000000000000000000000000000000000000000000000000000000000",
	}
	require.Len(t, data, len(want)*32, "encoding is not a whole number of the expected words")
	for i, w := range want {
		require.Equal(t, w, fmt.Sprintf("%x", data[i*32:(i+1)*32]), "word %d", i)
	}

	// And the table is not just a snapshot of the encoder: it must decode back to the value.
	got, err := DecodeMode(data, ModeProven)
	require.NoError(t, err)
	require.Equal(t, e, got)
}

// TestAttestedModeRefusesProof is the standing v1 rule, stated as a test: a verifier that accepts
// what it cannot check has a hole exactly the size of the thing it skipped.
func TestAttestedModeRefusesProof(t *testing.T) {
	e := sampleClaim()
	e.Proof = []byte{0x01}
	data, err := Encode(e)
	require.NoError(t, err)

	_, err = Decode(data)
	require.ErrorIs(t, err, ErrProofNotEmpty)
	// A one-byte proof is refused for the same reason a megabyte one would be: the rule is about
	// the slot being non-empty, not about the proof being plausible.
	require.Contains(t, err.Error(), "1 bytes")

	// The default is the strict one: ModeAttested is Mode's zero value, so a caller who never
	// thought about proof posture gets the refusal rather than the hole.
	var unset Mode
	require.Equal(t, ModeAttested, unset)
	_, err = DecodeMode(data, unset)
	require.ErrorIs(t, err, ErrProofNotEmpty)

	proven, err := DecodeMode(data, ModeProven)
	require.NoError(t, err)
	require.Equal(t, []byte{0x01}, proven.Proof)
}

// TestTruncationAtEveryBoundary refuses every prefix of a valid encoding. Exhaustive rather than
// sampled: "truncated at a field boundary" and "truncated mid-word" are the same class of bug to a
// decoder that does its length arithmetic properly and different classes to one that does not, and
// there is no reason to guess which boundaries matter.
func TestTruncationAtEveryBoundary(t *testing.T) {
	e := sampleClaim()
	e.Proof = []byte{1, 2, 3, 4, 5}
	data, err := Encode(e)
	require.NoError(t, err)
	require.Greater(t, len(data), EncodedSizeEmptyProof)

	for n := 0; n < len(data); n++ {
		_, err := DecodeMode(data[:n], ModeProven)
		require.Errorf(t, err, "a %d-byte prefix of a %d-byte claim decoded", n, len(data))
	}
	// The full thing still decodes — otherwise the loop above proves nothing.
	_, err = DecodeMode(data, ModeProven)
	require.NoError(t, err)
}

// TestTrailingBytesAtEveryLength appends junk of every length up to two words. A decoder that
// enforced "no trailing data" by comparing against a multiple of 32 would pass at 32 and 64 and
// fail everywhere else; a decoder that only checked the total length would pass the padded cases.
func TestTrailingBytesAtEveryLength(t *testing.T) {
	data, err := Encode(sampleClaim())
	require.NoError(t, err)
	for n := 1; n <= 64; n++ {
		_, err := DecodeMode(append(append([]byte(nil), data...), make([]byte, n)...), ModeProven)
		require.ErrorIsf(t, err, ErrNonCanonical, "%d trailing bytes accepted", n)
	}
}

func TestVersionGate(t *testing.T) {
	for v := 0; v <= 255; v++ {
		version := uint8(v)
		if version == ClaimVersion {
			continue
		}
		data, err := encodeAtVersion(sampleClaim(), version)
		require.NoError(t, err)
		_, err = DecodeMode(data, ModeProven)
		require.ErrorIsf(t, err, ErrBadVersion, "version %d was accepted", version)
	}
	data, err := encodeAtVersion(sampleClaim(), ClaimVersion)
	require.NoError(t, err)
	_, err = DecodeMode(data, ModeProven)
	require.NoError(t, err)
}

func TestInvertedRangeRefusedByEncodeAndDecode(t *testing.T) {
	e := sampleClaim()
	e.FirstBlock, e.LastBlock = 300, 299

	_, err := Encode(e)
	require.ErrorIs(t, err, ErrInvertedRange, "a producer must not be able to emit what its decoder refuses")

	data, err := encodeAtVersion(e, ClaimVersion)
	require.NoError(t, err)
	_, err = DecodeMode(data, ModeProven)
	require.ErrorIs(t, err, ErrInvertedRange)

	// The boundary: equal is legal and means one block.
	ok := sampleClaim()
	ok.FirstBlock, ok.LastBlock = 300, 300
	require.NoError(t, ok.CheckStructure())
	require.Equal(t, uint64(1), ok.Blocks())
	require.Equal(t, uint64(0), (&RangeClaim{FirstBlock: 2, LastBlock: 1}).Blocks())
	require.Equal(t, uint64(300), sampleClaim().Blocks())
}

// TestProofSizeCap walks the boundary from both sides and through both doors. The interesting
// value is MaxProofSize+1: one byte over, everything else about the object perfect.
func TestProofSizeCap(t *testing.T) {
	atCap := sampleClaim()
	atCap.Proof = make([]byte, MaxProofSize)
	data, err := Encode(atCap)
	require.NoError(t, err, "exactly at the cap is legal")
	require.Len(t, data, MaxEncodedSize, "the cap and the size bound must be the same fact")
	got, err := DecodeMode(data, ModeProven)
	require.NoError(t, err)
	require.Equal(t, atCap, got)

	over := sampleClaim()
	over.Proof = make([]byte, MaxProofSize+1)
	_, err = Encode(over)
	require.ErrorIs(t, err, ErrProofTooLarge, "a producer must not be able to emit what its decoder refuses")

	// encodeAtVersion applies no policy, so it is how the over-cap bytes get built at all.
	overBytes, err := encodeAtVersion(over, ClaimVersion)
	require.NoError(t, err)
	require.Greater(t, len(overBytes), MaxEncodedSize)
	_, err = DecodeMode(overBytes, ModeProven)
	require.ErrorIs(t, err, ErrProofTooLarge)
	_, err = Decode(overBytes)
	require.ErrorIs(t, err, ErrProofTooLarge)

	// The length guard fires before anything is unpacked, so garbage of an impossible length is
	// refused just as cheaply as a well-formed over-cap claim.
	_, err = DecodeMode(make([]byte, MaxEncodedSize+1), ModeProven)
	require.ErrorIs(t, err, ErrProofTooLarge)
}

// TestSolidityProducesTheSameBytes pins this codec against solc's own abi.encode.
//
// The two hex strings below are the Go encoder's output for the two vectors above. They are in this
// test because a Solidity test asserted the SAME bytes: testdata/solidity/RangeClaimVector.t.sol
// encodes the identical struct values with solc's abi.encode and requires equality, and it passed
// under solc 0.8.30 / forge 1.4.4. That file is self-contained (no forge-std), so the claim is
// reproducible rather than merely asserted here.
//
// This is the check the fixture corpus cannot make on its own. The corpus proves this package is
// self-consistent and stable; only the other compiler proves it is speaking ABI rather than a
// convincing dialect of it. And that matters more here than it would with an event to read: the
// registry is log-less, so the durable record is the CALLDATA this encoder produces, decoded by
// whoever scans it. Producer and reader meet on these bytes and nothing in between re-encodes them.
func TestSolidityProducesTheSameBytes(t *testing.T) {
	const solidityEmptyProof = "0x" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000001" +
		"0000000000000000000000000000000000000000000000000000000000000001" +
		"000000000000000000000000000000000000000000000000000000000000012c" +
		"0000000000000000000000000000000000000000000000000000000000000001" +
		"0000000000000000000000000000000000000000000000000000000000000006" +
		"0000000000000000000000000000000000000000000000000000000000000002" +
		"0000000000000000000000000000000000000000000000000000000000000003" +
		"0000000000000000000000000000000000000000000000000000000000000004" +
		"0000000000000000000000000000000000000000000000000000000000000005" +
		"0000000000000000000000000000000000000000000000000000000000000140" +
		"0000000000000000000000000000000000000000000000000000000000000000"
	const solidityWithProof = "0x" +
		"0000000000000000000000000000000000000000000000000000000000000020" +
		"0000000000000000000000000000000000000000000000000000000000000001" +
		"0000000000000000000000000000000000000000000000000102030405060708" +
		"0000000000000000000000000000000000000000000000001112131415161718" +
		"2222222222222222222222222222222222222222222222222222222222222222" +
		"7777777777777777777777777777777777777777777777777777777777777777" +
		"3333333333333333333333333333333333333333333333333333333333333333" +
		"4444444444444444444444444444444444444444444444444444444444444444" +
		"5555555555555555555555555555555555555555555555555555555555555555" +
		"6666666666666666666666666666666666666666666666666666666666666666" +
		"0000000000000000000000000000000000000000000000000000000000000140" +
		"0000000000000000000000000000000000000000000000000000000000000003" +
		"aabbcc0000000000000000000000000000000000000000000000000000000000"

	empty, err := Encode(&RangeClaim{
		FirstBlock:                1,
		LastBlock:                 300,
		PrivateTerminalBlockHash:  common.HexToHash("0x01"),
		PrivateTerminalParentHash: common.HexToHash("0x06"),
		L1Head:                    common.HexToHash("0x02"),
		RollupConfigHash:          common.HexToHash("0x03"),
		DepSetHash:                common.HexToHash("0x04"),
		PrivateDataHash:           common.HexToHash("0x05"),
	})
	require.NoError(t, err)
	require.Equal(t, solidityEmptyProof, hexutil.Encode(empty))

	withProof, err := Encode(&RangeClaim{
		FirstBlock:                0x0102030405060708,
		LastBlock:                 0x1112131415161718,
		PrivateTerminalBlockHash:  allBytes(0x22),
		PrivateTerminalParentHash: allBytes(0x77),
		L1Head:                    allBytes(0x33),
		RollupConfigHash:          allBytes(0x44),
		DepSetHash:                allBytes(0x55),
		PrivateDataHash:           allBytes(0x66),
		Proof:                     []byte{0xaa, 0xbb, 0xcc},
	})
	require.NoError(t, err)
	require.Equal(t, solidityWithProof, hexutil.Encode(withProof))

	// And the strict decoder accepts what a registry call would carry — solc's own encoding of the
	// struct, which is what a reader scanning postClaim calldata has to make sense of.
	decoded, err := DecodeMode(hexutil.MustDecode(solidityWithProof), ModeProven)
	require.NoError(t, err)
	require.Equal(t, []byte{0xaa, 0xbb, 0xcc}, decoded.Proof)
	_, err = Decode(hexutil.MustDecode(solidityEmptyProof))
	require.NoError(t, err)
}

func TestDecodeModeString(t *testing.T) {
	require.Equal(t, "attested", ModeAttested.String())
	require.Equal(t, "proven", ModeProven.String())
	require.Equal(t, "mode(9)", Mode(9).String())
}
