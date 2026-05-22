package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestDeployScriptsForge tests deployment scripts via Forge with Anvil
func TestDeployScriptsForge(t *testing.T) {
	l1ChainID := uint64(devnet.DefaultChainID)
	l1ChainIDBig := big.NewInt(int64(l1ChainID))

	dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	superchainProxyAdminOwner := shared.AddrFor(t, dk, devkeys.L1ProxyAdminOwnerRole.Key(l1ChainIDBig))
	guardian := shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	challenger := shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	t.Run("deploy altda with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		forgeClient := newLocalForgeClient(t)

		// Deploy AltDA using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployAltDAViaForge(forgeEnv, opcm.DeployAltDAInput{
			"salt":                     common.BigToHash(big.NewInt(12345)),
			"proxyAdmin":               superchainProxyAdminOwner,
			"challengeContractOwner":   challenger,
			"challengeWindow":          big.NewInt(3600),
			"resolveWindow":            big.NewInt(7200),
			"bondSize":                 big.NewInt(1000000000000000000), // 1 ETH
			"resolverRefundPercentage": big.NewInt(50),
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.Address("dataAvailabilityChallengeProxy"))
		require.NotEqual(t, common.Address{}, output.Address("dataAvailabilityChallengeImpl"))
	})

	t.Run("deploy alphabet vm with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_alphabet.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		implsOutputFile := filepath.Join(runner.GetWorkDir(), "impls_for_alphabet.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--superchain-config-proxy", superchainOutput.Address("superchainConfigProxy").Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.Address("superchainProxyAdmin").Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		forgeClient := newLocalForgeClient(t)

		// Deploy AlphabetVM using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployAlphabetVMViaForge(forgeEnv, opcm.DeployAlphabetVMInput{
			"absolutePrestate": common.BigToHash(big.NewInt(12345)),
			"preimageOracle":   implsOutput.Address("preimageOracleSingleton"),
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.Address("alphabetVM"))
	})

	t.Run("deploy mips with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		// First deploy PreimageOracle (needed for MIPS)
		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_mips.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		implsOutputFile := filepath.Join(runner.GetWorkDir(), "impls_for_mips.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--superchain-config-proxy", superchainOutput.Address("superchainConfigProxy").Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.Address("superchainProxyAdmin").Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		forgeClient := newLocalForgeClient(t)

		// Deploy MIPS using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployMIPSViaForge(forgeEnv, opcm.DeployMIPSInput{
			"preimageOracle": implsOutput.Address("preimageOracleSingleton"),
			"mipsVersion":    big.NewInt(int64(standard.MIPSVersion)),
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.Address("mipsSingleton"))
	})

	t.Run("deploy dispute game with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_dispute.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		implsOutputFile := filepath.Join(runner.GetWorkDir(), "impls_for_dispute.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "implementations",
			"--outfile", implsOutputFile,
			"--mips-version", strconv.Itoa(int(standard.MIPSVersion)),
			"--superchain-config-proxy", superchainOutput.Address("superchainConfigProxy").Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.Address("superchainProxyAdmin").Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		forgeClient := newLocalForgeClient(t)

		// Deploy DisputeGame using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployDisputeGameViaForge(forgeEnv, opcm.DeployDisputeGameInput{
			"release":                  "dev",
			"gameKind":                 "FaultDisputeGame",
			"gameType":                 uint32(1),
			"absolutePrestate":         common.BigToHash(big.NewInt(12345)),
			"maxGameDepth":             big.NewInt(int64(standard.DisputeMaxGameDepth)),
			"splitDepth":               big.NewInt(int64(standard.DisputeSplitDepth)),
			"clockExtension":           standard.DisputeClockExtension,
			"maxClockDuration":         standard.DisputeMaxClockDuration,
			"delayedWethProxy":         implsOutput.Address("delayedWETHImpl"), // Use impl address as placeholder
			"anchorStateRegistryProxy": implsOutput.Address("anchorStateRegistryImpl"),
			"vmAddress":                implsOutput.Address("mipsSingleton"),
			"l2ChainId":                big.NewInt(420),
			"proposer":                 shared.AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainIDBig)),
			"challenger":               challenger,
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.Address("disputeGameImpl"))
	})

	t.Run("read superchain deployment with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_for_read.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		forgeClient := newLocalForgeClient(t)

		// Read superchain deployment using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:   forgeClient,
			Context:  context.Background(),
			L1RPCUrl: runner.GetL1RPC(),
			// PrivateKey not required for read-only operations
		}
		output, err := opcm.ReadSuperchainDeploymentViaForge(forgeEnv, opcm.ReadSuperchainDeploymentInput{
			"superchainConfigProxy": superchainOutput.Address("superchainConfigProxy"),
		})
		require.NoError(t, err)

		require.Equal(t, superchainOutput.Address("superchainConfigProxy"), output.Address("superchainConfigProxy"))
		require.Equal(t, superchainOutput.Address("superchainConfigImpl"), output.Address("superchainConfigImpl"))
		require.Equal(t, superchainOutput.Address("superchainProxyAdmin"), output.Address("superchainProxyAdmin"))
		require.NotEqual(t, common.Address{}, output.Address("guardian"))
		require.NotEqual(t, common.Address{}, output.Address("superchainProxyAdminOwner"))
	})
}

func newLocalForgeClient(t *testing.T) *forge.Client {
	_, afacts := testutil.LocalArtifacts(t)
	forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", afacts))
	require.NoError(t, err)
	return forgeClient
}
