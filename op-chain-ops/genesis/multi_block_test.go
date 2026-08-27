package genesis

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
)

// multiBlockRollupFixture is the checked-in rollup config that kona's rollup config parse test
// reads, so both clients agree on the JSON names and shape of the multi-blocks keys.
const multiBlockRollupFixture = "testdata/rollup-multiblock.json"

func multiBlockDeployConfig(t *testing.T) *DeployConfig {
	t.Helper()
	b, err := os.ReadFile("testdata/test-deploy-config-full.json")
	require.NoError(t, err)
	cfg := new(DeployConfig)
	require.NoError(t, json.NewDecoder(bytes.NewReader(b)).Decode(cfg))

	cfg.ActivateForkAtGenesis(forks.Lagoon)
	cfg.L2GenesisMultiBlockTimeOffset = (*hexutil.Uint64)(ptr.New(uint64(20)))
	cfg.MaxMultiBlocks = (*hexutil.Uint64)(ptr.New(uint64(16)))
	// Lagoon is moved past the activation so the fixture also pins the rule that siblings are
	// never allowed on a fork activation timestamp.
	cfg.L2GenesisLagoonTimeOffset = (*hexutil.Uint64)(ptr.New(uint64(40)))
	return cfg
}

func TestMultiBlockTimeOffset(t *testing.T) {
	cfg := multiBlockDeployConfig(t)
	require.Equal(t, uint64(1234+20), *cfg.MultiBlockTime(1234))

	cfg.L2GenesisMultiBlockTimeOffset = nil
	require.Nil(t, cfg.MultiBlockTime(1234))
}

// TestRollupConfigMultiBlockFixture pins the rollup.json that DeployConfig produces for a
// multi-blocks chain. kona parses the same file, so the two clients cannot drift on the key names,
// nor on siblings_allowed being false at the Lagoon activation the fixture schedules after the
// multi-blocks activation.
// Set UPDATE_GOLDEN=1 to regenerate it after an intentional change.
func TestRollupConfigMultiBlockFixture(t *testing.T) {
	l1Start := &eth.BlockRef{Hash: common.HexToHash("0xaa"), Number: 100, Time: 5000}
	rollupCfg, err := multiBlockDeployConfig(t).RollupConfig(l1Start, common.HexToHash("0xbb"), 0)
	require.NoError(t, err)
	require.NoError(t, rollupCfg.Check())
	require.Equal(t, uint64(5020), *rollupCfg.MultiBlockTime)
	require.Equal(t, uint64(16), *rollupCfg.MaxMultiBlocks)
	require.Equal(t, uint64(5040), *rollupCfg.LagoonTime)
	require.False(t, rollupCfg.SiblingsAllowed(*rollupCfg.LagoonTime))
	require.True(t, rollupCfg.SiblingsAllowed(*rollupCfg.LagoonTime+rollupCfg.BlockTime))

	got, err := json.MarshalIndent(rollupCfg, "", "  ")
	require.NoError(t, err)
	got = append(got, '\n')

	if os.Getenv("UPDATE_GOLDEN") != "" {
		require.NoError(t, os.WriteFile(multiBlockRollupFixture, got, 0o644))
	}
	want, err := os.ReadFile(multiBlockRollupFixture)
	require.NoError(t, err)
	require.Equal(t, string(want), string(got))
}
