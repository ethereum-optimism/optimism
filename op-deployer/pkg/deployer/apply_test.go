package deployer

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log/slog"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

// The refusal must come before any stage runs: the L1 RPC URL is unreachable, so reaching the
// pipeline would fail with a different error.
func TestApplyPipelineRejectsMockSP1VerifierOnReleaseL1(t *testing.T) {
	for _, target := range []DeploymentTarget{DeploymentTargetLive, DeploymentTargetGenesis} {
		for _, l1ChainID := range []uint64{1, 11155111} {
			err := ApplyPipeline(context.Background(), ApplyPipelineOpts{
				DeploymentTarget:      target,
				DeployMockSP1Verifier: true,
				L1RPCUrl:              "http://127.0.0.1:1",
				Intent:                &state.Intent{L1ChainID: l1ChainID},
				State:                 &state.State{},
			})
			require.ErrorContains(t, err, fmt.Sprintf("L1 chain ID %d has a release SP1 verifier", l1ChainID), "target %s", target)
		}
	}
}

// A genesis apply removes the deployer's account from the L1 genesis, so it must refuse any L1
// prefund entry for the deployer before a stage runs. Every intent here is otherwise invalid, so an
// apply that passes the check fails on the missing L1 chain ID instead.
func TestApplyPipelineRejectsL1DeployerPrefund(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	deployer := crypto.PubkeyToAddress(key.PublicKey)
	prefund := func(addr common.Address, amount uint64) *state.Intent {
		return &state.Intent{L1DevGenesisParams: &state.L1DevGenesisParams{
			Prefund: map[common.Address]*hexutil.U256{addr: (*hexutil.U256)(uint256.NewInt(amount))},
		}}
	}
	const refused, invalid = "l1DevGenesisParams.prefund has an entry for the deployer", "l1ChainID cannot be 0"
	for _, tt := range []struct {
		name    string
		target  DeploymentTarget
		key     *ecdsa.PrivateKey
		intent  *state.Intent
		wantErr string
	}{
		{"deployer prefunded", DeploymentTargetGenesis, key, prefund(deployer, 1), refused},
		{"zero deployer prefund", DeploymentTargetGenesis, key, prefund(deployer, 0), refused},
		{"placeholder deployer prefunded", DeploymentTargetGenesis, nil, prefund(standard.PlaceholderAddress, 1), refused},
		{"other account prefunded", DeploymentTargetGenesis, key, prefund(common.Address{0xaa}, 1), invalid},
		{"no L1 dev genesis params", DeploymentTargetGenesis, key, &state.Intent{}, invalid},
		{"live target", DeploymentTargetLive, key, prefund(deployer, 1), invalid},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ApplyPipeline(t.Context(), ApplyPipelineOpts{
				DeploymentTarget:   tt.target,
				DeployerPrivateKey: tt.key,
				Intent:             tt.intent,
				State:              &state.State{},
			})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestNewPipelineEnvForgeCacheProfile checks that the forge clients run only a default-profile build,
// even where contracts may exceed the code size limit. Only a GenesisState run, which deploys to no
// chain, may run a build of another profile.
func TestNewPipelineEnvForgeCacheProfile(t *testing.T) {
	_, afs := testutil.LocalArtifacts(t)
	bundle := artifacts.Bundle{L1: afs, L2: afs}
	for _, target := range []DeploymentTarget{DeploymentTargetLive, DeploymentTargetGenesis} {
		t.Run(string(target), func(t *testing.T) {
			pEnv, err := newPipelineEnv(t.Context(), ApplyPipelineOpts{
				DeploymentTarget:          target,
				L1RPCUrl:                  "http://127.0.0.1:1",
				Logger:                    testlog.Logger(t, slog.LevelInfo),
				CacheDir:                  t.TempDir(),
				AllowUnoptimizedContracts: true,
				Intent:                    &state.Intent{L1ChainID: 900},
			}, bundle)
			require.NoError(t, err)
			require.False(t, pEnv.L2Forge.AnyCacheProfile)
			require.False(t, pEnv.Forge.Client.AnyCacheProfile, "the L1 client runs live deployments")
			if target == DeploymentTargetLive {
				require.Nil(t, pEnv.Forge.Genesis, "a live deployment must not run offline")
			} else {
				require.True(t, pEnv.Forge.Genesis.NoCodeSizeLimit)
				require.Equal(t, uint64(900), pEnv.Forge.Genesis.ChainID, "the L1 genesis scripts must see the L1 chain ID")
			}
		})
	}
}
