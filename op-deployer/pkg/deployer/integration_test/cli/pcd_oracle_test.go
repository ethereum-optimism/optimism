package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

const (
	pcdOracleOneMemberOutputRoot = "0x9a796318a96bc4298e35dda0e08fee1a3e4a75d72e8db3f7b17280a84d34d8bb"
	pcdOracleOneMemberSuperRoot  = "0x0121849e39dd263255e80f3457bded5be97fb3294c87051b5921e5293cf0c24c"
	pcdOracleTwoMemberSuperRoot  = "0xd52294ac6b9dd6cdd394a20e1f659e48901913543a5a167a57d2409d392aa5fc"
)

func TestPCDOracleOneMemberSuperRoot(t *testing.T) {
	artifact := pcdOracleTestArtifact(900)
	got, genesisTime, err := pcdSuperRootFromArtifacts([]pcdChainArtifacts{artifact})
	require.NoError(t, err)
	require.Equal(t, uint64(1234), genesisTime)
	require.Equal(t, common.HexToHash(pcdOracleOneMemberSuperRoot), got)

	genesis, err := readPCDGenesis(artifact.genesisPath)
	require.NoError(t, err)
	header := genesis.ToBlock().Header()
	require.NotNil(t, header.WithdrawalsHash)
	require.NotEqual(t, types.EmptyRootHash, *header.WithdrawalsHash)
	memberRoot := common.Hash(pcdOutputRoot(header))
	require.Equal(t, common.HexToHash(pcdOracleOneMemberOutputRoot), memberRoot)
	require.NotEqual(t, memberRoot, got, "one-member SuperV1 root must not use the bare OutputV0 root")
}

func TestPCDOracleTwoMembersSortsByChainID(t *testing.T) {
	first := pcdOracleTestArtifact(900)
	second := pcdOracleTestArtifact(901)

	forward, forwardTime, err := pcdSuperRootFromArtifacts([]pcdChainArtifacts{first, second})
	require.NoError(t, err)
	reversed, reversedTime, err := pcdSuperRootFromArtifacts([]pcdChainArtifacts{second, first})
	require.NoError(t, err)

	require.Equal(t, uint64(1234), forwardTime)
	require.Equal(t, forwardTime, reversedTime)
	require.Equal(t, common.HexToHash(pcdOracleTwoMemberSuperRoot), forward)
	require.Equal(t, forward, reversed)
}

func TestPCDOracleRejectsInvalidChainIDs(t *testing.T) {
	chain900 := pcdOracleTestArtifact(900)
	chain901 := pcdOracleTestArtifact(901)

	swapped900 := chain900
	swapped900.genesisPath = chain901.genesisPath
	swapped900.rollupPath = chain901.rollupPath
	swapped901 := chain901
	swapped901.genesisPath = chain900.genesisPath
	swapped901.rollupPath = chain900.rollupPath

	missingRollupPath := filepath.Join(t.TempDir(), "rollup.json")
	require.NoError(t, os.WriteFile(missingRollupPath, []byte(`{"genesis":{"l2_time":1234}}`), 0o600))
	missingRollupChainID := chain900
	missingRollupChainID.rollupPath = missingRollupPath

	mismatchedRollup := chain900
	mismatchedRollup.rollupPath = chain901.rollupPath

	tests := []struct {
		name      string
		artifacts []pcdChainArtifacts
		wantErr   string
	}{
		{
			name:      "swapped chain artifacts",
			artifacts: []pcdChainArtifacts{swapped900, swapped901},
			wantErr:   "identifies chain 901, expected chain 900",
		},
		{
			name:      "rollup L2 chain ID is missing",
			artifacts: []pcdChainArtifacts{missingRollupChainID},
			wantErr:   "does not identify an L2 chain",
		},
		{
			name:      "genesis and rollup chain IDs do not match",
			artifacts: []pcdChainArtifacts{mismatchedRollup},
			wantErr:   "identifies chain 901, expected chain 900",
		},
		{
			name:      "duplicate chain ID",
			artifacts: []pcdChainArtifacts{chain900, chain900},
			wantErr:   "contain duplicate chain ID 900",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := pcdSuperRootFromArtifacts(test.artifacts)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func pcdOracleTestArtifact(chainID uint64) pcdChainArtifacts {
	artifactDir := filepath.Join("testdata", "oracle", "chain-"+strconv.FormatUint(chainID, 10))
	return pcdChainArtifacts{
		chainID:     uint256.NewInt(chainID).Bytes32(),
		genesisPath: filepath.Join(artifactDir, "genesis.json"),
		rollupPath:  filepath.Join(artifactDir, "rollup.json"),
	}
}
