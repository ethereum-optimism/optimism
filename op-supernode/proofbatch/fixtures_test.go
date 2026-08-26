package proofbatch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The Cove proof-batch wire format is pinned by fixtures rather than by an implementation: the bytes
// are the contract, and any second implementation is checked against them instead of against this
// code. The corpus is a wire record, so it is language-agnostic and lives beside the codec that
// reads it. What produced it — the prover-side Rust codec — is not on this branch; it is on
// `karl/silhouette-guest`. These bytes are therefore the frozen record of the agreement rather than
// a build output of either side, which is the only form that survives one side being absent.
const fixtureDir = "testdata/proof-batch"

type fixtureIndex struct {
	Magic   string `json:"magic"`
	Version int    `json:"version"`
	// ExportPolicyHash / ExportPolicyV2Hash are the same fact under either name the generator
	// might use; whichever is present is checked against this side's default policy. Accepting
	// both is deliberate: the generator lives on another branch (`karl/silhouette-guest`), and a
	// key rename is not worth a red cross-language gate.
	ExportPolicyHash   common.Hash   `json:"exportPolicyHash"`
	ExportPolicyV2Hash common.Hash   `json:"exportPolicyV2Hash"`
	Cases              []fixtureCase `json:"cases"`
}

// defaultPolicy is whichever spelling the index used, or the zero hash if it pinned neither.
func (i *fixtureIndex) defaultPolicy() common.Hash {
	if i.ExportPolicyHash != (common.Hash{}) {
		return i.ExportPolicyHash
	}
	return i.ExportPolicyV2Hash
}

type fixtureCase struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Bin  string `json:"bin"`
	JSON string `json:"json"`
	// Bytes is the envelope length; zero means the fixture does not pin it.
	Bytes int `json:"bytes"`
}

type fixtureLog struct {
	LogIndex uint32        `json:"logIndex"`
	LogHash  common.Hash   `json:"logHash"`
	Preimage hexutil.Bytes `json:"preimage"`
}

// fixtureExecMsg reads the v3 import list. The generator publishes the six wire fields FLAT (no
// nested `id` object) plus three DERIVED values — logHash, checksum and the canonical 192-byte key —
// which is what turns the fixture from a field-order check into an arithmetic check: this side
// recomputes all three and must agree.
type fixtureExecMsg struct {
	Origin      common.Address `json:"origin"`
	BlockNumber uint64         `json:"blockNumber"`
	LogIndex    uint32         `json:"logIndex"`
	Timestamp   uint64         `json:"timestamp"`
	// ChainID is a DECIMAL STRING in the fixtures, because a u256 does not fit a JSON number.
	ChainID string      `json:"chainId"`
	MsgHash common.Hash `json:"msgHash"`
	// LogHash / Checksum / Key are derived; each is asserted when present.
	LogHash  common.Hash   `json:"logHash"`
	Checksum common.Hash   `json:"checksum"`
	Key      hexutil.Bytes `json:"key"`
}

func (m *fixtureExecMsg) chainID(t *testing.T) eth.ChainID {
	t.Helper()
	require.NotEmpty(t, m.ChainID, "fixture execMsg carries no chain id")
	n, ok := new(big.Int).SetString(m.ChainID, 10)
	require.Truef(t, ok, "fixture execMsg chainId %q is not a decimal integer", m.ChainID)
	return eth.ChainIDFromBig(n)
}

type fixtureBlock struct {
	BlockNumber              uint64           `json:"blockNumber"`
	Timestamp                uint64           `json:"timestamp"`
	BlockHash                common.Hash      `json:"blockHash"`
	StateRoot                common.Hash      `json:"stateRoot"`
	MessagePasserStorageRoot common.Hash      `json:"messagePasserStorageRoot"`
	Logs                     []fixtureLog     `json:"logs"`
	ExecMsgs                 []fixtureExecMsg `json:"execMsgs"`
	// OutputRoot is optional: when the generator publishes it, the two implementations' output
	// root derivations are checked against each other rather than only against the spec.
	OutputRoot common.Hash `json:"outputRoot"`
}

type fixturePublicValues struct {
	PrevOutputRoot   common.Hash    `json:"prevOutputRoot"`
	NewOutputRoot    common.Hash    `json:"newOutputRoot"`
	L1Head           common.Hash    `json:"l1Head"`
	RollupConfigHash common.Hash    `json:"rollupConfigHash"`
	DepSetHash       common.Hash    `json:"depSetHash"`
	ExportPolicyHash common.Hash    `json:"exportPolicyHash"`
	Blocks           []fixtureBlock `json:"blocks"`
}

type fixtureSourceLog struct {
	BlockNumber uint64         `json:"blockNumber"`
	LogIndex    uint           `json:"logIndex"`
	Address     common.Address `json:"address"`
	Topics      []common.Hash  `json:"topics"`
	Data        hexutil.Bytes  `json:"data"`
	LogHash     common.Hash    `json:"logHash"`
}

// fixtureFile is deliberately tolerant: under the slim fixture policy a case pins its bytes with
// keccaks rather than hex mirrors, and the big generative cases carry no source logs. Every field
// that can be absent is checked only when present, so this side never forces the generator's hand.
type fixtureFile struct {
	Envelope struct {
		Version  int `json:"version"`
		PvLen    int `json:"pvLen"`
		ProofLen int `json:"proofLen"`
	} `json:"envelope"`
	PublicValues       *fixturePublicValues `json:"publicValues"`
	PublicValuesKeccak common.Hash          `json:"publicValuesKeccak"`
	ProofHex           hexutil.Bytes        `json:"proofHex"`
	EnvelopeKeccak     common.Hash          `json:"envelopeKeccak"`
	SourceLogs         []fixtureSourceLog   `json:"sourceLogs"`
}

// requireFixtures skips when the canonical set is not in the tree. The set ships with this package,
// so this guard should never fire here; it exists so that a tree carrying only the Go side still
// builds and runs its own spec-derived vectors (proofbatch_test.go) rather than failing, and names
// what it loses when it does — the cross-check against the other implementation.
func requireFixtures(t *testing.T) {
	t.Helper()
	_, err := os.Stat(filepath.Join(fixtureDir, "index.json"))
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("canonical proof-batch fixtures are absent from %s; they are produced by the "+
			"prover-side codec on karl/silhouette-guest, so cross-language verification is skipped", fixtureDir)
	}
	require.NoError(t, err)
}

func readFixtureJSON(t *testing.T, name string, out any) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(fixtureDir, name))
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(raw, out))
}

// requireReadableFixtures skips only a set this codec cannot read AT ALL.
//
// It used to skip any set that was not the CURRENT version, and that was wrong in a way G7 made
// expensive. The two implementations rotate on their own branches and meet at integration, so there
// is a window between a VERSION bump on this side and the regenerated set landing on the other — and
// during that window the old rule silently dropped every one of the 55 cross-language assertions
// while the suite stayed green. Since v2 is still a version this codec decodes (it is the version
// the live chain runs, and the rotation needs both), the set is exercised AT ITS OWN VERSION instead.
// What the gap costs is then named by its own test rather than hidden: see
// TestFixturesCoverCurrentVersion.
func requireReadableFixtures(t *testing.T, idx *fixtureIndex) uint8 {
	t.Helper()
	version := uint8(idx.Version) //nolint:gosec // a version byte; the range check is the next line
	if _, err := argsFor(version); err != nil {
		t.Skipf("proof-batch fixtures in %s are version %d, which this codec does not implement "+
			"(it reads %d and %d); regenerate them from the prover-side codec on karl/silhouette-guest",
			fixtureDir, idx.Version, VersionV2, Version)
	}
	return version
}

// TestFixturesCoverCurrentVersion is the honest accounting of what the cross-language gate is
// covering today. It asserts nothing about bytes; its whole job is to make a MISSING canonical set
// for the current wire version visible as a skipped test with a name, instead of as silence.
func TestFixturesCoverCurrentVersion(t *testing.T) {
	requireFixtures(t)
	var idx fixtureIndex
	readFixtureJSON(t, "index.json", &idx)
	if idx.Version != Version {
		t.Skipf("the canonical fixtures are version %d; wire version %d (execMsgs) is pinned only by "+
			"this side's own layout tests until the Rust crate regenerates the set. "+
			"Byte-identity for v%d is UNVERIFIED across languages.", idx.Version, Version, Version)
	}
	require.Equal(t, Version, idx.Version)
}

func TestFixtures(t *testing.T) {
	requireFixtures(t)
	var idx fixtureIndex
	readFixtureJSON(t, "index.json", &idx)
	version := requireReadableFixtures(t, &idx)
	require.NotEmpty(t, idx.Cases)
	require.Equal(t, Magic, idx.Magic)
	if policy := idx.defaultPolicy(); policy != (common.Hash{}) {
		require.Equal(t, ExportPolicyAllHashes, policy)
	}

	for _, c := range idx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			bin, err := os.ReadFile(filepath.Join(fixtureDir, c.Bin))
			require.NoError(t, err)
			if c.Bytes != 0 {
				require.Len(t, bin, c.Bytes)
			}

			var fx fixtureFile
			readFixtureJSON(t, c.JSON, &fx)
			if fx.EnvelopeKeccak != (common.Hash{}) {
				require.Equal(t, fx.EnvelopeKeccak, crypto.Keccak256Hash(bin))
			}

			env, decodeErr := DecodeAs(bin, version)
			if c.Kind != "valid" {
				requireRejected(t, env, decodeErr)
				return
			}
			require.NoError(t, decodeErr, "expected %s to decode", c.Name)
			require.NoError(t, env.Batch.CheckStructure())

			// The public values are the bytes a proof commits to, so they are pinned as bytes, not
			// only as a decoded object.
			require.Equal(t, fx.PublicValuesKeccak, crypto.Keccak256Hash(env.PublicValues))
			require.Len(t, env.PublicValues, fx.Envelope.PvLen)
			require.Equal(t, []byte(fx.ProofHex), env.Proof)
			require.Len(t, env.Proof, fx.Envelope.ProofLen)

			pv := fx.PublicValues
			require.NotNil(t, pv)
			require.Equal(t, pv.PrevOutputRoot, env.Batch.PrevOutputRoot)
			require.Equal(t, pv.NewOutputRoot, env.Batch.NewOutputRoot)
			require.Equal(t, pv.L1Head, env.Batch.L1Head)
			require.Equal(t, pv.RollupConfigHash, env.Batch.RollupConfigHash)
			require.Equal(t, pv.DepSetHash, env.Batch.DepSetHash)
			require.Equal(t, pv.ExportPolicyHash, env.Batch.ExportPolicyHash)
			require.Len(t, env.Batch.Blocks, len(pv.Blocks))
			for i, want := range pv.Blocks {
				got := env.Batch.Blocks[i]
				require.Equal(t, want.BlockNumber, got.Number)
				require.Equal(t, want.Timestamp, got.Timestamp)
				require.Equal(t, want.BlockHash, got.Hash, "block %d hash", want.BlockNumber)
				require.Equal(t, want.StateRoot, got.StateRoot, "block %d state root", want.BlockNumber)
				require.Equal(t, want.MessagePasserStorageRoot, got.MessagePasserStorageRoot,
					"block %d message passer storage root", want.BlockNumber)
				if want.OutputRoot != (common.Hash{}) {
					require.Equal(t, want.OutputRoot, got.OutputRoot(), "block %d output root", want.BlockNumber)
				}
				require.Len(t, got.Logs, len(want.Logs), "block %d log count", want.BlockNumber)
				for j, wantLog := range want.Logs {
					gotLog := got.Logs[j]
					require.Equal(t, wantLog.LogIndex, gotLog.Index, "block %d logs[%d] index", want.BlockNumber, j)
					require.Equal(t, wantLog.LogHash, gotLog.Hash, "block %d logs[%d] hash", want.BlockNumber, j)
					// An absent preimage is a zero-length `bytes` on the wire, which this codec
					// normalises to nil on decode so a decoded batch re-encodes byte-identically.
					// The rule is about content and length, not about nil-ness, so compare that
					// way — the generator writes an absent preimage as "0x".
					require.Equal(t, len(wantLog.Preimage), len(gotLog.Preimage),
						"block %d logs[%d] preimage length", want.BlockNumber, j)
					require.True(t, bytes.Equal(wantLog.Preimage, gotLog.Preimage),
						"block %d logs[%d] preimage", want.BlockNumber, j)
				}
				// The v3 import list. A v2 set carries none, and a v2 batch must decode to none —
				// asserted either way, because "the field is missing" and "the field is empty" are
				// different claims and the version is what tells them apart.
				require.Len(t, got.ExecMsgs, len(want.ExecMsgs), "block %d execMsg count", want.BlockNumber)
				if version < Version {
					require.Nil(t, got.ExecMsgs, "block %d: a v%d batch cannot carry an import list",
						want.BlockNumber, version)
				}
				for j, wantMsg := range want.ExecMsgs {
					gotMsg := got.ExecMsgs[j]
					where := fmt.Sprintf("block %d execMsgs[%d]", want.BlockNumber, j)
					require.Equal(t, wantMsg.Origin, gotMsg.Identifier.Origin, "%s origin", where)
					require.Equal(t, wantMsg.BlockNumber, gotMsg.Identifier.BlockNumber, "%s blockNumber", where)
					require.Equal(t, wantMsg.LogIndex, gotMsg.Identifier.LogIndex, "%s logIndex", where)
					require.Equal(t, wantMsg.Timestamp, gotMsg.Identifier.Timestamp, "%s timestamp", where)
					require.Equal(t, wantMsg.chainID(t), gotMsg.Identifier.ChainID, "%s chainId", where)
					require.Equal(t, wantMsg.MsgHash, gotMsg.PayloadHash, "%s msgHash", where)

					// The DERIVED values, which are the ones worth crossing a language boundary for: a
					// field-order agreement that disagreed about the checksum would accept batches the
					// judge then failed to resolve.
					if wantMsg.Checksum != (common.Hash{}) {
						require.Equal(t, messages.MessageChecksum(wantMsg.Checksum), gotMsg.Executing().Checksum,
							"%s checksum", where)
					}
					if wantMsg.LogHash != (common.Hash{}) {
						require.Equal(t, wantMsg.LogHash,
							messages.PayloadHashToLogHash(gotMsg.PayloadHash, gotMsg.Identifier.Origin),
							"%s logHash", where)
					}
					// The canonical sort key, byte for byte. This is the ordering rule itself — the one
					// thing about the set that two implementations must agree on to produce the same
					// BYTES for the same import list.
					if len(wantMsg.Key) > 0 {
						key := gotMsg.SortKey()
						require.Equal(t, []byte(wantMsg.Key), key[:], "%s canonical key", where)
					}
				}
			}

			// Byte-identity in the encoding direction too: this is what makes the Go submitter
			// able to post batches the Rust prover side would produce identically.
			reEncoded, err := EncodeAs(&env.Batch, env.Proof, version)
			require.NoError(t, err)
			require.Equal(t, bin, reEncoded)

			blobs, err := ToBlobs(bin)
			require.NoError(t, err)
			back, err := FromBlobs(blobs)
			require.NoError(t, err)
			require.Equal(t, bin, back)

			checkSourceLogs(t, &fx, env)
		})
	}
}

// requireRejected asserts that an invalid fixture is refused, either by the codec or by the
// structural rules. Both are rejections; which one fires is an implementation detail of where the
// rule lives, and the spec only requires that the batch never be accepted.
func requireRejected(t *testing.T, env *Envelope, decodeErr error) {
	t.Helper()
	if decodeErr != nil {
		return
	}
	require.NotNil(t, env)
	require.Error(t, env.Batch.CheckStructure(), "invalid fixture decoded and passed structural checks")
}

// checkSourceLogs recomputes each exported log hash from the log the prover exported it for, and
// checks it was filed under that log's real block-level index. This is the seam where the two
// sides can silently disagree — a log hash derived differently, or an index that is really a
// position — and in v2 the index is carried explicitly precisely so the disagreement is visible.
func checkSourceLogs(t *testing.T, fx *fixtureFile, env *Envelope) {
	t.Helper()
	if len(fx.SourceLogs) == 0 {
		return
	}
	byBlock := map[uint64]map[uint32]common.Hash{}
	for _, blk := range env.Batch.Blocks {
		exported := map[uint32]common.Hash{}
		for _, l := range blk.Logs {
			exported[l.Index] = l.Hash
		}
		byBlock[blk.Number] = exported
	}
	for _, src := range fx.SourceLogs {
		got := messages.LogToLogHash(&types.Log{
			Address: src.Address,
			Topics:  src.Topics,
			Data:    src.Data,
		})
		require.Equal(t, src.LogHash, got, "log %d of block %d", src.LogIndex, src.BlockNumber)
		exported, ok := byBlock[src.BlockNumber][uint32(src.LogIndex)]
		require.True(t, ok, "block %d exports no log at index %d", src.BlockNumber, src.LogIndex)
		require.Equal(t, got, exported, "exported hash must be filed under the log's own index")
	}
}

func TestFixtureLogHashVectors(t *testing.T) {
	requireFixtures(t)
	var vectors struct {
		Vectors []struct {
			Name string           `json:"name"`
			Log  fixtureSourceLog `json:"log"`
		} `json:"vectors"`
	}
	readFixtureJSON(t, "log-hash-vectors.json", &vectors)
	require.NotEmpty(t, vectors.Vectors)
	for _, v := range vectors.Vectors {
		t.Run(v.Name, func(t *testing.T) {
			require.Equal(t, v.Log.LogHash, messages.LogToLogHash(&types.Log{
				Address: v.Log.Address,
				Topics:  v.Log.Topics,
				Data:    v.Log.Data,
			}))
		})
	}
}
