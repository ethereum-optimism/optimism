package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
)

func TestDeployDisputeGame(t *testing.T) {
	t.Parallel()

	env := newTestGenesisEnv(t)
	vm, err := DeployMIPS(env, DeployMIPSInput{
		PreimageOracle: common.Address{'P'},
		MipsVersion:    new(big.Int).SetUint64(standard.MIPSVersion),
	})
	require.NoError(t, err)

	output, err := DeployDisputeGame(env, DeployDisputeGameInput{
		Release:                  "dev",
		VmAddress:                vm.MipsSingleton,
		GameKind:                 "PermissionedDisputeGame",
		GameType:                 1,
		AbsolutePrestate:         common.Hash{'A'},
		MaxGameDepth:             big.NewInt(int64(standard.DisputeMaxGameDepth)),
		SplitDepth:               big.NewInt(int64(standard.DisputeSplitDepth)),
		ClockExtension:           standard.DisputeClockExtension,
		MaxClockDuration:         standard.DisputeMaxClockDuration,
		DelayedWethProxy:         common.Address{'D'},
		AnchorStateRegistryProxy: common.Address{'A'},
		L2ChainId:                big.NewInt(69),
		Proposer:                 common.Address{'P'},
		Challenger:               common.Address{'C'},
	})
	require.NoError(t, err)
	requireCode(t, env, output.DisputeGameImpl)
}
