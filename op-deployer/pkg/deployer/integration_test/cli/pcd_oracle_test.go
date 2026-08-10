package cli

import (
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

const (
	pcdOracleOneMemberOutputRoot = "0x9a796318a96bc4298e35dda0e08fee1a3e4a75d72e8db3f7b17280a84d34d8bb"
	pcdOracleOneMemberSuperRoot  = "0x0121849e39dd263255e80f3457bded5be97fb3294c87051b5921e5293cf0c24c"
	pcdOracleTwoMemberSuperRoot  = "0xd52294ac6b9dd6cdd394a20e1f659e48901913543a5a167a57d2409d392aa5fc"
)

func TestPCDOracleOneMemberSuperRoot(t *testing.T) {
	artifact := pcdOracleTestArtifact("genesis-chain-900.json")
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
	first := pcdOracleTestArtifact("genesis-chain-900.json")
	second := pcdOracleTestArtifact("genesis-chain-901.json")

	forward, forwardTime, err := pcdSuperRootFromArtifacts([]pcdChainArtifacts{first, second})
	require.NoError(t, err)
	reversed, reversedTime, err := pcdSuperRootFromArtifacts([]pcdChainArtifacts{second, first})
	require.NoError(t, err)

	require.Equal(t, uint64(1234), forwardTime)
	require.Equal(t, forwardTime, reversedTime)
	require.Equal(t, common.HexToHash(pcdOracleTwoMemberSuperRoot), forward)
	require.Equal(t, forward, reversed)
}

func pcdOracleTestArtifact(genesisFile string) pcdChainArtifacts {
	return pcdChainArtifacts{
		genesisPath: filepath.Join("testdata", "oracle", genesisFile),
		rollupPath:  filepath.Join("testdata", "oracle", "rollup.json"),
	}
}
