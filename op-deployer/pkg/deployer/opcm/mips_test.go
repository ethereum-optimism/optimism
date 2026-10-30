package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
)

func TestDeployMIPS(t *testing.T) {
	env := newTestGenesisEnv(t)
	output, err := DeployMIPS(env, DeployMIPSInput{
		PreimageOracle: common.Address{'P'},
		MipsVersion:    new(big.Int).SetUint64(standard.MIPSVersion),
	})
	require.NoError(t, err)
	requireCode(t, env, output.MipsSingleton)
}
