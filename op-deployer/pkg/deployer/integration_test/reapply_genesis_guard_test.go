package integration_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/state"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils/devnet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"log/slog"
)

// --- In-process L1 mock so the REAL InitLiveStrategy guard is reached --------------------
//
// pipeline.Env.L1Client is a concrete *ethclient.Client. We back one with an in-process
// JSON-RPC server (no external process) that answers just the two calls the guard makes
// before the immutability comparison: eth_chainId (ethclient.ChainID) and eth_getCode
// (ethclient.CodeAt, for the deterministic-deployer presence check).

type guardMockEthService struct{ chainID uint64 }

func (m *guardMockEthService) ChainId() (*hexutil.Big, error) {
	return (*hexutil.Big)(new(big.Int).SetUint64(m.chainID)), nil
}

func (m *guardMockEthService) GetCode(_ common.Address, _ string) (hexutil.Bytes, error) {
	return hexutil.Bytes{0x60, 0x00}, nil // non-empty so the deployer check passes
}

func newGuardMockL1Client(t *testing.T, chainID uint64) *ethclient.Client {
	t.Helper()
	srv := rpc.NewServer()
	require.NoError(t, srv.RegisterName("eth", &guardMockEthService{chainID: chainID}))
	t.Cleanup(srv.Stop)
	client := ethclient.NewClient(rpc.DialInProc(srv))
	t.Cleanup(client.Close)
	// sanity: confirm the mock answers the calls the guard relies on.
	gotID, err := client.ChainID(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, chainID, gotID.Uint64())
	code, err := client.CodeAt(context.Background(), script.DeterministicDeployerAddress, nil)
	require.NoError(t, err)
	require.NotEmpty(t, code)
	return client
}

// applyGenesisChainForGuard produces a REAL, fully-applied state (allocs + start block) via
// the in-memory genesis strategy, so the immutability guard's RenderGenesisAndRollup calls
// have everything they need. Returns the applied intent (a deep-ish copy is the caller's
// job via mutate funcs) and state.
func applyGenesisChainForGuard(t *testing.T) (*state.Intent, *state.State) {
	t.Helper()
	ctx := context.Background()
	opts, intent, st := setupGenesisChain(t, devnet.DefaultChainID)
	require.NoError(t, deployer.ApplyPipeline(ctx, opts))
	require.NotNil(t, st.AppliedIntent, "apply must record AppliedIntent")
	require.NotEmpty(t, st.Chains)
	require.NotNil(t, st.Chains[0].Allocs, "applied chain must have frozen allocs")
	require.NotNil(t, st.Chains[0].StartBlock, "applied chain must have a start block")
	return intent, st
}

// cloneIntentForReapply returns a copy of the applied intent suitable for re-apply, with a
// freshly copied first ChainIntent so the caller can mutate genesis-affecting fields without
// touching st.AppliedIntent. (Other fields are shared; tests only mutate the first chain or
// top-level scalar/override fields, which are reassigned wholesale.)
func cloneIntentForReapply(applied *state.Intent) *state.Intent {
	cp := *applied
	cp.Chains = make([]*state.ChainIntent, len(applied.Chains))
	for i, c := range applied.Chains {
		cc := *c
		cp.Chains[i] = &cc
	}
	if applied.GlobalDeployOverrides != nil {
		cp.GlobalDeployOverrides = make(map[string]any, len(applied.GlobalDeployOverrides))
		for k, v := range applied.GlobalDeployOverrides {
			cp.GlobalDeployOverrides[k] = v
		}
	}
	return &cp
}

func runGuard(t *testing.T, st *state.State, newIntent *state.Intent) error {
	t.Helper()
	env := &pipeline.Env{
		L1Client: newGuardMockL1Client(t, newIntent.L1ChainID),
		Logger:   testlog.Logger(t, slog.LevelError),
	}
	return pipeline.InitLiveStrategy(context.Background(), env, newIntent, st)
}

// TestReapplyGenesisGuard_RejectsGenesisAffectingChanges drives the REAL guard
// (pipeline.InitLiveStrategy) against a fully-applied genesis-strategy state and asserts it
// REJECTS each genesis-affecting re-apply mutation, ACCEPTS legitimate re-applies, and
// verifies the comparison is deterministic (old-vs-old never falsely triggers).
func TestReapplyGenesisGuard_RejectsGenesisAffectingChanges(t *testing.T) {
	op_e2e.InitParallel(t)

	applied, st := applyGenesisChainForGuard(t)
	chain0 := applied.Chains[0]

	// --- CONTROL 1: an UNCHANGED re-apply must PASS (no false positive, determinism). ---
	{
		unchanged := cloneIntentForReapply(applied)
		require.NoError(t, runGuard(t, st, unchanged),
			"unchanged re-apply must pass (proves RenderGenesisAndRollup is deterministic)")
	}

	// --- CONTROL 2: the pre-existing immutable check still works (guard executed). ---
	{
		fundFlip := cloneIntentForReapply(applied)
		fundFlip.FundDevAccounts = !applied.FundDevAccounts
		err := runGuard(t, st, fundFlip)
		require.Error(t, err)
		require.Contains(t, err.Error(), "fundDevAccounts is immutable",
			"control: confirms the real guard ran")
	}

	// --- REJECTIONS: genesis-affecting field mutations. ---
	// "expect" is the substring of the immutability error that names the changed surface:
	//   "genesis system config" -> rollup Genesis.SystemConfig differs (batcher, operatorFee, gasLimit)
	//   "genesis block"         -> the genesis block hash differs (header field changed)
	rejectCases := []struct {
		name   string
		mutate func(in *state.Intent)
		expect string
	}{
		{
			name:   "roles.batcher",
			mutate: func(in *state.Intent) { in.Chains[0].Roles.Batcher = common.HexToAddress("0xDEADBEEF") },
			expect: "genesis system config",
		},
		{
			name:   "operatorFeeScalar",
			mutate: func(in *state.Intent) { in.Chains[0].OperatorFeeScalar = chain0.OperatorFeeScalar + 7 },
			expect: "genesis system config",
		},
		{
			name:   "operatorFeeConstant",
			mutate: func(in *state.Intent) { in.Chains[0].OperatorFeeConstant = chain0.OperatorFeeConstant + 9 },
			expect: "genesis system config",
		},
		{
			// gasLimit feeds BOTH the genesis header (block hash) and the genesis SystemConfig.
			// The block-hash check runs first, so the error names the genesis block.
			name:   "gasLimit",
			mutate: func(in *state.Intent) { in.Chains[0].GasLimit = chain0.GasLimit + 1_000_000 },
			expect: "genesis block",
		},
		{
			// A genesis-header override (base fee per gas) changes the genesis block hash.
			name: "genesis-block GlobalDeployOverride (l2GenesisBlockBaseFeePerGas)",
			mutate: func(in *state.Intent) {
				if in.GlobalDeployOverrides == nil {
					in.GlobalDeployOverrides = map[string]any{}
				}
				in.GlobalDeployOverrides["l2GenesisBlockBaseFeePerGas"] = "0x3b9aca01" // 1e9+1
			},
			expect: "genesis block",
		},
		{
			// A genesis-SystemConfig override (operator fee scalar) changes the rollup GenesisSystemConfig.
			name: "genesis-sysconfig chain DeployOverride (gasPriceOracleOperatorFeeScalar)",
			mutate: func(in *state.Intent) {
				in.Chains[0].DeployOverrides = map[string]any{"gasPriceOracleOperatorFeeScalar": float64(4242)}
			},
			expect: "genesis system config",
		},
	}
	for _, tc := range rejectCases {
		t.Run("reject/"+tc.name, func(t *testing.T) {
			ni := cloneIntentForReapply(applied)
			tc.mutate(ni)
			err := runGuard(t, st, ni)
			require.Error(t, err, "guard must reject genesis-affecting change: %s", tc.name)
			require.Contains(t, err.Error(), "is immutable", "must be an immutability rejection")
			require.Contains(t, err.Error(), tc.expect, "rejection should name what changed")
			t.Logf("rejected %s: %v", tc.name, err)
		})
	}

	// --- ACCEPTANCES: legitimate re-applies that do NOT alter genesis. ---
	acceptCases := []struct {
		name   string
		mutate func(in *state.Intent)
	}{
		{
			name: "future hardfork schedule via GlobalDeployOverrides",
			mutate: func(in *state.Intent) {
				if in.GlobalDeployOverrides == nil {
					in.GlobalDeployOverrides = map[string]any{}
				}
				// Scheduling the next fork (Karst, the one immediately after the default
				// genesis-active Jovian) at a far-future offset changes only the genesis
				// ChainConfig fork time -- NOT the genesis block hash or genesis SystemConfig --
				// so the guard must allow it. (This is the canonical "legitimate re-apply".)
				in.GlobalDeployOverrides["l2GenesisKarstTimeOffset"] = "0x7fffffffffffffff"
			},
		},
		{
			name: "deploy-only proof param override (faultGameMaxDepth)",
			mutate: func(in *state.Intent) {
				if in.GlobalDeployOverrides == nil {
					in.GlobalDeployOverrides = map[string]any{}
				}
				in.GlobalDeployOverrides["faultGameMaxDepth"] = float64(73)
			},
		},
	}
	for _, tc := range acceptCases {
		t.Run("accept/"+tc.name, func(t *testing.T) {
			ni := cloneIntentForReapply(applied)
			tc.mutate(ni)
			require.NoError(t, runGuard(t, st, ni),
				"guard must ALLOW non-genesis-affecting change: %s", tc.name)
			t.Logf("accepted %s", tc.name)
		})
	}

	// --- ACCEPTANCE: adding a BRAND-NEW chain on re-apply must be allowed. ---
	t.Run("accept/brand-new-chain", func(t *testing.T) {
		ni := cloneIntentForReapply(applied)
		newChain := *applied.Chains[0]
		// Give it a distinct chain ID so it is genuinely a new, not-yet-applied chain.
		newChain.ID = common.BigToHash(uint256.NewInt(0xABCDEF).ToBig())
		newChain.Roles.Batcher = common.HexToAddress("0xC0FFEE") // arbitrary; new chain isn't frozen
		ni.Chains = append(ni.Chains, &newChain)
		require.NoError(t, runGuard(t, st, ni),
			"adding a brand-new chain on re-apply must be allowed")
	})
}

// TestReapplyGenesisGuard_DocumentedResidualGaps adversarially documents what the
// genesis-output comparison (genesis block hash + genesis SystemConfig) intentionally does
// NOT catch, so the guard's exact scope is pinned and honestly stated:
//
//   - Fields baked only into the FROZEN L2 allocs (e.g. fee-vault recipients stored in
//     predeploy storage) are not regenerated on re-apply, so old and new genesis OUTPUT are
//     identical -> not a re-apply divergence at all (true negative).
//   - Fields that live only in the genesis ChainConfig (e.g. eip1559Denominator) or only in
//     the broader rollup config (e.g. l2BlockTime) -- but NOT in the genesis block header or
//     the genesis SystemConfig -- are NOT covered by this comparison. These are residual gaps
//     of the chosen surface; they are deliberately out of scope here because the genesis
//     ChainConfig also legitimately changes for future-hardfork scheduling (which must be
//     allowed). Closing them would require a separate, ordering-aware comparison.
//
// If a future change makes any of these alter the genesis block hash or SystemConfig, this
// test will start failing and should be moved into the rejection set.
func TestReapplyGenesisGuard_DocumentedResidualGaps(t *testing.T) {
	op_e2e.InitParallel(t)

	applied, st := applyGenesisChainForGuard(t)

	notCaught := []struct {
		name   string
		mutate func(in *state.Intent)
	}{
		{
			name:   "baseFeeVaultRecipient (frozen in allocs; genesis output unchanged)",
			mutate: func(in *state.Intent) { in.Chains[0].BaseFeeVaultRecipient = common.HexToAddress("0xFEE5") },
		},
		{
			name:   "eip1559Denominator (genesis ChainConfig only; not in block hash or SystemConfig)",
			mutate: func(in *state.Intent) { in.Chains[0].Eip1559Denominator = in.Chains[0].Eip1559Denominator + 1 },
		},
		{
			name: "l2BlockTime (rollup config only; not in genesis block hash or SystemConfig)",
			mutate: func(in *state.Intent) {
				if in.GlobalDeployOverrides == nil {
					in.GlobalDeployOverrides = map[string]any{}
				}
				in.GlobalDeployOverrides["l2BlockTime"] = float64(3)
			},
		},
	}
	for _, tc := range notCaught {
		t.Run(tc.name, func(t *testing.T) {
			ni := cloneIntentForReapply(applied)
			tc.mutate(ni)
			err := runGuard(t, st, ni)
			require.NoError(t, err,
				"documented residual gap: this comparison does not catch %s", tc.name)
			t.Logf("NOT caught (documented residual gap): %s", tc.name)
		})
	}
}
