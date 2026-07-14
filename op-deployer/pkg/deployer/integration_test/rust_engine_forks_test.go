package integration_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-chain-ops/genesis"
	"github.com/ethereum-optimism/optimism/op-core/forks"
)

// e2eGenesisForks is the exact set of L2 genesis forks op-e2e's config.initAllocType builds allocs
// for (op-e2e/config/init.go). The default script engine (rust) routes every one of these through
// the Rust engine, so the fork golden gate pins every one, not just the latest fork.
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
