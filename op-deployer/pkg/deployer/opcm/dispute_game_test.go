package opcm

import (
	"fmt"
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

	for _, useV2 := range []bool{false, true} {
		t.Run(fmt.Sprintf("useV2=%v", useV2), func(t *testing.T) {
			input := DeployDisputeGameInput{
				Release:                  "dev",
				UseV2:                    useV2,
				VmAddress:                vmAddr,
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
			}

			script, err := NewDeployDisputeGameScript(host)
			require.NoError(t, err)

			output, err := script.Run(input)
			require.NoError(t, err)

			require.NotEmpty(t, output.DisputeGameImpl)
			require.NotEmpty(t, host.GetCode(output.DisputeGameImpl))
		})
	}
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
