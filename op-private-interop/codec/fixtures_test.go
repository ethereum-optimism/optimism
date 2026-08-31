package codec

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

// The range-claim wire is pinned by FIXTURES rather than by this implementation. The bytes are
// the contract: a second implementation — the batching service's producer, the claim registry
// predeploy, a permissioned replica's adapter, an auditor's tool, or the proof program that will
// one day fill the proof slot — is checked against this corpus, not against this package's opinion
// of itself. A test that only round-trips encode(decode(x)) passes just as happily when both halves
// are wrong in the same direction.
//
// Regenerate with `go test ./op-private-interop/codec -run TestFixtures -update`. Treat any
// resulting diff in a `valid` case's bytes as a WIRE CHANGE needing a version bump, not as a test
// fixture that drifted.
const fixtureDir = "testdata/range-claim-v1"

var updateFixtures = flag.Bool("update", false, "regenerate the range-claim fixture corpus")

type fixtureIndex struct {
	Spec                   string   `json:"spec"`
	Version                uint8    `json:"version"`
	Transport              string   `json:"transport"`
	PayloadEncoding        string   `json:"payloadEncoding"`
	PayloadFields          []string `json:"payloadFields"`
	PayloadSolidityTypes   []string `json:"payloadSolidityTypes"`
	EncodedBytesEmptyProof int      `json:"encodedBytesEmptyProof"`
	MaxProofBytes          int      `json:"maxProofBytes"`
	MaxEncodedBytes        int      `json:"maxEncodedBytes"`
	Registry               string   `json:"registry"`
	// RefusedVersions is []uint16 rather than []uint8 for one dull but load-bearing reason:
	// encoding/json renders a []byte as a base64 STRING, which would put "AAL/" in a corpus meant
	// to be read by implementations in other languages.
	RefusedVersions []uint16      `json:"refusedVersions"`
	Cases           []fixtureCase `json:"cases"`
}

type fixtureCase struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Bin  string `json:"bin"`
	JSON string `json:"json"`
}

// fixtureBody is the claim's fields as JSON, one key per ABI component name.
type fixtureBody struct {
	Version                   uint8       `json:"version"`
	FirstBlock                uint64      `json:"firstBlock"`
	LastBlock                 uint64      `json:"lastBlock"`
	PrivateTerminalBlockHash  common.Hash `json:"privateTerminalBlockHash"`
	PrivateTerminalParentHash common.Hash `json:"privateTerminalParentHash"`
	L1Head                    common.Hash `json:"l1Head"`
	RollupConfigHash          common.Hash `json:"rollupConfigHash"`
	DepSetHash                common.Hash `json:"depSetHash"`
	PrivateDataHash           common.Hash `json:"privateDataHash"`
}

func bodyFixture(e *RangeClaim, version uint8) fixtureBody {
	return fixtureBody{
		Version:                   version,
		FirstBlock:                e.FirstBlock,
		LastBlock:                 e.LastBlock,
		PrivateTerminalBlockHash:  e.PrivateTerminalBlockHash,
		PrivateTerminalParentHash: e.PrivateTerminalParentHash,
		L1Head:                    e.L1Head,
		RollupConfigHash:          e.RollupConfigHash,
		DepSetHash:                e.DepSetHash,
		PrivateDataHash:           e.PrivateDataHash,
	}
}

// fixtureFile is one case's JSON sidecar: what the bytes are, what they decode to, and — for the
// invalid cases — what a decoder is expected to refuse them for.
type fixtureFile struct {
	Name        string `json:"name"`
	Spec        string `json:"spec"`
	Kind        string `json:"kind"`
	Description string `json:"description"`
	Bin         string `json:"bin"`
	Bytes       int    `json:"bytes"`
	// Keccak over the encoded bytes is the cross-implementation checksum for the whole case:
	// agreeing on it means agreeing on every byte, padding included.
	Keccak   common.Hash   `json:"keccak"`
	ProofHex hexutil.Bytes `json:"proofHex"`
	Body     *fixtureBody  `json:"body,omitempty"`
	// Refusal is the CATEGORY an invalid case must be refused under, from the small closed set
	// below. It is a category rather than an error string on purpose: the corpus is meant to be
	// read by implementations in other languages, and pinning one library's wording would make a
	// cross-language gate fail for reasons that have nothing to do with the bytes.
	Refusal string `json:"refusal,omitempty"`

	// bytes is the case's content; it is written to Bin rather than into the JSON.
	bytes []byte
}

const (
	kindValid = "valid"
	// kindValidProven is a well-formed claim with a NON-EMPTY proof slot: valid bytes, and
	// refused by an attested-mode verifier. It is its own kind because "the bytes are fine and this
	// verifier still says no" is the entire point of the attested rule.
	kindValidProven = "valid-proven"
	kindInvalid     = "invalid"
)

// The refusal categories. Every invalid fixture names exactly one.
const (
	refusalTruncated = "truncated"
	refusalVersion   = "bad-version"
	refusalRange     = "inverted-range"
	// refusalNonCanonical covers every way bytes can fail to be THE encoding of a claim while
	// still being A decoding of one: trailing data, dirty bits above a narrow integer, a
	// non-minimal offset to the dynamic member. Which layer catches which is an implementation
	// detail — go-ethereum's ABI decoder rejects the dirty-integer case itself, and this package's
	// re-encode comparison catches the rest — so the contract the corpus states is only that the
	// bytes are refused.
	refusalNonCanonical = "non-canonical"
	refusalProofSize    = "proof-too-large"
)

// assertRefusal checks that err is the refusal the fixture's category calls for.
func assertRefusal(t *testing.T, category string, err error) {
	t.Helper()
	require.Error(t, err)
	switch category {
	case refusalTruncated:
		require.ErrorIs(t, err, ErrTruncated)
	case refusalVersion:
		require.ErrorIs(t, err, ErrBadVersion)
	case refusalRange:
		require.ErrorIs(t, err, ErrInvertedRange)
	case refusalProofSize:
		require.ErrorIs(t, err, ErrProofTooLarge)
	case refusalNonCanonical:
		// Refused, by whichever layer notices first. See the constant.
	default:
		t.Fatalf("unknown refusal category %q", category)
	}
}

func fixtureHash(parts ...string) common.Hash {
	s := "private-interop-fixture"
	for _, p := range parts {
		s += ":" + p
	}
	return crypto.Keccak256Hash([]byte(s))
}

func allBytes(v byte) common.Hash {
	var h common.Hash
	for i := range h {
		h[i] = v
	}
	return h
}

// buildFixtures is the corpus recipe. Every scalar is keccak256 of a printable label, so the corpus
// is reproducible from this function alone, in any language.
func buildFixtures(t *testing.T) []fixtureFile {
	t.Helper()

	// range-0 is the first claim a chain ever emits. It is an ordinary case, not an edge one: a
	// claim leads the range it describes, so range 0 opens with range 0's claim exactly like every
	// range after it. There is no parent field and no zero-hash special case anywhere in the wire.
	range0 := &RangeClaim{
		FirstBlock:                1,
		LastBlock:                 300,
		PrivateTerminalBlockHash:  fixtureHash("range-0", "privateTerminalBlockHash"),
		PrivateTerminalParentHash: fixtureHash("range-0", "privateTerminalParentHash"),
		L1Head:                    fixtureHash("range-0", "l1Head"),
		RollupConfigHash:          fixtureHash("range-0", "rollupConfigHash"),
		DepSetHash:                fixtureHash("range-0", "depSetHash"),
		PrivateDataHash:           fixtureHash("range-0", "privateDataHash"),
	}

	// The operating point: one 10-minute cadence at 2 s blocks, mid-chain.
	cadence := &RangeClaim{
		FirstBlock:                9000001,
		LastBlock:                 9000300,
		PrivateTerminalBlockHash:  fixtureHash("cadence-300-blocks", "privateTerminalBlockHash"),
		PrivateTerminalParentHash: fixtureHash("cadence-300-blocks", "privateTerminalParentHash"),
		L1Head:                    fixtureHash("cadence-300-blocks", "l1Head"),
		RollupConfigHash:          fixtureHash("cadence-300-blocks", "rollupConfigHash"),
		DepSetHash:                fixtureHash("cadence-300-blocks", "depSetHash"),
		PrivateDataHash:           fixtureHash("cadence-300-blocks", "privateDataHash"),
	}

	// A single-block range: firstBlock == lastBlock is legal and covers one block, not zero.
	singleBlock := &RangeClaim{
		FirstBlock:                42,
		LastBlock:                 42,
		PrivateTerminalBlockHash:  fixtureHash("single-block-range", "privateTerminalBlockHash"),
		PrivateTerminalParentHash: fixtureHash("single-block-range", "privateTerminalParentHash"),
		L1Head:                    fixtureHash("single-block-range", "l1Head"),
		RollupConfigHash:          fixtureHash("single-block-range", "rollupConfigHash"),
		DepSetHash:                fixtureHash("single-block-range", "depSetHash"),
		PrivateDataHash:           fixtureHash("single-block-range", "privateDataHash"),
	}

	maxWidths := &RangeClaim{
		FirstBlock:                ^uint64(0) - 1,
		LastBlock:                 ^uint64(0),
		PrivateTerminalBlockHash:  allBytes(0xff),
		PrivateTerminalParentHash: allBytes(0xff),
		L1Head:                    allBytes(0xff),
		RollupConfigHash:          allBytes(0xff),
		DepSetHash:                allBytes(0xff),
		PrivateDataHash:           allBytes(0xff),
	}

	// 65 bytes: deliberately NOT a multiple of 32, so the fixture pins the ABI tail padding that a
	// hand-rolled encoder is most likely to get wrong.
	proof := make([]byte, 65)
	for i := range proof {
		proof[i] = byte(i)
	}
	proven := &RangeClaim{
		FirstBlock:                9000301,
		LastBlock:                 9000600,
		PrivateTerminalBlockHash:  fixtureHash("proven-slot", "privateTerminalBlockHash"),
		PrivateTerminalParentHash: fixtureHash("proven-slot", "privateTerminalParentHash"),
		L1Head:                    fixtureHash("proven-slot", "l1Head"),
		RollupConfigHash:          fixtureHash("proven-slot", "rollupConfigHash"),
		DepSetHash:                fixtureHash("proven-slot", "depSetHash"),
		PrivateDataHash:           fixtureHash("proven-slot", "privateDataHash"),
		Proof:                     proof,
	}

	valid := []struct {
		name  string
		kind  string
		desc  string
		claim *RangeClaim
	}{
		{
			name: "range-0", kind: kindValid, claim: range0,
			desc: "The first claim a chain ever emits: it describes range 0 (blocks 1..300) and is the " +
				"leading user transaction of block 1. Present to show there is NO genesis edge — a " +
				"claim leads the range it describes, so the first range opens exactly like every " +
				"later one, with no parent field and no zero-hash special case to handle.",
		},
		{
			name: "cadence-300-blocks", kind: kindValid, claim: cadence,
			desc: "The operating point: one 10-minute cadence at 2 s block time, 300 public blocks, " +
				"mid-chain. Note it is the same 352 bytes as every other empty-proof claim — v1 " +
				"carries no per-block data, so claim size is independent of range size.",
		},
		{
			name: "single-block-range", kind: kindValid, claim: singleBlock,
			desc: "firstBlock == lastBlock. A legal, one-block range: the range bound is inclusive on " +
				"both ends, so this covers one block and not zero. Pins the boundary that separates " +
				"the valid degenerate case from the inverted-range refusal.",
		},
		{
			name: "max-widths", kind: kindValid, claim: maxWidths,
			desc: "Every field at its maximum: 0xff.. hashes, and a block range at the top of uint64. " +
				"Pins the ABI field WIDTHS — a decoder that read firstBlock as uint32, or that " +
				"sign-extended it, disagrees here and nowhere else.",
		},
		{
			name: "proven-slot", kind: kindValidProven, claim: proven,
			desc: "Well-formed bytes carrying a 65-byte proof slot (the bytes 0x00..0x40, a placeholder " +
				"for a real proof; 65 is deliberately not a multiple of 32, so this also pins the ABI " +
				"tail padding). ModeProven decodes it; ModeAttested REFUSES it, which is the standing " +
				"v1 rule: a verifier with no proof system must not accept a claim whose central " +
				"claim it is not equipped to evaluate.",
		},
	}

	var out []fixtureFile
	for _, c := range valid {
		data, err := Encode(c.claim)
		require.NoError(t, err)
		body := bodyFixture(c.claim, ClaimVersion)
		proofHex := c.claim.Proof
		if proofHex == nil {
			proofHex = []byte{}
		}
		out = append(out, fixtureFile{
			Name: c.name, Kind: c.kind, Description: c.desc,
			Bin: c.name + ".bin", Bytes: len(data),
			Keccak:   crypto.Keccak256Hash(data),
			ProofHex: proofHex,
			Body:     &body,
			bytes:    data,
		})
	}

	// Wrong-version cases are REAL encodings at the wrong version — the bytes a mis-versioned
	// producer would genuinely emit, right down to the padding — rather than a valid encoding with
	// a word poked. Poking tests the gate; encoding tests the gate against a plausible producer.
	for _, v := range []struct {
		version uint8
		desc    string
	}{
		{0x00, "Version 0: the zero value, which is what an uninitialised producer emits and what a " +
			"decoder treating zero as 'unset, assume current' would wave through. Refused."},
		{0x02, "Version 2: the next version, which does not exist yet. Forwards-leniency is how a " +
			"future field addition gets silently misread out of the wrong offsets by something that " +
			"reports success, so an unrecognised successor is refused as firmly as anything else."},
		{0xff, "Version 255: the top of the field. Refused like any other unrecognised version; " +
			"present so the gate is pinned at both ends of uint8 rather than only near 1."},
	} {
		data, err := encodeAtVersion(range0, v.version)
		require.NoError(t, err)
		body := bodyFixture(range0, v.version)
		out = append(out, fixtureFile{
			Name: versionCaseName(v.version), Kind: kindInvalid, Description: v.desc,
			Bin: versionCaseName(v.version) + ".bin", Bytes: len(data),
			Keccak:   crypto.Keccak256Hash(data),
			ProofHex: []byte{},
			Body:     &body,
			Refusal:  refusalVersion,
			bytes:    data,
		})
	}

	// The remaining invalid cases are mutations of `range-0`, so a reader can diff each against it
	// and see exactly one thing wrong.
	range0Bytes, err := Encode(range0)
	require.NoError(t, err)
	mutate := func(f func(b []byte) []byte) []byte { return f(append([]byte(nil), range0Bytes...)) }

	invalid := []struct {
		name    string
		desc    string
		refusal string
		data    []byte
	}{
		{
			name: "inverted-range",
			desc: "lastBlock (299) is below firstBlock (300): a range that reads backwards. Refused at " +
				"DECODE, not merely by an optional structural check, and refused by Encode too, so " +
				"there is no path that produces it and no consumer that has to remember to look.",
			refusal: refusalRange,
			data: mustEncode(t, &RangeClaim{
				FirstBlock:               300,
				LastBlock:                299,
				PrivateTerminalBlockHash: range0.PrivateTerminalBlockHash,
				L1Head:                   range0.L1Head,
				RollupConfigHash:         range0.RollupConfigHash,
				DepSetHash:               range0.DepSetHash,
				PrivateDataHash:          range0.PrivateDataHash,
			}),
		},
		{
			name: "trailing-bytes",
			desc: "A complete, otherwise-valid encoding with four extra bytes after it. The ABI decoder " +
				"alone would ignore them; canonical form refuses them. Bytes that decode at all must " +
				"decode to EXACTLY one claim, or a transaction carries a second object that only " +
				"some readers see.",
			refusal: refusalNonCanonical,
			data:    mutate(func(b []byte) []byte { return append(b, 0xde, 0xad, 0xbe, 0xef) }),
		},
		{
			name: "dirty-version-word",
			desc: "The version word is 0x...0101 instead of 0x...01: correct in its low byte, junk in " +
				"the bits above a uint8. Refused, because two readers agreeing on the VALUE while " +
				"disagreeing about the BYTES is how one transaction becomes two facts. A decoder that " +
				"masked to the low byte would read this as a perfectly good version 1. (go-ethereum " +
				"happens to reject it inside abi.Unpack rather than at the re-encode comparison, which " +
				"is why the corpus states a refusal CATEGORY and not an error string.)",
			refusal: refusalNonCanonical,
			data: mutate(func(b []byte) []byte {
				// Word 1 is the tuple's first head word (word 0 is the outer offset).
				b[32+30] = 0x01
				return b
			}),
		},
		{
			name: "non-minimal-proof-offset",
			desc: "The head's offset to `proof` points 32 bytes past the canonical position, with the " +
				"length word duplicated there so a lenient decoder still finds an empty proof and " +
				"reports success. Two encodings of one value is one encoding too many when the value " +
				"is going to be hashed; canonical form refuses it.",
			refusal: refusalNonCanonical,
			data: mutate(func(b []byte) []byte {
				// The tuple starts at byte 32; its ninth head word (bytes 288..319) is the offset to
				// `proof`, canonically 0x140. Push it one word further and append the room it now
				// claims, so a lenient decoder finds a zero length word there and succeeds.
				moved := uint16(proofOffset + 32)
				b[32+(headWords-1)*32+30] = byte(moved >> 8)
				b[32+(headWords-1)*32+31] = byte(moved)
				return append(b, make([]byte, 32)...)
			}),
		},
		{
			name: "proof-too-large",
			desc: "A well-formed claim whose proof slot is MaxProofSize + 1 bytes — the smallest " +
				"violation there is. Everything about it is canonical; the only thing wrong is that it " +
				"is too big, which is why the case has to carry the real bytes rather than an " +
				"over-declared length. The proof is all zeroes so the fixture compresses to nothing. " +
				"Refused by Encode as well as Decode: a producer that can emit what its own decoder " +
				"rejects will do it at the worst possible time.",
			refusal: refusalProofSize,
			data: mustEncode(t, &RangeClaim{
				FirstBlock:               range0.FirstBlock,
				LastBlock:                range0.LastBlock,
				PrivateTerminalBlockHash: range0.PrivateTerminalBlockHash,
				L1Head:                   range0.L1Head,
				RollupConfigHash:         range0.RollupConfigHash,
				DepSetHash:               range0.DepSetHash,
				PrivateDataHash:          range0.PrivateDataHash,
				Proof:                    make([]byte, MaxProofSize+1),
			}),
		},
		{
			name: "truncated-one-word",
			desc: "A valid encoding with its last 32-byte word removed — the proof's length word. The " +
				"shortest thing that is still shaped like a claim and is not one.",
			refusal: refusalTruncated,
			data:    range0Bytes[:len(range0Bytes)-32],
		},
		{
			name: "empty",
			desc: "Zero bytes. The degenerate input, present because an empty calldata argument and an " +
				"absent one are easy to conflate at the call site.",
			refusal: refusalTruncated,
			data:    []byte{},
		},
	}

	for _, c := range invalid {
		out = append(out, fixtureFile{
			Name: c.name, Kind: kindInvalid, Description: c.desc,
			Bin: c.name + ".bin", Bytes: len(c.data),
			Keccak:   crypto.Keccak256Hash(c.data),
			ProofHex: []byte{},
			Refusal:  c.refusal,
			bytes:    c.data,
		})
	}
	return out
}

func versionCaseName(v uint8) string {
	return "version-" + hexutil.Encode([]byte{v})[2:]
}

func mustEncode(t *testing.T, e *RangeClaim) []byte {
	t.Helper()
	// Encode refuses an inverted range, which is the point of that case — build its bytes through
	// the version-parameterised encoder, which applies no structural rule.
	data, err := encodeAtVersion(e, ClaimVersion)
	require.NoError(t, err)
	return data
}

// TestFixtures checks this codec against the stored corpus, and regenerates it under -update.
func TestFixtures(t *testing.T) {
	cases := buildFixtures(t)
	if *updateFixtures {
		writeFixtures(t, cases)
	}

	var index fixtureIndex
	readJSON(t, filepath.Join(fixtureDir, "index.json"), &index)
	require.Equal(t, ClaimVersion, index.Version)
	require.Equal(t, EncodedSizeEmptyProof, index.EncodedBytesEmptyProof)
	require.Equal(t, MaxProofSize, index.MaxProofBytes)
	require.Equal(t, MaxEncodedSize, index.MaxEncodedBytes)
	require.Len(t, index.Cases, len(cases), "the index case list is out of step with the recipe")

	for i, want := range cases {
		t.Run(want.Name, func(t *testing.T) {
			require.Equal(t, want.Name, index.Cases[i].Name, "index case order must match the recipe")
			require.Equal(t, want.Kind, index.Cases[i].Kind)

			var stored fixtureFile
			readJSON(t, filepath.Join(fixtureDir, want.Name+".json"), &stored)
			data, err := os.ReadFile(filepath.Join(fixtureDir, want.Bin))
			require.NoError(t, err)

			// The bytes on disk are the contract. They are checked against the RECIPE, not against
			// whatever this code happens to produce today.
			require.Equal(t, want.bytes, data, "stored bytes differ from the recipe")
			require.Equal(t, len(data), stored.Bytes)
			require.Equal(t, crypto.Keccak256Hash(data), stored.Keccak)
			require.Equal(t, want.Kind, stored.Kind)

			switch stored.Kind {
			case kindValid, kindValidProven:
				mode := ModeAttested
				if stored.Kind == kindValidProven {
					mode = ModeProven
				}
				claim, err := DecodeMode(data, mode)
				require.NoError(t, err)
				require.Equal(t, *stored.Body, bodyFixture(claim, ClaimVersion), "decoded body differs from the fixture")
				require.Equal(t, []byte(stored.ProofHex), []byte(sliceOrEmpty(claim.Proof)))
				require.NoError(t, claim.CheckStructure())
				require.Positive(t, claim.Blocks())

				// Re-encoding must reproduce the file byte for byte: the format has exactly one
				// encoding of a given claim, which is what lets a hash of it mean anything.
				again, err := Encode(claim)
				require.NoError(t, err)
				require.Equal(t, data, again)

				if stored.Kind == kindValidProven {
					require.NotEmpty(t, claim.Proof, "the proven-slot case must actually carry a proof")
					_, err := DecodeMode(data, ModeAttested)
					require.ErrorIs(t, err, ErrProofNotEmpty,
						"an attested verifier must refuse a filled proof slot it cannot check")
				} else {
					require.Empty(t, claim.Proof)
				}
			case kindInvalid:
				// Proven mode is the LENIENT one; if even it refuses, every configuration does.
				_, err := DecodeMode(data, ModeProven)
				require.Error(t, err, "invalid fixture decoded: %s", stored.Description)
				assertRefusal(t, stored.Refusal, err)
				_, err = Decode(data)
				require.Error(t, err)
			default:
				t.Fatalf("unknown fixture kind %q", stored.Kind)
			}
		})
	}
}

func sliceOrEmpty(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}

func writeFixtures(t *testing.T, cases []fixtureFile) {
	t.Helper()
	require.NoError(t, os.MkdirAll(fixtureDir, 0o755))
	index := fixtureIndex{
		Spec: "op-private-interop/docs/DESIGN.md: the claim is an ordinary L2 transaction, and it " +
			"leads the range it describes.",
		Version: ClaimVersion,
		Transport: "Range N's claim is the LEADING user transaction of range N's first block, and it " +
			"describes range N itself. That is possible rather than circular because " +
			"privateTerminalBlockHash names the PRIVATE chain's block at lastBlock, and the private " +
			"chain runs ahead of its public rendering, so the hash is a known past fact at build time. " +
			"There is therefore no genesis edge: range 0 opens with range 0's claim. These bytes are " +
			"the registry call's struct argument; the selector, the function name and the registry's " +
			"storage layout belong to the registry binding, not to this codec.",
		Registry: "postClaim(RangeClaim calldata claim) — calldata is a 4-byte selector followed by " +
			"exactly these bytes. THERE IS NO EVENT: the registry is log-less by ratified design, " +
			"because a claim leads its range and a rendering-only log at the front of a range-opening " +
			"block would shift every message index in that block. The durable record is the calldata " +
			"itself, plus a storage hash chain the registry exposes through a getter, so a reader scans " +
			"transactions addressed to the registry and reads the chain via that getter rather than " +
			"filtering logs. Nothing re-encodes calldata on the way in, so canonicality is a property a " +
			"reader must CHECK rather than assume — which is why the decoder insists on canonical form, " +
			"and why agreement with solc is pinned by testdata/solidity/RangeClaimVector.t.sol and by " +
			"TestSolidityProducesTheSameBytes. Contiguity between consecutive ranges is REGISTRY " +
			"policy; the truth of privateTerminalBlockHash is off-chain verifier/tooling policy, and " +
			"could not be anything else — the public chain's EVM cannot see the private chain at all.",
		PayloadEncoding: "abi.encode(RangeClaim). The struct has a dynamic member (`proof`), so the " +
			"encoding is the offset word 0x20, then the tuple: ten head words (nine statics plus the " +
			"tuple-relative offset to `proof`, which is 0x140 canonically), then the proof's length word " +
			"and its bytes padded to a multiple of 32.",
		PayloadFields: []string{
			"version", "firstBlock", "lastBlock", "privateTerminalBlockHash", "privateTerminalParentHash",
			"l1Head", "rollupConfigHash", "depSetHash", "privateDataHash", "proof",
		},
		PayloadSolidityTypes: []string{
			"uint8", "uint64", "uint64", "bytes32", "bytes32", "bytes32", "bytes32", "bytes32", "bytes32", "bytes",
		},
		EncodedBytesEmptyProof: EncodedSizeEmptyProof,
		MaxProofBytes:          MaxProofSize,
		MaxEncodedBytes:        MaxEncodedSize,
		RefusedVersions:        []uint16{0x00, 0x02, 0xff},
	}
	for _, c := range cases {
		index.Cases = append(index.Cases, fixtureCase{
			Name: c.Name, Kind: c.Kind, Bin: c.Name + ".bin", JSON: c.Name + ".json",
		})
		c.Spec = index.Spec
		writeJSON(t, filepath.Join(fixtureDir, c.Name+".json"), c)
		require.NoError(t, os.WriteFile(filepath.Join(fixtureDir, c.Bin), c.bytes, 0o644))
	}
	writeJSON(t, filepath.Join(fixtureDir, "index.json"), index)
	t.Logf("wrote %d fixtures to %s", len(cases), fixtureDir)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(readFile(t, path), v), "parsing %s", path)
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, append(data, '\n'), 0o644))
}
