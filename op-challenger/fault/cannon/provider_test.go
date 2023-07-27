package cannon

import (
	"context"
	"embed"
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

//go:embed test_data
var testData embed.FS

func TestGet(t *testing.T) {
	dataDir := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		value, err := provider.Get(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x45fd9aa59768331c726e719e76aa343e73123af888804604785ae19506e65e87"), value)
		require.Empty(t, generator.generated)
	})

	t.Run("ProofUnavailable", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		_, err := provider.Get(context.Background(), 7)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Contains(t, generator.generated, 7, "should have tried to generate the proof")
	})

	t.Run("MissingPostHash", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		_, err := provider.Get(context.Background(), 1)
		require.ErrorContains(t, err, "missing post hash")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		value, err := provider.Get(context.Background(), 2)
		require.NoError(t, err)
		expected := common.HexToHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.Equal(t, expected, value)
		require.Empty(t, generator.generated)
	})
}

func TestGetOracleData(t *testing.T) {
	dataDir := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		oracleData, err := provider.GetOracleData(context.Background(), 420)
		require.NoError(t, err)
		require.False(t, oracleData.IsLocal)
		expectedKey := common.Hex2Bytes("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
		require.Equal(t, expectedKey, oracleData.OracleKey)
		expectedData := common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
		require.Equal(t, expectedData, oracleData.OracleData)
		require.Empty(t, generator.generated)
	})

	t.Run("ProofUnavailable", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		_, err := provider.GetOracleData(context.Background(), 7)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Contains(t, generator.generated, 7, "should have tried to generate the proof")
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		oracleData, err := provider.GetOracleData(context.Background(), 421)
		require.NoError(t, err)
		require.False(t, oracleData.IsLocal)
		expectedKey := common.Hex2Bytes("eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
		require.Equal(t, expectedKey, oracleData.OracleKey)
		expectedData := common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
		require.Equal(t, expectedData, oracleData.OracleData)
		require.Empty(t, generator.generated)
	})
}

func TestGetPreimage(t *testing.T) {
	dataDir := setupTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		value, proof, err := provider.GetPreimage(context.Background(), 0)
		require.NoError(t, err)
		expected := common.Hex2Bytes("b8f068de604c85ea0e2acd437cdb47add074a2d70b81d018390c504b71fe26f400000000000000000000000000000000000000000000000000000000000000000000000000")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("08028e3c0000000000000000000000003c01000a24210b7c00200008000000008fa40004")
		require.Equal(t, expectedProof, proof)
		require.Empty(t, generator.generated)
	})

	t.Run("ProofUnavailable", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		_, _, err := provider.GetPreimage(context.Background(), 7)
		require.ErrorIs(t, err, os.ErrNotExist)
		require.Contains(t, generator.generated, 7, "should have tried to generate the proof")
	})

	t.Run("MissingStateData", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		_, _, err := provider.GetPreimage(context.Background(), 1)
		require.ErrorContains(t, err, "missing state data")
		require.Empty(t, generator.generated)
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		provider, generator := setupWithTestData(dataDir)
		value, proof, err := provider.GetPreimage(context.Background(), 2)
		require.NoError(t, err)
		expected := common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
		require.Equal(t, expectedProof, proof)
		require.Empty(t, generator.generated)
	})
}

func setupTestData(t *testing.T) string {
	srcDir := filepath.Join("test_data", "proofs")
	entries, err := testData.ReadDir(srcDir)
	require.NoError(t, err)
	dataDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dataDir, proofsDir), 0o777))
	for _, entry := range entries {
		path := filepath.Join(srcDir, entry.Name())
		file, err := testData.ReadFile(path)
		require.NoErrorf(t, err, "reading %v", path)
		err = os.WriteFile(filepath.Join(dataDir, "proofs", entry.Name()), file, 0o644)
		require.NoErrorf(t, err, "writing %v", path)
	}
	return dataDir
}

func setupWithTestData(dataDir string) (*CannonTraceProvider, *stubGenerator) {
	generator := &stubGenerator{}
	return &CannonTraceProvider{
		dir:       dataDir,
		generator: generator,
	}, generator
}

type stubGenerator struct {
	generated []int // Using int makes assertions easier
}

func (e *stubGenerator) GenerateProof(ctx context.Context, dir string, i uint64) error {
	e.generated = append(e.generated, int(i))
	return nil
}
