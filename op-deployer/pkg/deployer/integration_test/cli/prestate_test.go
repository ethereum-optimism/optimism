package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/upgrade/embedded"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPrestateCLICommandSources(t *testing.T) {
	const (
		selectedEnvVar = "DEPLOYER_DISPUTE_ABSOLUTE_PRESTATE"
		fallbackEnvVar = "DEPLOYER_CANNON_FALLBACK_PRESTATE"
	)

	tests := []struct {
		name         string
		envSelected  common.Hash
		envFallback  common.Hash
		cliSelected  common.Hash
		cliFallback  common.Hash
		wantSelected common.Hash
		wantFallback common.Hash
	}{
		{
			name:         "CLI values reach state",
			cliSelected:  common.HexToHash("0x11"),
			cliFallback:  common.HexToHash("0x22"),
			wantSelected: common.HexToHash("0x11"),
			wantFallback: common.HexToHash("0x22"),
		},
		{
			name:         "environment-only values reach state",
			envSelected:  common.HexToHash("0x33"),
			envFallback:  common.HexToHash("0x44"),
			wantSelected: common.HexToHash("0x33"),
			wantFallback: common.HexToHash("0x44"),
		},
		{
			name:         "CLI wins over environment",
			envSelected:  common.HexToHash("0x55"),
			envFallback:  common.HexToHash("0x66"),
			cliSelected:  common.HexToHash("0x77"),
			cliFallback:  common.HexToHash("0x88"),
			wantSelected: common.HexToHash("0x77"),
			wantFallback: common.HexToHash("0x88"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearTestEnv(t, selectedEnvVar)
			clearTestEnv(t, fallbackEnvVar)
			if tt.envSelected != (common.Hash{}) {
				t.Setenv(selectedEnvVar, tt.envSelected.Hex())
			}
			if tt.envFallback != (common.Hash{}) {
				t.Setenv(fallbackEnvVar, tt.envFallback.Hex())
			}

			workdir, chainID := writePreparedPrestateCLIWorkdir(t)
			args := []string{
				"op-deployer",
				"--cache-dir", filepath.Join(workdir, "cache"),
				"prestate",
				"--workdir", workdir,
			}
			if tt.cliSelected != (common.Hash{}) {
				args = append(args, "--dispute-absolute-prestate", tt.cliSelected.Hex())
			}
			if tt.cliFallback != (common.Hash{}) {
				args = append(args, "--cannon-fallback-prestate", tt.cliFallback.Hex())
			}

			var stdout bytes.Buffer
			var stderr bytes.Buffer
			err := RunCLI(context.Background(), &stdout, &stderr, args)
			require.NoError(t, err, stderr.String())

			persisted, err := pipeline.ReadState(workdir)
			require.NoError(t, err)
			chain, err := persisted.Chain(chainID)
			require.NoError(t, err)
			require.Equal(t, tt.wantSelected, chain.Prestate)
			require.Equal(t, tt.wantFallback, chain.CannonFallbackPrestate)
		})
	}
}

func writePreparedPrestateCLIWorkdir(t *testing.T) (string, common.Hash) {
	t.Helper()

	chainID := common.HexToHash("0x01")
	intent, err := state.NewIntentCustom(1, []common.Hash{chainID})
	require.NoError(t, err)
	role := common.HexToAddress("0x0000000000000000000000000000000000000001")
	intent.SuperchainRoles = &addresses.SuperchainRoles{
		SuperchainProxyAdminOwner: role,
		SuperchainGuardian:        role,
		Challenger:                role,
	}
	intent.GlobalDeployOverrides = make(map[string]any)
	chain := intent.Chains[0]
	chain.BaseFeeVaultRecipient = role
	chain.L1FeeVaultRecipient = role
	chain.SequencerFeeVaultRecipient = role
	chain.OperatorFeeVaultRecipient = role
	chain.Eip1559DenominatorCanyon = standard.Eip1559DenominatorCanyon
	chain.Eip1559Denominator = standard.Eip1559Denominator
	chain.Eip1559Elasticity = standard.Eip1559Elasticity
	chain.Roles = state.ChainRoles{
		L1ProxyAdminOwner: role,
		L2ProxyAdminOwner: role,
		SystemConfigOwner: role,
		UnsafeBlockSigner: role,
		Batcher:           role,
		Proposer:          role,
		Challenger:        role,
	}
	chain.DeployOverrides = map[string]any{"respectedGameType": embedded.GameTypeCannonKona}

	deployed := false
	st := &state.State{
		Version:  1,
		Prepared: true,
		Chains: []*state.ChainState{{
			ID:       chainID,
			Deployed: &deployed,
		}},
	}

	workdir := t.TempDir()
	require.NoError(t, intent.WriteToFile(filepath.Join(workdir, "intent.toml")))
	require.NoError(t, pipeline.WriteState(workdir, st))
	return workdir, chainID
}

func clearTestEnv(t *testing.T, key string) {
	t.Helper()

	value, set := os.LookupEnv(key)
	require.NoError(t, os.Unsetenv(key))
	t.Cleanup(func() {
		if set {
			require.NoError(t, os.Setenv(key, value))
		} else {
			require.NoError(t, os.Unsetenv(key))
		}
	})
}
