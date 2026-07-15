package sysgo

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params/forks"

	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func TestLocalContractSourcesLocator(t *testing.T) {
	t.Parallel()

	t.Run("valid dir", func(t *testing.T) {
		t.Parallel()

		artifactsDir := filepath.Join(t.TempDir(), "forge-artifacts")
		require.NoError(t, os.Mkdir(artifactsDir, 0o755))

		loc, err := localContractSourcesLocator(artifactsDir)
		require.NoError(t, err)

		require.True(t, artifacts.MustNewFileLocator(artifactsDir).Equal(loc))
	})

	t.Run("missing dir", func(t *testing.T) {
		t.Parallel()

		_, err := localContractSourcesLocator(filepath.Join(t.TempDir(), "missing"))
		require.Error(t, err)
	})
}

func TestLocalContractSourcesLocatorAcceptsContractsBedrockRoot(t *testing.T) {
	t.Parallel()

	contractsDir := filepath.Join(t.TempDir(), "packages", "contracts-bedrock")
	require.NoError(t, os.MkdirAll(filepath.Join(contractsDir, "forge-artifacts"), 0o755))

	loc, err := localContractSourcesLocator(contractsDir)
	require.NoError(t, err)

	require.True(t, artifacts.MustNewFileLocator(contractsDir).Equal(loc))
}

func TestWithLocalContractSourcesAt(t *testing.T) {
	t.Parallel()

	artifactsDir := filepath.Join(t.TempDir(), "forge-artifacts")
	require.NoError(t, os.Mkdir(artifactsDir, 0o755))

	builder := newValidIntentBuilder()
	dt := devtest.SerialT(t)

	WithLocalContractSourcesAt(artifactsDir)(dt, nil, builder)

	intent, err := builder.Build()
	require.NoError(t, err)

	expected := artifacts.MustNewFileLocator(artifactsDir)
	require.True(t, expected.Equal(intent.L1ContractsLocator))
	require.True(t, expected.Equal(intent.L2ContractsLocator))
}

func TestParseL1Fork(t *testing.T) {
	tests := map[string]forks.Fork{
		"dencun":      forks.Cancun,
		"CANCUN":      forks.Cancun,
		"pectra":      forks.Prague,
		"prague":      forks.Prague,
		"fusaka":      forks.Osaka,
		"Osaka":       forks.Osaka,
		"bpo-1":       forks.BPO1,
		"BPO_5":       forks.BPO5,
		"glamsterdam": forks.Amsterdam,
		" amsterdam ": forks.Amsterdam,
	}
	for input, expected := range tests {
		t.Run(input, func(t *testing.T) {
			actual, err := parseL1Fork(input)
			require.NoError(t, err)
			require.Equal(t, expected, actual)
		})
	}

	_, err := parseL1Fork("frontier")
	require.EqualError(t, err, `unsupported L1 fork "frontier"`)
}

func TestDeployerOptionsOverrideDevstackL1Fork(t *testing.T) {
	t.Setenv(DevstackL1ForkEnvVar, "glamsterdam")
	builder := newValidIntentBuilder()
	builder.WithL1ContractsLocator(artifacts.EmbeddedLocator)
	builder.WithL2ContractsLocator(artifacts.EmbeddedLocator)
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)
	dt := devtest.SerialT(t)

	applyConfigCommons(dt, keys, DefaultL1ID, builder)
	applyConfigDeployerOptions(dt, keys, builder, []DeployerOption{
		WithForkAtL1Genesis(forks.Osaka),
		WithForkAtL1Offset(forks.BPO1, 1),
	})

	intent, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, intent.L1DevGenesisParams.PragueTimeOffset)
	require.Zero(t, *intent.L1DevGenesisParams.PragueTimeOffset)
	require.NotNil(t, intent.L1DevGenesisParams.OsakaTimeOffset)
	require.Zero(t, *intent.L1DevGenesisParams.OsakaTimeOffset)
	require.NotNil(t, intent.L1DevGenesisParams.BPO1TimeOffset)
	require.Equal(t, uint64(1), *intent.L1DevGenesisParams.BPO1TimeOffset)
	require.Nil(t, intent.L1DevGenesisParams.BPO2TimeOffset)
	require.Nil(t, intent.L1DevGenesisParams.AmsterdamTimeOffset)
}

func TestDevstackFutureL1ForkAddsBlobSchedule(t *testing.T) {
	t.Setenv(DevstackL1ForkEnvVar, "glamsterdam")
	builder := newValidIntentBuilder()
	builder.WithL1ContractsLocator(artifacts.EmbeddedLocator)
	builder.WithL2ContractsLocator(artifacts.EmbeddedLocator)
	keys, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
	require.NoError(t, err)

	applyConfigCommons(devtest.SerialT(t), keys, DefaultL1ID, builder)

	intent, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, intent.L1DevGenesisParams.AmsterdamTimeOffset)
	require.Zero(t, *intent.L1DevGenesisParams.AmsterdamTimeOffset)
	schedule := intent.L1DevGenesisParams.BlobSchedule
	require.NotNil(t, schedule)
	require.NotNil(t, schedule.BPO1)
	require.NotNil(t, schedule.BPO2)
	require.NotNil(t, schedule.BPO3)
	require.NotNil(t, schedule.BPO4)
	require.NotNil(t, schedule.BPO5)
	require.NotNil(t, schedule.Amsterdam)
}

func newValidIntentBuilder() intentbuilder.Builder {
	builder := intentbuilder.New()

	_, superchain := builder.WithSuperchain()
	superchain.WithProxyAdminOwner(common.HexToAddress("0x1"))
	superchain.WithGuardian(common.HexToAddress("0x2"))
	superchain.WithChallenger(common.HexToAddress("0x4"))

	builder, _ = builder.WithL1(eth.ChainIDFromUInt64(1))
	_, l2 := builder.WithL2(eth.ChainIDFromUInt64(10))
	l2.WithBaseFeeVaultRecipient(common.HexToAddress("0x5"))
	l2.WithSequencerFeeVaultRecipient(common.HexToAddress("0x6"))
	l2.WithL1FeeVaultRecipient(common.HexToAddress("0x7"))
	l2.WithOperatorFeeVaultRecipient(common.HexToAddress("0x8"))
	l2.WithL1ProxyAdminOwner(common.HexToAddress("0x9"))
	l2.WithL2ProxyAdminOwner(common.HexToAddress("0xa"))
	l2.WithSystemConfigOwner(common.HexToAddress("0xb"))
	l2.WithUnsafeBlockSigner(common.HexToAddress("0xc"))
	l2.WithBatcher(common.HexToAddress("0xd"))
	l2.WithProposer(common.HexToAddress("0xe"))
	l2.WithChallenger(common.HexToAddress("0xf"))

	return builder
}
