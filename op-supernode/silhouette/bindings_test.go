package silhouette

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-core/interop/depset"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestBindingHashesGolden(t *testing.T) {
	cfg := &rollup.Config{
		Genesis: rollup.Genesis{
			L1:     eth.BlockID{Hash: common.HexToHash("0x1111"), Number: 12},
			L2:     eth.BlockID{Hash: common.HexToHash("0x2222"), Number: 34},
			L2Time: 1_800_000_000,
		},
		BlockTime:         2,
		SeqWindowSize:     3600,
		L1ChainID:         big.NewInt(900),
		L2ChainID:         big.NewInt(901),
		BatchInboxAddress: common.HexToAddress("0x1234"),
	}
	depSet, err := depset.NewStaticConfigDependencySet(map[eth.ChainID]*depset.StaticConfigDependency{
		eth.ChainIDFromUInt64(901): {},
		eth.ChainIDFromUInt64(902): {},
	})
	require.NoError(t, err)

	rollupHash, depSetHash, err := BindingHashes(cfg, depSet)
	require.NoError(t, err)
	require.Equal(t, common.HexToHash("0x2c0fecad745edee988a8b25d9eed31ca5ea7c170b4881523731f1b07d74b75c2"), rollupHash)
	require.Equal(t, common.HexToHash("0x343a504cd220194b36ed907bbc644fbd58d2b3e32bcbbe271f11e17f50f1ba74"), depSetHash)
}

func TestCanonicalArtifactHashRejectsNil(t *testing.T) {
	_, err := CanonicalArtifactHash(nil)
	require.ErrorContains(t, err, "nil artifact")
}
