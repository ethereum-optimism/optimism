package sysgo

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBuildSuperRootUpgradeGameConfigs(t *testing.T) {
	cannonPrestate := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	konaPrestate := common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	proposer := common.HexToAddress("0x1111111111111111111111111111111111111111")
	challenger := common.HexToAddress("0x2222222222222222222222222222222222222222")

	configs := buildSuperRootUpgradeGameConfigs(cannonPrestate, konaPrestate, proposer, challenger)
	require.Len(t, configs, 6)

	superPerm := configs[3]
	require.True(t, superPerm.Enabled)
	require.Equal(t, embedded.GameTypeSuperPermCannon, superPerm.GameType)
	require.Equal(t, new(big.Int), superPerm.InitBond)
	require.NotNil(t, superPerm.PermissionedDisputeGameConfig)
	require.Equal(t, cannonPrestate, superPerm.PermissionedDisputeGameConfig.AbsolutePrestate)
	require.Equal(t, proposer, superPerm.PermissionedDisputeGameConfig.Proposer)
	require.Equal(t, challenger, superPerm.PermissionedDisputeGameConfig.Challenger)

	superCannonKona := configs[4]
	require.True(t, superCannonKona.Enabled)
	require.Equal(t, embedded.GameTypeSuperCannonKona, superCannonKona.GameType)
	require.NotNil(t, superCannonKona.FaultDisputeGameConfig)
	require.Equal(t, konaPrestate, superCannonKona.FaultDisputeGameConfig.AbsolutePrestate)
}
