package main

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/proofbatch"
)

// sampleBatch is a well-formed two-block batch whose NewOutputRoot really derives from its last
// block, so that a report claiming otherwise is a bug in this tool rather than in the fixture.
func sampleBatch(t *testing.T) *proofbatch.ProofBatch {
	t.Helper()
	blocks := []proofbatch.BlockExport{
		{
			Number: 41, Timestamp: 1_700_000_002,
			Hash:                     common.HexToHash("0xaa41"),
			StateRoot:                common.HexToHash("0xbb41"),
			MessagePasserStorageRoot: common.HexToHash("0xcc41"),
		},
		{
			Number: 42, Timestamp: 1_700_000_004,
			Hash:                     common.HexToHash("0xaa42"),
			StateRoot:                common.HexToHash("0xbb42"),
			MessagePasserStorageRoot: common.HexToHash("0xcc42"),
			Logs: []proofbatch.LogExport{
				{Index: 0, Hash: common.HexToHash("0x100")},
				{Index: 3, Hash: common.HexToHash("0x103")},
			},
		},
	}
	b := &proofbatch.ProofBatch{
		PrevOutputRoot:   common.HexToHash("0xdead"),
		L1Head:           common.HexToHash("0xbeef"),
		RollupConfigHash: common.HexToHash("0xc0ffee"),
		DepSetHash:       common.HexToHash("0xd00d"),
		ExportPolicyHash: proofbatch.ExportPolicyAllHashes,
		Blocks:           blocks,
	}
	b.NewOutputRoot = blocks[len(blocks)-1].OutputRoot()
	require.NoError(t, b.CheckStructure())
	return b
}

func encode(t *testing.T, b *proofbatch.ProofBatch, proof []byte) []byte {
	t.Helper()
	raw, err := proofbatch.Encode(b, proof)
	require.NoError(t, err)
	return raw
}

// TestInspectReadsWhatTheSubmitterWrote is the round trip that makes this tool worth trusting: the
// bytes are framed into blobs by the submitter's own encoder and lifted back out by the tool, so a
// framing change breaks this test rather than breaking a cutover.
func TestInspectReadsWhatTheSubmitterWrote(t *testing.T) {
	b := sampleBatch(t)
	raw := encode(t, b, nil)

	blobs, err := proofbatch.ToBlobs(raw)
	require.NoError(t, err)
	require.NotEmpty(t, blobs)

	dir := t.TempDir()
	paths := make([]string, 0, len(blobs))
	for i, blob := range blobs {
		p := filepath.Join(dir, "blob-"+string(rune('a'+i)))
		require.NoError(t, os.WriteFile(p, blob[:], 0o600))
		paths = append(paths, p)
	}

	lifted := make([]*eth.Blob, 0, len(paths))
	for _, p := range paths {
		blob, err := readBlob(p)
		require.NoError(t, err)
		lifted = append(lifted, blob)
	}
	back, err := proofbatch.FromBlobs(lifted)
	require.NoError(t, err)

	env, err := proofbatch.Decode(back)
	require.NoError(t, err)
	rep := describe(env)

	// The two values the cross-check exists for.
	require.Equal(t, b.RollupConfigHash, rep.RollupConfigHash)
	require.Equal(t, b.DepSetHash, rep.DepSetHash)
	require.True(t, rep.ExportPolicyIsAllHashes)

	require.Equal(t, b.PrevOutputRoot, rep.PrevOutputRoot)
	require.Equal(t, b.NewOutputRoot, rep.NewOutputRoot)
	require.True(t, rep.NewOutputRootDerives)
	require.Empty(t, rep.StructureError)
	require.Equal(t, 2, rep.BlockCount)
	require.EqualValues(t, 41, rep.FirstBlock)
	require.EqualValues(t, 42, rep.LastBlock)
	require.Equal(t, 2, rep.ExportedLogs)
	require.Zero(t, rep.ProofBytes, "an attested batch posts no proof, and the report must say so as a size")
	require.Positive(t, rep.PublicValuesBytes)
}

// TestInspectReportsAMalformedBatchRatherThanRefusingIt. A tool used to diagnose a batch that is
// already being rejected must be able to read a bad one: refusing to decode is the one behaviour
// that would make it useless exactly when it is needed.
func TestInspectReportsAMalformedBatchRatherThanRefusingIt(t *testing.T) {
	b := sampleBatch(t)
	b.NewOutputRoot = common.HexToHash("0x1234") // no longer derives from the last block
	env, err := proofbatch.Decode(encode(t, b, []byte("proof-bytes")))
	require.NoError(t, err)

	rep := describe(env)
	require.False(t, rep.NewOutputRootDerives, "the tool must name this, not hide it")
	require.Equal(t, len("proof-bytes"), rep.ProofBytes)
}

// TestInspectAcceptsHexAndBeaconSidecarJSON covers the two encodings an operator actually has in
// hand: hex text pasted out of a beacon response, and the sidecar object itself.
func TestInspectAcceptsHexAndBeaconSidecarJSON(t *testing.T) {
	raw := encode(t, sampleBatch(t), nil)
	blobs, err := proofbatch.ToBlobs(raw)
	require.NoError(t, err)
	blob := blobs[0]
	dir := t.TempDir()

	hexPath := filepath.Join(dir, "blob.hex")
	require.NoError(t, os.WriteFile(hexPath, []byte("0x"+hex.EncodeToString(blob[:])+"\n"), 0o600))
	got, err := readBlob(hexPath)
	require.NoError(t, err)
	require.Equal(t, blob[:], got[:])

	sidecar, err := json.Marshal(map[string]string{"blob": "0x" + hex.EncodeToString(blob[:])})
	require.NoError(t, err)
	jsonPath := filepath.Join(dir, "sidecar.json")
	require.NoError(t, os.WriteFile(jsonPath, sidecar, 0o600))
	got, err = readBlob(jsonPath)
	require.NoError(t, err)
	require.Equal(t, blob[:], got[:])

	short := filepath.Join(dir, "short.bin")
	require.NoError(t, os.WriteFile(short, []byte("nope"), 0o600))
	_, err = readBlob(short)
	require.ErrorContains(t, err, "a blob is exactly")
}
