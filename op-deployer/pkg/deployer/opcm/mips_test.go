package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
)

func TestNewDeployMIPSScript(t *testing.T) {
	t.Run("should not fail with current version of DeployMIPS contract", func(t *testing.T) {
		host1 := createTestHost(t)

		deploySuperchain, err := NewDeployMIPSScript(host1)
		require.NoError(t, err)

		mipsVersion := int64(standard.MIPSVersion)
		output, err := deploySuperchain.Run(DeployMIPSInput{
			PreimageOracle: common.Address{'P'},
			MipsVersion:    big.NewInt(mipsVersion),
		})

		require.NoError(t, err)
		require.NotNil(t, output)

		host2 := createTestHost(t)
		deprecatedOutput, err := DeployMIPS(host2, DeployMIPSInput{
			PreimageOracle: common.Address{'P'},
			MipsVersion:    big.NewInt(mipsVersion),
		})

		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		require.Equal(t, deprecatedOutput.MipsSingleton, output.MipsSingleton)

		require.Equal(t, host2.GetCode(deprecatedOutput.MipsSingleton), host1.GetCode(output.MipsSingleton))
	})
}
