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
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
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
	protocolVersionsOwner := shared.AddrFor(t, dk, devkeys.SuperchainDeployerKey.Key(l1ChainIDBig))
	guardian := shared.AddrFor(t, dk, devkeys.SuperchainConfigGuardianKey.Key(l1ChainIDBig))
	challenger := shared.AddrFor(t, dk, devkeys.ChallengerRole.Key(l1ChainIDBig))

	t.Run("deploy altda with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		tmpDir := t.TempDir()
		embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
		require.NoError(t, err)

		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
		require.NoError(t, err)

		// Deploy AltDA using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployAltDAViaForge(forgeEnv, opcm.DeployAltDAInput{
			Salt:                     common.BigToHash(big.NewInt(12345)),
			ProxyAdmin:               superchainProxyAdminOwner,
			ChallengeContractOwner:   challenger,
			ChallengeWindow:          big.NewInt(3600),
			ResolveWindow:            big.NewInt(7200),
			BondSize:                 big.NewInt(1000000000000000000), // 1 ETH
			ResolverRefundPercentage: big.NewInt(50),
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.DataAvailabilityChallengeProxy)
		require.NotEqual(t, common.Address{}, output.DataAvailabilityChallengeImpl)
	})

	t.Run("deploy alphabet vm with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_alphabet.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		tmpDir := t.TempDir()
		embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
		require.NoError(t, err)

		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
		require.NoError(t, err)

		// Deploy AlphabetVM using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployAlphabetVMViaForge(forgeEnv, opcm.DeployAlphabetVMInput{
			AbsolutePrestate: common.BigToHash(big.NewInt(12345)),
			PreimageOracle:   implsOutput.PreimageOracleSingleton,
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.AlphabetVM)
	})

	t.Run("deploy mips with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		// First deploy PreimageOracle (needed for MIPS)
		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_mips.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		tmpDir := t.TempDir()
		embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
		require.NoError(t, err)

		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
		require.NoError(t, err)

		// Deploy MIPS using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployMIPSViaForge(forgeEnv, opcm.DeployMIPSInput{
			PreimageOracle: implsOutput.PreimageOracleSingleton,
			MipsVersion:    big.NewInt(int64(standard.MIPSVersion)),
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.MipsSingleton)
	})

	t.Run("deploy dispute game with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)

		superchainOutputFile := filepath.Join(runner.GetWorkDir(), "superchain_for_dispute.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
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
			"--protocol-versions-proxy", superchainOutput.ProtocolVersionsProxy.Hex(),
			"--superchain-config-proxy", superchainOutput.SuperchainConfigProxy.Hex(),
			"--l1-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--superchain-proxy-admin", superchainOutput.SuperchainProxyAdmin.Hex(),
			"--challenger", challenger.Hex(),
			"--use-forge",
		}, nil)

		var implsOutput opcm.DeployImplementationsOutput
		data, err = os.ReadFile(implsOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &implsOutput)
		require.NoError(t, err)

		tmpDir := t.TempDir()
		embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
		require.NoError(t, err)

		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
		require.NoError(t, err)

		// Deploy DisputeGame using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:     forgeClient,
			Context:    context.Background(),
			L1RPCUrl:   runner.GetL1RPC(),
			PrivateKey: runner.GetPrivateKey(),
		}
		output, err := opcm.DeployDisputeGameViaForge(forgeEnv, opcm.DeployDisputeGameInput{
			Release:                  "dev",
			GameKind:                 "FaultDisputeGame",
			GameType:                 1,
			AbsolutePrestate:         common.BigToHash(big.NewInt(12345)),
			MaxGameDepth:             big.NewInt(int64(standard.DisputeMaxGameDepth)),
			SplitDepth:               big.NewInt(int64(standard.DisputeSplitDepth)),
			ClockExtension:           standard.DisputeClockExtension,
			MaxClockDuration:         standard.DisputeMaxClockDuration,
			DelayedWethProxy:         implsOutput.DelayedWETHImpl, // Use impl address as placeholder
			AnchorStateRegistryProxy: implsOutput.AnchorStateRegistryImpl,
			VmAddress:                implsOutput.MipsSingleton,
			L2ChainId:                big.NewInt(420),
			Proposer:                 shared.AddrFor(t, dk, devkeys.ProposerRole.Key(l1ChainIDBig)),
			Challenger:               challenger,
		})
		require.NoError(t, err)
		require.NotEqual(t, common.Address{}, output.DisputeGameImpl)
	})

	t.Run("read superchain deployment with forge", func(t *testing.T) {
		runner := NewCLITestRunnerWithNetwork(t)
		workDir := runner.GetWorkDir()

		superchainOutputFile := filepath.Join(workDir, "bootstrap_superchain_for_read.json")
		runner.ExpectSuccessWithNetwork(t, []string{
			"bootstrap", "superchain",
			"--outfile", superchainOutputFile,
			"--superchain-proxy-admin-owner", superchainProxyAdminOwner.Hex(),
			"--protocol-versions-owner", protocolVersionsOwner.Hex(),
			"--guardian", guardian.Hex(),
			"--use-forge",
		}, nil)

		var superchainOutput opcm.DeploySuperchainOutput
		data, err := os.ReadFile(superchainOutputFile)
		require.NoError(t, err)
		err = json.Unmarshal(data, &superchainOutput)
		require.NoError(t, err)

		tmpDir := t.TempDir()
		embeddedArtifactsFS, err := artifacts.ExtractEmbedded(tmpDir)
		require.NoError(t, err)

		forgeClient, err := forge.NewStandardClient(fmt.Sprintf("%v", embeddedArtifactsFS))
		require.NoError(t, err)

		// Read superchain deployment using Forge wrapper function
		forgeEnv := &opcm.ForgeEnv{
			Client:   forgeClient,
			Context:  context.Background(),
			L1RPCUrl: runner.GetL1RPC(),
			// PrivateKey not required for read-only operations
		}
		output, err := opcm.ReadSuperchainDeploymentViaForge(forgeEnv, opcm.ReadSuperchainDeploymentInput{
			OpcmAddress:           common.Address{}, // OPCM v2 flow - use SuperchainConfigProxy
			SuperchainConfigProxy: superchainOutput.SuperchainConfigProxy,
		})
		require.NoError(t, err)

		require.Equal(t, superchainOutput.SuperchainConfigProxy, output.SuperchainConfigProxy)
		require.Equal(t, superchainOutput.SuperchainConfigImpl, output.SuperchainConfigImpl)
		require.Equal(t, superchainOutput.SuperchainProxyAdmin, output.SuperchainProxyAdmin)
		require.NotEqual(t, common.Address{}, output.Guardian)
		require.NotEqual(t, common.Address{}, output.SuperchainProxyAdminOwner)

		// For OPCM v2, ProtocolVersions fields should be zero
		require.Equal(t, common.Address{}, output.ProtocolVersionsProxy)
		require.Equal(t, common.Address{}, output.ProtocolVersionsImpl)
		require.Equal(t, common.Address{}, output.ProtocolVersionsOwner)
	})
}
