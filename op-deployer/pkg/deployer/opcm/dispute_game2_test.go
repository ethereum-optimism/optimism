package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestNewDeployDisputeGameScript(t *testing.T) {
	t.Run("should not fail with current version of DeployDisputeGame2 contract", func(t *testing.T) {
		// First we grab a test host
		host1 := createTestHost(t)

		// Deploy the prerequisite contracts
		vm1Address := deployDisputeGameScriptVM(t, host1)

		// Then we load the script
		//
		// This would raise an error if the Go types didn't match the ABI
		deploySuperchain, err := NewDeployDisputeGameScript(host1)
		require.NoError(t, err)

		// Then we deploy
		output, err := deploySuperchain.Run(DeployDisputeGame2Input{
			Release:                  "dev",
			StandardVersionsToml:     "dev.toml",
			VmAddress:                vm1Address,
			GameKind:                 "PermissionedDisputeGame",
			GameType:                 big.NewInt(1),
			AbsolutePrestate:         common.Hash{'A'},
			MaxGameDepth:             big.NewInt(int64(standard.DisputeMaxGameDepth)),
			SplitDepth:               big.NewInt(int64(standard.DisputeSplitDepth)),
			ClockExtension:           big.NewInt(int64(standard.DisputeClockExtension)),
			MaxClockDuration:         big.NewInt(int64(standard.DisputeMaxClockDuration)),
			DelayedWethProxy:         common.Address{'D'},
			AnchorStateRegistryProxy: common.Address{'A'},
			L2ChainId:                big.NewInt(69),
			Proposer:                 common.Address{'P'},
			Challenger:               common.Address{'C'},
		})

		// And do some simple asserts
		require.NoError(t, err)
		require.NotNil(t, output)

		// Now we run the old deployer
		//
		// We run it on a fresh host so that the deployer nonces are the same
		// which in turn means we should get identical output
		host2 := createTestHost(t)

		// Deploy the prerequisite contracts
		vm2Address := deployDisputeGameScriptVM(t, host2)
		require.NoError(t, err)

		deprecatedOutput, err := DeployDisputeGame(host2, DeployDisputeGameInput{
			Release:                  "dev",
			VmAddress:                vm2Address,
			GameKind:                 "PermissionedDisputeGame",
			GameType:                 1,
			AbsolutePrestate:         common.Hash{'A'},
			MaxGameDepth:             standard.DisputeMaxGameDepth,
			SplitDepth:               standard.DisputeSplitDepth,
			ClockExtension:           standard.DisputeClockExtension,
			MaxClockDuration:         standard.DisputeMaxClockDuration,
			DelayedWethProxy:         common.Address{'D'},
			AnchorStateRegistryProxy: common.Address{'A'},
			L2ChainId:                common.BigToHash(big.NewInt(69)),
			Proposer:                 common.Address{'P'},
			Challenger:               common.Address{'C'},
		})

		// Make sure it succeeded
		require.NoError(t, err)
		require.NotNil(t, deprecatedOutput)

		// Now make sure the addresses are the same
		require.Equal(t, deprecatedOutput.DisputeGameImpl, output.DisputeGameImpl)

		// And just to be super sure we also compare the code deployed to the addresses
		require.Equal(t, host2.GetCode(deprecatedOutput.DisputeGameImpl), host1.GetCode(output.DisputeGameImpl))
	})
}

func deployDisputeGameScriptVM(t *testing.T, host *script.Host) common.Address {
	preimageOracleArtifact, err := host.Artifacts().ReadArtifact("PreimageOracle.sol", "PreimageOracle")
	require.NoError(t, err)

	encodedPreimageOracleConstructor, err := preimageOracleArtifact.ABI.Pack("", big.NewInt(0), big.NewInt(0))
	require.NoError(t, err)

	preimageOracleAddress, err := host.Create(addresses.ScriptDeployer, append(preimageOracleArtifact.Bytecode.Object, encodedPreimageOracleConstructor...))
	require.NoError(t, err)

	bigStepperArtifact, err := host.Artifacts().ReadArtifact("RISCV.sol", "RISCV")
	require.NoError(t, err)

	encodedBigStepperConstructor, err := bigStepperArtifact.ABI.Pack("", preimageOracleAddress)
	require.NoError(t, err)

	bigStepperAddress, err := host.Create(addresses.ScriptDeployer, append(bigStepperArtifact.Bytecode.Object, encodedBigStepperConstructor...))
	require.NoError(t, err)

	host.Label(preimageOracleAddress, "PreimageOracle")
	host.Label(bigStepperAddress, "BigStepper")

	return bigStepperAddress

}
