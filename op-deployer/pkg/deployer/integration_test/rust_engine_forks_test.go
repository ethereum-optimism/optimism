package integration_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/integration_test/shared"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/pipeline"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/testutil"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/env"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-service/testutils"
)

// e2eGenesisForks is the exact set of L2 genesis forks op-e2e's config.initAllocType builds allocs
// for (op-e2e/config/init.go). Flipping the default script engine to rust routes every one of these
// through the Rust engine, so byte-parity must hold across all of them, not just the latest fork the
// feature-mode parity tests cover.
var e2eGenesisForks = []genesis.L2AllocsMode{
	genesis.L2AllocsLagoon,
	genesis.L2AllocsKarst,
	genesis.L2AllocsJovian,
	genesis.L2AllocsIsthmus,
	genesis.L2AllocsHolocene,
	genesis.L2AllocsGranite,
	genesis.L2AllocsFjord,
	genesis.L2AllocsEcotone,
	genesis.L2AllocsDelta,
}

// forkOverrides reproduces op-e2e/config.initAllocType's per-fork upgrade schedule: reset every
// hardfork time offset to nil, then activate the target fork at genesis. Returned as the map op-deployer
// merges into intent.GlobalDeployOverrides.
func forkOverrides(t *testing.T, mode genesis.L2AllocsMode) map[string]any {
	t.Helper()
	base := map[string]any{
		"l2GenesisRegolithTimeOffset": nil,
		"l2GenesisCanyonTimeOffset":   nil,
		"l2GenesisDeltaTimeOffset":    nil,
		"l2GenesisEcotoneTimeOffset":  nil,
		"l2GenesisFjordTimeOffset":    nil,
		"l2GenesisGraniteTimeOffset":  nil,
		"l2GenesisHoloceneTimeOffset": nil,
		"l2GenesisIsthmusTimeOffset":  nil,
		"l2GenesisJovianTimeOffset":   nil,
	}
	schedule := new(genesis.UpgradeScheduleDeployConfig)
	schedule.ActivateForkAtGenesis(forks.Name(mode))
	b, err := json.Marshal(schedule)
	require.NoError(t, err)
	var scheduleMap map[string]any
	require.NoError(t, json.Unmarshal(b, &scheduleMap))
	for k, v := range scheduleMap {
		base[k] = v
	}
	return base
}

// TestApplyPipelineRustEngineGenesisForks runs the real genesis pipeline once per op-e2e genesis fork
// through both engines and asserts byte-identical L2 allocs. This is the blast-radius de-risk for
// flipping the default engine to rust: op-e2e/op-devstack build L2 genesis at each of these forks, so
// the Rust engine must match the Go host across the whole fork range, not only the latest fork.
func TestApplyPipelineRustEngineGenesisForks(t *testing.T) {
	bin := buildRustScriptEngine(t)
	t.Setenv("RUST_BINARY_PATH_OP_SCRIPT_ENGINE", bin)

	runPipeline := func(t *testing.T, mode genesis.L2AllocsMode, engine env.ScriptEngineKind) *foundry.ForgeAllocs {
		t.Helper()
		lgr := testlog.Logger(t, slog.LevelWarn)
		_, pk, dk := shared.DefaultPrivkey(t)
		l1ChainID := big.NewInt(900)
		l2ChainID := uint256.NewInt(1)
		loc, _ := testutil.LocalArtifacts(t)
		intent, st := shared.NewIntent(t, l1ChainID, dk, l2ChainID, loc, loc, testCustomGasLimit)
		intent.GlobalDeployOverrides = forkOverrides(t, mode)
		// Pin the CREATE2 salt so both engine runs get identical L1 deploy addresses (see
		// TestApplyPipelineRustEngineGenesis); only then is the L2Genesis output comparable across runs.
		st.Create2Salt = common.HexToHash("0x00000000000000000000000000000000000000000000000000000000deadbeef")

		require.NoError(t, deployer.ApplyPipeline(context.Background(), deployer.ApplyPipelineOpts{
			DeploymentTarget:   deployer.DeploymentTargetGenesis,
			L1RPCUrl:           "",
			DeployerPrivateKey: pk,
			Intent:             intent,
			State:              st,
			Logger:             lgr,
			StateWriter:        pipeline.NoopStateWriter(),
			CacheDir:           testutils.IsolatedTestDirWithAutoCleanup(t),
			ScriptEngine:       engine,
		}))
		require.NotEmpty(t, st.Chains)
		require.NotNil(t, st.Chains[0].Allocs)
		return st.Chains[0].Allocs.Data
	}

	for _, mode := range e2eGenesisForks {
		mode := mode
		t.Run(string(mode), func(t *testing.T) {
			goDump := runPipeline(t, mode, env.ScriptEngineGo)
			rustDump := runPipeline(t, mode, env.ScriptEngineRust)

			require.Equal(t, len(goDump.Accounts), len(rustDump.Accounts), "rust dump account count")
			t.Logf("[%s] L2Genesis fork parity: %d accounts, byte-identical (go vs rust)", mode, len(rustDump.Accounts))
			requireAllocsJSONEq(t, "fork-"+string(mode), goDump, rustDump)
		})
	}
}
