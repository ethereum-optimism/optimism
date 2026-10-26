package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/devfeatures"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/standard"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
)

// TestGenesisForgeMatchesGoHost runs genesis deployments on the Go script host and on forge, and
// requires the same L1 state and L2 genesis from both.
func TestGenesisForgeMatchesGoHost(t *testing.T) {
	zkGame := state.AdditionalDisputeGame{
		ChainProofParams: state.ChainProofParams{DisputeGameType: uint32(opcm.GameTypeZKDisputeGame)},
		VMType:           state.VMTypeZK,
		ZKDisputeGame: &state.ZKDisputeGameParams{
			AbsolutePrestate:     common.Hash{0x0a},
			MaxChallengeDuration: 3600,
			MaxProveDuration:     3600,
			ChallengerBond:       (*hexutil.Big)(big.NewInt(1e18)),
		},
	}
	tests := []struct {
		name   string
		mutate func(t *testing.T, intent *state.Intent, opts *deployer.ApplyPipelineOpts)
	}{
		{name: "standard"},
		{
			name: "interop with two chains",
			mutate: func(t *testing.T, intent *state.Intent, _ *deployer.ApplyPipelineOpts) {
				dk, err := devkeys.NewMnemonicDevKeys(devkeys.TestMnemonic)
				require.NoError(t, err)
				intent.Chains = append(intent.Chains, shared.NewChainIntent(t, dk, intent.L1ChainIDBig(), uint256.NewInt(2), testCustomGasLimit))
				intent.UseInterop = true
				intent.GlobalDeployOverrides = map[string]any{"devFeatureBitmap": devfeatures.OptimismPortalInteropFlag}
			},
		},
		{
			name: "zk with mock verifier and additional games",
			mutate: func(t *testing.T, intent *state.Intent, opts *deployer.ApplyPipelineOpts) {
				deployerAddr := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
				intent.Chains[0].Roles.L1ProxyAdminOwner = deployerAddr
				intent.SuperchainRoles.SuperchainGuardian = deployerAddr
				intent.GlobalDeployOverrides = map[string]any{
					"devFeatureBitmap":              devfeatures.ZKDisputeGameFlag,
					"preimageOracleChallengePeriod": 1,
				}
				opts.DeployMockSP1Verifier = true
				custom := state.ChainProofParams{
					DisputeAbsolutePrestate:                 standard.DisputeAbsolutePrestate,
					DisputeMaxGameDepth:                     50,
					DisputeSplitDepth:                       14,
					DisputeClockExtension:                   1,
					DisputeMaxClockDuration:                 10,
					DangerouslyAllowCustomDisputeParameters: true,
				}
				alphabet, cannon := custom, custom
				alphabet.DisputeGameType = 255
				cannon.DisputeGameType = 254
				intent.Chains[0].AdditionalDisputeGames = []state.AdditionalDisputeGame{
					{ChainProofParams: alphabet, VMType: state.VMTypeAlphabet, MakeRespected: true},
					{ChainProofParams: cannon, VMType: state.VMTypeCannon},
					zkGame,
				}
			},
		},
		{
			name: "alt-da",
			mutate: func(t *testing.T, intent *state.Intent, _ *deployer.ApplyPipelineOpts) {
				intent.Chains[0].DangerousAltDAConfig = genesis.AltDADeployConfig{
					UseAltDA:                   true,
					DACommitmentType:           altda.KeccakCommitmentString,
					DAChallengeWindow:          10,
					DAResolveWindow:            10,
					DABondSize:                 100,
					DAResolverRefundPercentage: 50,
				}
			},
		},
		{
			name: "custom gas token",
			mutate: func(t *testing.T, intent *state.Intent, _ *deployer.ApplyPipelineOpts) {
				enableCustomGasToken(t, intent)
			},
		},
		{
			name: "funded dev accounts and governance",
			mutate: func(t *testing.T, intent *state.Intent, _ *deployer.ApplyPipelineOpts) {
				intent.FundDevAccounts = true
				intent.Chains[0].DeployOverrides = map[string]any{
					"enableGovernance":     true,
					"governanceTokenOwner": common.Address{0x60},
				}
			},
		},
		{
			name: "fee vault overrides",
			mutate: func(t *testing.T, intent *state.Intent, _ *deployer.ApplyPipelineOpts) {
				intent.GlobalDeployOverrides = map[string]any{
					"baseFeeVaultWithdrawalNetwork":           "remote",
					"sequencerFeeVaultWithdrawalNetwork":      "remote",
					"l1FeeVaultMinimumWithdrawalAmount":       "0x1bc16d674ec80000",
					"operatorFeeVaultMinimumWithdrawalAmount": "0x29a2241af62c0000",
				}
			},
		},
		{
			name: "prefunded deployer",
			mutate: func(t *testing.T, intent *state.Intent, opts *deployer.ApplyPipelineOpts) {
				deployerAddr := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
				intent.L1DevGenesisParams.Prefund = map[common.Address]*hexutil.U256{
					deployerAddr:      (*hexutil.U256)(uint256.NewInt(1e18)),
					common.Address{1}: (*hexutil.U256)(uint256.NewInt(2e18)),
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goHost := applyGenesis(t, true, tt.mutate)
			forge := applyGenesis(t, false, tt.mutate)
			requireEqualAllocs(t, goHost.L1StateDump.Data.Accounts, forge.L1StateDump.Data.Accounts)
			require.Equal(t, goHost.L1DevGenesis.ToBlock().Root(), forge.L1DevGenesis.ToBlock().Root())
			require.Len(t, forge.Chains, len(goHost.Chains))
			for i, chain := range goHost.Chains {
				requireEqualAllocs(t, chain.Allocs.Data.Accounts, forge.Chains[i].Allocs.Data.Accounts)
				goHostGenesis, _, err := pipeline.RenderGenesisAndRollup(goHost, chain.ID, nil)
				require.NoError(t, err)
				forgeGenesis, _, err := pipeline.RenderGenesisAndRollup(forge, chain.ID, nil)
				require.NoError(t, err)
				require.Equal(t, goHostGenesis.ToBlock().Hash(), forgeGenesis.ToBlock().Hash())
			}
		})
	}
}

// TestL1DevGenesisForgeMatchesGoHost does the same for the L1 dev genesis that sysgo builds.
func TestL1DevGenesisForgeMatchesGoHost(t *testing.T) {
	applyL1DevGenesis := func(useScriptHost bool) *state.State {
		opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
		deployerAddr := crypto.PubkeyToAddress(opts.DeployerPrivateKey.PublicKey)
		intent.L1DevGenesisParams = &state.L1DevGenesisParams{
			BlockParams: state.L1DevGenesisBlockParams{Timestamp: 1000},
			Prefund:     map[common.Address]*hexutil.U256{deployerAddr: (*hexutil.U256)(uint256.NewInt(1e18))},
		}
		intent.GlobalDeployOverrides = map[string]any{"devFeatureBitmap": devfeatures.ZKDisputeGameFlag}
		opts.UseScriptHost = useScriptHost
		opts.DeployMockSP1Verifier = true
		opts.AllowUnoptimizedContracts = true
		require.NoError(t, deployer.ApplyL1DevGenesis(context.Background(), opts))
		return st
	}
	goHost := applyL1DevGenesis(true)
	forge := applyL1DevGenesis(false)
	requireEqualAllocs(t, goHost.L1StateDump.Data.Accounts, forge.L1StateDump.Data.Accounts)
	require.Equal(t, goHost.L1DevGenesis.ToBlock().Root(), forge.L1DevGenesis.ToBlock().Root())
}

func applyGenesis(t *testing.T, useScriptHost bool, mutate func(*testing.T, *state.Intent, *deployer.ApplyPipelineOpts)) *state.State {
	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	intent.L1DevGenesisParams = &state.L1DevGenesisParams{BlockParams: state.L1DevGenesisBlockParams{Timestamp: 1000}}
	opts.UseScriptHost = useScriptHost
	// A fixed salt, so that both runs deploy the chain contracts to the same addresses.
	st.Create2Salt = common.Hash{0x5a}
	if mutate != nil {
		mutate(t, intent, &opts)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	require.NoError(t, deployer.ApplyPipeline(ctx, opts))
	return st
}

// requireEqualAllocs fails with every account that differs, and how.
func requireEqualAllocs(t *testing.T, want, got types.GenesisAlloc) {
	t.Helper()
	addrs := make(map[common.Address]struct{})
	for addr := range want {
		addrs[addr] = struct{}{}
	}
	for addr := range got {
		addrs[addr] = struct{}{}
	}
	sorted := make([]common.Address, 0, len(addrs))
	for addr := range addrs {
		sorted = append(sorted, addr)
	}
	slices.SortFunc(sorted, func(a, b common.Address) int { return a.Cmp(b) })

	var diffs []string
	for _, addr := range sorted {
		w, inWant := want[addr]
		g, inGot := got[addr]
		switch {
		case !inGot:
			diffs = append(diffs, fmt.Sprintf("%s: only on the Go host (nonce %d, balance %s, %d code bytes, %d slots)", addr, w.Nonce, w.Balance, len(w.Code), len(w.Storage)))
			continue
		case !inWant:
			diffs = append(diffs, fmt.Sprintf("%s: only on forge (nonce %d, balance %s, %d code bytes, %d slots)", addr, g.Nonce, g.Balance, len(g.Code), len(g.Storage)))
			continue
		}
		if w.Nonce != g.Nonce {
			diffs = append(diffs, fmt.Sprintf("%s: nonce %d on the Go host, %d on forge", addr, w.Nonce, g.Nonce))
		}
		if w.Balance.Cmp(g.Balance) != 0 {
			diffs = append(diffs, fmt.Sprintf("%s: balance %s on the Go host, %s on forge", addr, w.Balance, g.Balance))
		}
		if !bytes.Equal(w.Code, g.Code) {
			diffs = append(diffs, fmt.Sprintf("%s: %d code bytes on the Go host, %d on forge", addr, len(w.Code), len(g.Code)))
		}
		for slot, value := range w.Storage {
			if g.Storage[slot] != value {
				diffs = append(diffs, fmt.Sprintf("%s: slot %s is %s on the Go host, %s on forge", addr, slot, value, g.Storage[slot]))
			}
		}
		for slot, value := range g.Storage {
			if _, ok := w.Storage[slot]; !ok {
				diffs = append(diffs, fmt.Sprintf("%s: slot %s is only on forge, %s", addr, slot, value))
			}
		}
	}
	require.Empty(t, diffs, "allocs differ:\n%s", strings.Join(diffs, "\n"))
}
