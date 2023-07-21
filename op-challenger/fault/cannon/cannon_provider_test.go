package cannon

import (
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
	provider := setupWithTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		value, err := provider.Get(0)
		require.NoError(t, err)
		require.Equal(t, common.HexToHash("0x45fd9aa59768331c726e719e76aa343e73123af888804604785ae19506e65e87"), value)
	})

	t.Run("ProofUnavailable", func(t *testing.T) {
		_, err := provider.Get(7)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("MissingPostHash", func(t *testing.T) {
		_, err := provider.Get(1)
		require.ErrorContains(t, err, "missing post hash")
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		value, err := provider.Get(2)
		require.NoError(t, err)
		expected := common.HexToHash("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
		require.Equal(t, expected, value)
	})
}

func TestGetPreimage(t *testing.T) {
	provider := setupWithTestData(t)
	t.Run("ExistingProof", func(t *testing.T) {
		value, proof, err := provider.GetPreimage(0)
		require.NoError(t, err)
		expected := common.Hex2Bytes("b8f068de604c85ea0e2acd437cdb47add074a2d70b81d018390c504b71fe26f400000000000000000000000000000000000000000000000000000000000000000000000000")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("08028e3c0000000000000000000000003c01000a24210b7c00200008000000008fa40004")
		require.Equal(t, expectedProof, proof)
	})

	t.Run("ProofUnavailable", func(t *testing.T) {
		_, _, err := provider.GetPreimage(7)
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("MissingStateData", func(t *testing.T) {
		_, _, err := provider.GetPreimage(1)
		require.ErrorContains(t, err, "missing state data")
	})

	t.Run("IgnoreUnknownFields", func(t *testing.T) {
		value, proof, err := provider.GetPreimage(2)
		require.NoError(t, err)
		expected := common.Hex2Bytes("cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc")
		require.Equal(t, expected, value)
		expectedProof := common.Hex2Bytes("dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd")
		require.Equal(t, expectedProof, proof)
	})
}

func setupWithTestData(t *testing.T) *CannonTraceProvider {
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
	return NewCannonTraceProvider(dataDir)
}
