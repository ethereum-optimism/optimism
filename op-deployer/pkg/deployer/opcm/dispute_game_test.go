package opcm

import (
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-chain-ops/script/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/broadcaster"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestDeployDisputeGame(t *testing.T) {
	t.Parallel()

	_, artifacts := testutil.LocalArtifacts(t)

	host, err := env.DefaultScriptHost(
		broadcaster.NoopBroadcaster(),
		testlog.Logger(t, log.LevelInfo),
		common.Address{'D'},
		artifacts,
	)
	require.NoError(t, err)

	vmAddr := deployDisputeGameScriptVM(t, host)

	input := DeployDisputeGameInput{
		"release":                  "dev",
		"vmAddress":                vmAddr,
		"gameKind":                 "PermissionedDisputeGame",
		"gameType":                 uint32(1),
		"absolutePrestate":         common.Hash{'A'},
		"maxGameDepth":             big.NewInt(int64(standard.DisputeMaxGameDepth)),
		"splitDepth":               big.NewInt(int64(standard.DisputeSplitDepth)),
		"clockExtension":           standard.DisputeClockExtension,
		"maxClockDuration":         standard.DisputeMaxClockDuration,
		"delayedWethProxy":         common.Address{'D'},
		"anchorStateRegistryProxy": common.Address{'A'},
		"l2ChainId":                big.NewInt(69),
		"proposer":                 common.Address{'P'},
		"challenger":               common.Address{'C'},
	}

	script, err := NewDeployDisputeGameScript(host)
	require.NoError(t, err)

	output, err := script.Run(input)
	require.NoError(t, err)

	disputeGameImpl := output.Address("disputeGameImpl")
	require.NotEmpty(t, disputeGameImpl)
	require.NotEmpty(t, host.GetCode(disputeGameImpl))
}

func deployDisputeGameScriptVM(t *testing.T, host *script.Host) common.Address {
	preimageOracleArtifact, err := host.Artifacts().ReadArtifact("PreimageOracle.sol", "PreimageOracle")
	require.NoError(t, err)

	encodedPreimageOracleConstructor, err := preimageOracleArtifact.ABI.Pack("", big.NewInt(0), big.NewInt(0))
	require.NoError(t, err)

	preimageOracleAddress, err := host.Create(addresses.ScriptDeployer, append(preimageOracleArtifact.Bytecode.Object, encodedPreimageOracleConstructor...))
	require.NoError(t, err)

	bigStepperArtifact, err := host.Artifacts().ReadArtifact("MIPS64.sol", "MIPS64")
	require.NoError(t, err)

	encodedBigStepperConstructor, err := bigStepperArtifact.ABI.Pack("", preimageOracleAddress, new(big.Int).SetUint64(standard.MIPSVersion))
	require.NoError(t, err)

	bigStepperAddress, err := host.Create(addresses.ScriptDeployer, append(bigStepperArtifact.Bytecode.Object, encodedBigStepperConstructor...))
	require.NoError(t, err)

	host.Label(preimageOracleAddress, "PreimageOracle")
	host.Label(bigStepperAddress, "BigStepper")

	return bigStepperAddress
}
